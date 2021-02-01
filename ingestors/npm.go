package ingestors

import (
	"context"
	"time"

	log "github.com/sirupsen/logrus"

	kivik "github.com/go-kivik/kivik/v4"

	_ "github.com/go-kivik/couchdb/v4"
	"github.com/librariesio/depper/data"
	"github.com/librariesio/depper/redis"
)

const NPMRegistryHostname = "https://replicate.npmjs.com"
const NPMRegistryDatabase = "registry"
const latestSequenceBookmark = "npm:updates:latest_sequence"

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

func (ingestor *NPM) Name() string {
	return "npm"
}

func (ingestor *NPM) Ingest(results chan data.PackageVersion) {
	since, err := getBookmark(ingestor, "now")
	if err != nil {
		log.WithFields(log.Fields{"ingestor": "npm"}).Fatal(err)
	}

	// TODO: This is a migration for bookmarks, can delete after first deployment
	if since == nil {
		since, _ = redis.Client.Get(context.Background(), "npm:updates:latest_sequence").Result()
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
				if _, err := setBookmark(ingestor, changes.Seq()); err != nil {
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

func getCouchClient() *kivik.Client {
	kivikClient, err := kivik.New("couch", NPMRegistryHostname)
	if err != nil {
		log.WithFields(log.Fields{"ingestor": "npm"}).Fatal(err)
	}
	return kivikClient
}
