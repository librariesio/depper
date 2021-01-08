package publishers

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"os"
	"time"

	"github.com/go-redis/redis/v8"
	"github.com/librariesio/depper/data"
)

const TTL = 24 * time.Hour

type Sidekiq struct {
	RedisClient *redis.Client
	Context     context.Context
}

type LibrariesJob struct {
	Class      string   `json:"class"`
	Queue      string   `json:"queue"`
	Args       []string `json:"args"`
	Retry      bool     `json:"retry"`
	JID        string   `json:"jid"`
	CreatedAt  int64    `json:"created_at"`
	EnqueuedAt int64    `json:"enqueued_at"`
}

func NewSidekiq() *Sidekiq {
	address := "localhost:6379"
	envVal, envFound := os.LookupEnv("REDIS_CLOUD_URL")
	if envFound {
		address = envVal
	}
	rdb := redis.NewClient(&redis.Options{
		Addr:     address,
		Password: "",
		DB:       0,
	})

	return &Sidekiq{
		RedisClient: rdb,
		Context:     context.Background(),
	}
}
func getKey(packageVersion data.PackageVersion) string {
	return fmt.Sprintf("depper:ingest:%s:%s:%s", packageVersion.Platform, packageVersion.Name, packageVersion.Version)
}

func randomHex(n int) string {
	id := make([]byte, n)
	_, err := io.ReadFull(rand.Reader, id)
	if err != nil {
		log.Println("Error making random hex")
	}
	return hex.EncodeToString(id)
}

func createSyncJob(packageVersion data.PackageVersion) *LibrariesJob {
	return &LibrariesJob{
		Retry:      true,
		Class:      "PackageManagerDownloadWorker",
		Queue:      "critical",
		JID:        randomHex(12),
		EnqueuedAt: time.Now().Unix(),
		CreatedAt:  time.Now().Unix(),
		Args:       []string{packageVersion.Platform, packageVersion.Name},
	}
}

func (lib *Sidekiq) Publish(packageVersion data.PackageVersion) {
	key := getKey(packageVersion)

	wasSet, err := lib.RedisClient.SetNX(lib.Context, key, true, TTL).Result()
	if err != nil {
		log.Printf("Error trying to set key for redis %g", err)
		return
	}
	if wasSet {
		log.Printf("Sidekiq Publisher %s", key)
		lib.scheduleJob(packageVersion)
	}
}

func (lib *Sidekiq) scheduleJob(packageVersion data.PackageVersion) {
	job := createSyncJob(packageVersion)
	encoded, err := json.Marshal(job)
	if err != nil {
		log.Printf("Error encoding sync job for sidekiq %g", err)
		return
	}
	lib.RedisClient.LPush(lib.Context, fmt.Sprintf("queue:%s", job.Queue), string(encoded))
}
