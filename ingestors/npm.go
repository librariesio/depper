package ingestors

import (
	"context"
	"time"

	log "github.com/sirupsen/logrus"

	kivik "github.com/go-kivik/kivik/v4"

	_ "github.com/go-kivik/couchdb/v4"
	"github.com/librariesio/depper/data"
)

const NPMRegistryHostname = "https://replicate.npmjs.com"
const NPMRegistryDatabase = "registry"

type NPM struct {
	couchClient *kivik.Client
}

func NewNPM() *NPM {
	return &NPM{couchClient: getCouchClient()}
}

// See https://github.com/npm/registry-follower-tutorial#moar-data-please
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

	// See https://docs.couchdb.org/en/3.2.0/api/database/changes.html
	options := kivik.Options{
		"feed":         "continuous",
		"since":        since,
		"include_docs": true,
		// NB: previously with "timeout: 60000 * 2", we kept getting an internal error from npm, which surfaced as
		// "stream error: stream ID 123; INTERNAL_ERROR". They showed up when there was no activity for 50 seconds,
		// and we're not sure why. But setting a heartbeat ensures the connection stays open every 5 seconds via empty line.
		"heartbeat": 5000,
	}
	couchDb := ingestor.couchClient.DB(NPMRegistryDatabase)
	changes, err := couchDb.Changes(context.Background(), options)
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
				discoveryLag := time.Since(latestTime)
				results <- data.PackageVersion{
					Platform:     "npm",
					Name:         doc.Name,
					Version:      latestVersion,
					CreatedAt:    latestTime,
					DiscoveryLag: discoveryLag,
				}
				if _, err := setBookmark(ingestor, changes.Seq()); err != nil {
					log.WithFields(log.Fields{"ingestor": "npm"}).Fatal(err)
				}
			}
		} else {
			log.WithFields(log.Fields{"ingestor": "npm", "error": changes.Err()}).Error("Reconnecting in 5 seconds.")
			time.Sleep(5 * time.Second)
			couchDb = ingestor.couchClient.DB(NPMRegistryDatabase)
			changes, err = couchDb.Changes(context.Background(), options)
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
