package publishers

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"time"

	log "github.com/sirupsen/logrus"

	"github.com/librariesio/depper/data"
	"github.com/librariesio/depper/redis"
)

const TTL = 24 * time.Hour

type Sidekiq struct {
	Context context.Context
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
	return &Sidekiq{
		Context: context.Background(),
	}
}
func getKey(packageVersion data.PackageVersion) string {
	return fmt.Sprintf("depper:ingest:%s:%s:%s", packageVersion.Platform, packageVersion.Name, packageVersion.Version)
}

func randomHex(n int) string {
	id := make([]byte, n)
	_, err := io.ReadFull(rand.Reader, id)
	if err != nil {
		log.WithFields(log.Fields{"ingestor": "npm", "error": err})
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
		Args:       []string{packageVersion.Platform, packageVersion.Name, packageVersion.Version},
	}
}

func (lib *Sidekiq) Publish(packageVersion data.PackageVersion) {
	key := getKey(packageVersion)

	wasSet, err := redis.Client.SetNX(lib.Context, key, true, TTL).Result()
	if err != nil {
		log.WithFields(log.Fields{"publisher": "sidekiq"}).Error(err)
		return
	}
	if wasSet {
		lib.scheduleJob(packageVersion)
	}
}

func (lib *Sidekiq) scheduleJob(packageVersion data.PackageVersion) {
	job := createSyncJob(packageVersion)
	encoded, err := json.Marshal(job)
	if err != nil {
		log.WithFields(log.Fields{"publisher": "sidekiq"}).Error(err)
		return
	}
	redis.Client.LPush(lib.Context, fmt.Sprintf("queue:%s", job.Queue), string(encoded))
}
