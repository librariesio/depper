package ingestors

import (
	"context"
	"fmt"
	"time"

	log "github.com/sirupsen/logrus"

	kivik "github.com/go-kivik/kivik/v4"

	_ "github.com/go-kivik/couchdb/v4"
	"github.com/librariesio/depper/data"
	"github.com/librariesio/depper/redis"
)

const NPMRegistryHostname = "https://replicate.npmjs.com"
const NPMRegistryDatabase = "registry"
const NPMLatestSequenceRedisKey = "npm:updates:latest_sequence"

type NPM struct {
	couchClient *kivik.Client
}

func NewNPM() *NPM {
	return &NPM{couchClient: getCouchClient()}
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
		log.WithFields(log.Fields{"ingestor": "npm"}).Fatal(err)
	}

	couchDb := ingestor.couchClient.DB(NPMRegistryDatabase)
	changes, err := couchDb.Changes(context.Background(), kivik.Options{
		"feed":         "continuous",
		"since":        since,
		"include_docs": true,
		"timeout":      60000 * 2,
	})
	if err != nil {
		log.WithFields(log.Fields{"ingestor": "npm"}).Fatal(err)
	}
	defer changes.Close()

	for {
		if changes.Next() {
			var doc NPMChangeDoc
			if err := changes.ScanDoc(&doc); err != nil {
				log.WithFields(log.Fields{"seq": changes.Seq(), "id": changes.ID()}).Fatal(err)
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
					Platform:  "npm",
					Name:      doc.Name,
					Version:   latestVersion,
					CreatedAt: latestTime,
				}
				if err := ingestor.SetLatestSequence(changes.Seq()); err != nil {
					log.WithFields(log.Fields{"ingestor": "npm"}).Fatal(err)
				}
			}
		} else if changes.EOQ() {
			log.WithFields(log.Fields{"ingestor": "npm", "error": "EOQ"}).Error("Retrying in 5 seconds.")
			time.Sleep(5 * time.Second)
		} else {
			log.WithFields(log.Fields{"ingestor": "npm", "error": changes.Err()}).Error("Reconnecting in 5 seconds.")
			time.Sleep(5 * time.Second)
			couchDb = ingestor.couchClient.DB(NPMRegistryDatabase)
			changes, err = couchDb.Changes(context.Background(), kivik.Options{"feed": "continuous", "since": since, "include_docs": true})
			if err != nil {
				log.WithFields(log.Fields{"ingestor": "npm"}).Fatal(err)
			}
		}
	}
}

func (ingestor *NPM) SetLatestSequence(seq string) error {
	err := redis.Client.Set(context.Background(), NPMLatestSequenceRedisKey, seq, 0).Err()
	if err != nil {
		return fmt.Errorf("Error trying to set key %s for redis: %s", seq, err)
	}
	return nil
}

func (ingestorr *NPM) GetLatestSequence() (string, error) {
	val, err := redis.Client.Get(context.Background(), NPMLatestSequenceRedisKey).Result()
	if err == redis.Nil {
		return "now", nil
	} else if err != nil {
		return "", err
	} else {
		return val, nil
	}
}

func getCouchClient() *kivik.Client {
	kivikClient, err := kivik.New("couch", NPMRegistryHostname)
	if err != nil {
		log.WithFields(log.Fields{"ingestor": "npm"}).Fatal(err)
	}
	return kivikClient
}
