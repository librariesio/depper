package publishers

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"github.com/go-redis/redis/v8"
	"io"
	"os"
	"time"
)

const TTL = 24 * time.Hour

type LibrariesSidekiq struct {
	RedisClient redis.Client
	Context     context
}

type LibrariesJob struct {
	Class      string   `json:"class"`
	Queue      string   `json:"queue"`
	Args       []string `json:"args"`
	Retry      bool     `json:"retry"`
	JID        string   `json:"jid"`
	CreatedAt  int64    `json:"created_at"`
	EnqueuedAt in64     `json:"enqueued_at"`
}

func New() *LibrariesSidekiq {
	address := "localhost:6379"
	envVal, envFound := os.LookupEnv("REDIS_URL")
	if envFound {
		address = envVal
	}
	rdb := redis.NewClient(&redis.Options{
		Addr:     address,
		Password: "",
		DB:       0,
	})
	return &LibrariesSidekiq{
		RedisClient: rdb,
		Context:     context.Background(),
	}
}
func getKey(platform string, name string, version string) string {
	return fmt.Sprintf("depper:ingest:%s:%s:%s", platform, name, version)
}

func randomHex(n int) string {
	id := make([]byte, n)
	io.ReadFull(rand.Reader, id)
	return hex.EncodeToString(id)
}

func createSyncJob(platform string, name string, version string) {
	return &LibrariesJob{
		Retry:      true,
		Class:      "PackageManagerDownloadWorker",
		Queue:      "critical",
		JID:        randomHex(12),
		EnqueuedAt: time.Now().Unix(),
		CreatedAt:  time.Now().Unix(),
		Args:       [2]string{platform, name},
	}
}

func (lib *LibrariesSidekiq) QueueSync(platform string, name string, version string) error {
	key := getKey(platform, name, version)

	value, err := lib.RedisClient.SetNX(context, key, true, TTL)
	if err != nil {
		return err
	}
	if value {
		log.Println(key)
		return lib.ScheduleJob(platform, name)
	}
}

func (lib *LibrariesSidekiq) ScheduleJob(platform string, name string) error {
	encoded, err := json.Marshal(createSyncJob(platform, name))
	if err != nil {
		return err
	}
	return lib.RedisClient.LPush(fmt.Sprintf("queue:%s", job.Queue), string(encoded))
}
