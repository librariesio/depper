package publishers

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"github.com/librariesio/depper/data"
	"github.com/mediocregopher/radix/v3"
	"io"
	"log"
	"os"
	"time"
)

const TTL = 24 * time.Hour

type Sidekiq struct {
	RedisClient *radix.Pool
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
	envVal, envFound := os.LookupEnv("REDIS_URL")
	if envFound {
		address = envVal
	}
	client, err := radix.NewPool("tcp", address, 10)
	if err != nil {
		log.Fatalf("Error connecting to redis")
	}
	return &Sidekiq{
		RedisClient: client,
		Context:     context.Background(),
	}
}
func getKey(packageVersion data.PackageVersion) string {
	return fmt.Sprintf("depper:ingest:%s:%s:%s", packageVersion.Platform, packageVersion.Name, packageVersion.Version)
}

func randomHex(n int) string {
	id := make([]byte, n)
	io.ReadFull(rand.Reader, id)
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
	log.Println(key)
	var wasSet bool
	err := lib.RedisClient.Do(radix.Cmd(&wasSet, "SETNX", key, "true"))
	if err != nil {
		fmt.Errorf("Error trying to set key for redis %g", err)
		return
	}
	if wasSet {
		log.Println(key)
		lib.scheduleJob(packageVersion)
	}
}

func (lib *Sidekiq) scheduleJob(packageVersion data.PackageVersion) {
	job := createSyncJob(packageVersion)
	encoded, err := json.Marshal(job)
	if err != nil {
		fmt.Errorf("Error encoding sync job for sidekiq %g", err)
		return
	}
	err = lib.RedisClient.Do(radix.Cmd(nil, "LPUSH", fmt.Sprintf("queue:%s", job.Queue), string(encoded)))
	if err != nil {
		fmt.Errorf("Error calling redis.LPush to enqueue job: %g", err)
	}
}
