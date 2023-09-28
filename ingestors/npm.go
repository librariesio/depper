package ingestors

import (
	"context"
	"fmt"
	"time"

	log "github.com/sirupsen/logrus"

	kivik "github.com/go-kivik/kivik/v4"

	_ "github.com/go-kivik/kivik/v4/couchdb"
	"github.com/librariesio/depper/data"
)

const NPMRegistryHostname = "https://replicate.npmjs.com"
const NPMRegistryDatabase = "registry"
const RetryDelaySeconds = 5

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

/**
 * NPM package updates are ingested from a continuous reading of a remote
 * CouchDB database at replicate.npmjs.com. CouchDB databases provide a Changes
 * feed that provide the changes since the last set of changes were published:
 * https://docs.couchbase.com/sync-gateway/current/changes-feed.html
 *
 * We connect to the CouchDB database and continually read for the next set
 * of changes to the database. Once we receive changes, we take the found releases
 * and add them to the processing queue. If no changes are available, we
 * reconnect to the database in a number of seconds and try again.
 *
 * Since this is based on detecting changes to the NPM database, it may be
 * possible for a client to miss out on changes for some reason. Libraries only
 * processes individual NPM versions delivered by depper, rather than reprocessing
 * the whole package, so in that case, Libraries may not know about that version
 * onless something triggers a full package resync on Libraries.
 */
func (ingestor *NPM) Ingest(results chan data.PackageVersion) {
	since, err := getBookmark(ingestor, "now")
	if err != nil {
		log.WithFields(log.Fields{"ingestor": "npm"}).Fatal(err)
	}

	// See https://docs.couchdb.org/en/3.2.0/api/database/changes.html
	options := kivik.Params(map[string]interface{}{
		"feed":         "continuous",
		"since":        since,
		"include_docs": true,
		// NB: previously with "timeout: 60000 * 2", we kept getting an internal error from npm, which surfaced as
		// "stream error: stream ID 123; INTERNAL_ERROR". They showed up when there was no activity for 50 seconds,
		// and we're not sure why. But setting a heartbeat ensures the connection stays open every 5 seconds via empty line.
		"heartbeat": RetryDelaySeconds * 1000,
	})
	couchDb := ingestor.couchClient.DB(NPMRegistryDatabase)
	changes := couchDb.Changes(context.Background(), options)
	if err = changes.Err(); err != nil {
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
			log.WithFields(log.Fields{"ingestor": "npm", "error": changes.Err()}).Error(fmt.Sprintf("Reconnecting in %d seconds.", RetryDelaySeconds))
			time.Sleep(RetryDelaySeconds * time.Second)
			couchDb = ingestor.couchClient.DB(NPMRegistryDatabase)
			changes = couchDb.Changes(context.Background(), options)
			if err = changes.Err(); err != nil {
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
