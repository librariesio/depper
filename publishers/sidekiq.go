package publishers

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"github.com/go-redis/redis/v8"
	"io"
	"time"
)

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
	rdb := redis.NewClient(&redis.Options{
		Addr:     "localhost:6379",
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
	return &Job{
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
	value, err := lib.RedisClient.Get(context, key).Result()
	var queueSync = false

	if err == redis.Nil {
		queueSync = true
	} else if err != nil {
		return err
	}

	err := lib.RedisClient.Set(context, key, "queued", 24*time.Hour)
	if err != nil {
		return err
	}
	if queueSync {
		log.Println(key)
		return lib.ScheduleJob(createSyncJob(platform, name))
	}
}

func (lib *LibrariesSidekiq) ScheduleJob(job Job) error {
	encoded, err := json.Marshal(job)
	if err != nil {
		return err
	}
	return lib.RedisClient.LPush(fmt.Sprintf("queue:%s", job.Queue), string(encoded))
}
