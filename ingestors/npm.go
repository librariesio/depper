package ingestors

import (
	"context"
	"fmt"
	"log"
	"os"
	"time"

	kivik "github.com/go-kivik/kivik/v4"
	"github.com/go-redis/redis/v8"

	_ "github.com/go-kivik/couchdb/v4" // The CouchDB driver
	// "github.com/go-kivik/kivik"
	// "github.com/go-kivik/kivik/driver"
	"github.com/librariesio/depper/data"
)

// const NpmSchedule = "*/10 * * * *"
const NpmRegistryHostname = "https://replicate.npmjs.com"
const NpmRegistryDatabase = "registry"
const npmLatestSequenceRedisKey = "npm:updates:latest_sequence"

type Npm struct {
	redisClient *redis.Client
}

func NewNpm() *Npm {
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
	return &Npm{rdb}
}

type npmChangeDoc struct {
	ID       string `json:"_id"`
	Rev      string `json:"_rev,omitempty"`
	Name     string `json:"name"`
	DistTags struct {
		Latest string `json:"latest"`
	} `json:"dist-tags"`
	Time map[string]string `json:"time"`
}

func (ingestor *Npm) Ingest(results chan data.PackageVersion) {
	since, err := ingestor.GetLatestSequence()
	if err != nil {
		log.Fatal(err)
	} else if since == "" {
		since = "now"
	}

	client, err := kivik.New("couch", NpmRegistryHostname)
	if err != nil {
		log.Fatal(err)
	}
	db := client.DB(NpmRegistryDatabase)
	changes, err := db.Changes(context.Background(), kivik.Options{"feed": "continuous", "since": since, "include_docs": true})
	if err != nil {
		log.Fatal(err)
	}
	defer changes.Close()

	for {
		if changes.Next() {
			var doc npmChangeDoc
			if err := changes.ScanDoc(&doc); err != nil {
				log.Fatal("Error parsing json doc at sequence %s with ID %s\n", changes.Seq(), changes.ID())
			}
			var latestVersion string
			var latestTime time.Time
			for k, v := range doc.Time {
				if k != "modified" && k != "created" && len(v) > 0 {
					if t, err := time.Parse(time.RFC3339, v); err == nil {
						if t.After(latestTime) {
							latestTime = t
							latestVersion = k
						}
					}
				}
			}
			if latestVersion != "" {
				results <- data.PackageVersion{
					Platform:  "NPM",
					Name:      doc.Name,
					Version:   latestVersion,
					CreatedAt: latestTime,
				}
				if err := ingestor.SetLatestSequence(changes.Seq()); err != nil {
					log.Fatalf(err.Error())
				}
			}
		}
	}
}

func (ingestor *Npm) SetLatestSequence(seq string) error {
	err := ingestor.redisClient.Set(context.Background(), npmLatestSequenceRedisKey, seq, 0).Err()
	if err != nil {
		return fmt.Errorf("Error trying to set key %s for redis %g", err)
	}
	return nil
}

func (ingestor *Npm) GetLatestSequence() (string, error) {
	val, err := ingestor.redisClient.Get(context.Background(), npmLatestSequenceRedisKey).Result()
	if err != nil && err != redis.Nil {
		return "", err
	} else {
		return val, nil
	}
}
