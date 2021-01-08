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
	"github.com/librariesio/depper/data"
)

const NPMRegistryHostname = "https://replicate.npmjs.com"
const NPMRegistryDatabase = "registry"
const NPMLatestSequenceRedisKey = "npm:updates:latest_sequence"

type NPM struct {
	redisClient *redis.Client
}

func NewNPM() *NPM {
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
	return &NPM{rdb}
}

type NPMChangeDoc struct {
	ID       string `json:"_id"`
	Rev      string `json:"_rev,omitempty"`
	Name     string `json:"name"`
	DistTags struct {
		Latest string `json:"latest"`
	} `json:"dist-tags"`
	Time map[string]string `json:"time"`
}

func (ingestor *NPM) Ingest(results chan data.PackageVersion) {
	since, err := ingestor.GetLatestSequence()
	if err != nil {
		log.Fatal(err)
	} else if since == "" {
		since = "now"
	}

	client, err := kivik.New("couch", NPMRegistryHostname)
	if err != nil {
		log.Fatal(err)
	}
	db := client.DB(NPMRegistryDatabase)
	changes, err := db.Changes(context.Background(), kivik.Options{"feed": "continuous", "since": since, "include_docs": true})
	if err != nil {
		log.Fatal(err)
	}
	defer changes.Close()

	for {
		if changes.Next() {
			var doc NPMChangeDoc
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
		} else {
			fmt.Printf("Nope")
		}
	}
}

func (ingestor *NPM) SetLatestSequence(seq string) error {
	err := ingestor.redisClient.Set(context.Background(), NPMLatestSequenceRedisKey, seq, 0).Err()
	if err != nil {
		return fmt.Errorf("Error trying to set key %s for redis %g", err)
	}
	return nil
}

func (ingestor *NPM) GetLatestSequence() (string, error) {
	val, err := ingestor.redisClient.Get(context.Background(), NPMLatestSequenceRedisKey).Result()
	if err != nil && err != redis.Nil {
		return "", err
	} else {
		return val, nil
	}
}
