package ingestors

import (
	"context"
	"fmt"
	"log"
	"time"

	kivik "github.com/go-kivik/kivik/v4"

	_ "github.com/go-kivik/couchdb/v4" // The CouchDB driver
	// "github.com/go-kivik/kivik"
	// "github.com/go-kivik/kivik/driver"
	"github.com/librariesio/depper/data"
)

const NpmSchedule = "*/10 * * * *"
const NpmRegistryHostname = "https://replicate.npmjs.com"
const NpmRegistryDatabase = "registry"

type Npm struct {
	LatestRun time.Time
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

func NewNpm() *Npm {
	return &Npm{}
}

func (ingestor *Npm) Schedule() string {
	return NpmSchedule
}

func (ingestor *Npm) Ingest() []data.PackageVersion {
	// results := append(
	// 	ingestor.ingestURL(NpmJustUpdatedURL),
	// 	ingestor.ingestURL(NpmLatestURL)...,
	// )

	// ingestor.LatestRun = time.Now()

	updates := ingestor.ingestCouchUpdates()
	fmt.Printf("Updates: %s\n", updates)

	results := append(
		[]data.PackageVersion{},
		updates...,
	)

	return results
}

func (ingestor *Npm) ingestCouchUpdates() []data.PackageVersion {
	var results []data.PackageVersion

	client, err := kivik.New("couch", NpmRegistryHostname)
	if err != nil {
		log.Fatal(err)
	}

	db := client.DB(NpmRegistryDatabase)
	changes, err := db.Changes(context.Background(), kivik.Options{"feed": "continuous", "since": "now", "include_docs": true})
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
				results = append(results)
			}
			fmt.Printf("  Time is: %s\t=>\t%s\n", latestVersion, latestTime)
		} else {
			break // wait for next ingestor run
		}
	}
	return results
}

// func (ingestor *Npm) ingestURL(url string) []data.PackageVersion {
// 	var results []data.PackageVersion

// 	response, err := http.Get(url)
// 	if err != nil {
// 		log.Print(err)
// 		return results
// 	}

// 	defer response.Body.Close()

// 	body, _ := ioutil.ReadAll(response.Body)

// 	_, err = jsonparser.ArrayEach(body, func(value []byte, dataType jsonparser.ValueType, offset int, err error) {
// 		name, _ := jsonparser.GetString(value, "name")
// 		version, _ := jsonparser.GetString(value, "version")
// 		createdAt, _ := jsonparser.GetString(value, "version_created_at")
// 		createdAtTime, _ := time.Parse(time.RFC3339, createdAt)

// 		results = append(results,
// 			data.PackageVersion{
// 				Platform:  "npm",
// 				Name:      name,
// 				Version:   version,
// 				CreatedAt: createdAtTime,
// 			})
// 	})

// 	if err != nil {
// 		log.Print(err)
// 	}

// 	return results
// }
