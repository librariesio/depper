package ingestors

import (
	"io"
	"net/http"
	"time"

	"github.com/buger/jsonparser"
	log "github.com/sirupsen/logrus"

	"github.com/librariesio/depper/data"
)

const cargoSchedule = "*/5 * * * *"
const cargoFeed = "https://crates.io/api/v1/summary"

type Cargo struct {
	LatestRun time.Time
}

func NewCargo() *Cargo {
	return &Cargo{}
}

func (ingestor *Cargo) Schedule() string {
	return cargoSchedule
}

func (ingestor *Cargo) Ingest() []data.PackageVersion {
	packages := ingestor.ingestURL(cargoFeed)
	ingestor.LatestRun = time.Now()
	return packages
}

func (ingestor *Cargo) ingestURL(url string) []data.PackageVersion {
	var results []data.PackageVersion

	response, err := http.Get(url)
	if err != nil {
		log.WithFields(log.Fields{"ingestor": "cargo", "error": err}).Error()
		return results
	}

	defer response.Body.Close()

	body, _ := io.ReadAll(response.Body)
	err = jsonparser.ObjectEach(body, func(key []byte, value []byte, dataType jsonparser.ValueType, offset int) error {
		var subErr error
		if string(key) == "just_updated" || string(key) == "new_crates" {
			_, subErr = jsonparser.ArrayEach(value, func(value []byte, dataType jsonparser.ValueType, offset int, err error) {
				name, _ := jsonparser.GetString(value, "name")
				version, _ := jsonparser.GetString(value, "newest_version")
				createdAt, _ := jsonparser.GetString(value, "updated_at")
				createdAtTime, _ := time.Parse(time.RFC3339, createdAt)
				discoveryLag := time.Since(createdAtTime)

				results = append(
					results,
					data.PackageVersion{
						Platform:     "cargo",
						Name:         name,
						Version:      version,
						CreatedAt:    createdAtTime,
						DiscoveryLag: discoveryLag,
					},
				)
			})
		}
		return subErr
	})

	if err != nil {
		log.WithFields(log.Fields{"ingestor": "cargo", "error": err}).Error()
	}

	return results
}
