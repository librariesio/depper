package ingestors

import (
	"io"
	"time"

	log "github.com/sirupsen/logrus"

	"github.com/buger/jsonparser"
	"github.com/librariesio/depper/data"
)

const hexSchedule = "*/5 * * * *"
const hexPackagesUrl = "https://hex.pm/api/packages?sort=updated_at"

type Hex struct {
	LatestRun time.Time
}

func NewHex() *Hex {
	return &Hex{}
}

func (ingestor *Hex) Name() string {
	return "hex"
}

func (ingestor *Hex) Schedule() string {
	return hexSchedule
}

func (ingestor *Hex) Ingest() []data.PackageVersion {
	var results []data.PackageVersion

	response, err := depperGetUrl(hexPackagesUrl, map[string]string{})
	if err != nil {
		log.WithFields(log.Fields{"ingestor": ingestor.Name(), "error": err}).Error()
		return results
	}

	defer response.Body.Close()

	body, _ := io.ReadAll(response.Body)

	_, err = jsonparser.ArrayEach(
		body,
		func(value []byte, dataType jsonparser.ValueType, offset int, err error) {
			if err != nil {
				log.WithFields(log.Fields{"ingestor": ingestor.Name(), "error": err, "value": string(value), "dataType": dataType.String(), "offset": offset}).Error()
				return
			}
			name, _ := jsonparser.GetString(value, "name")
			updatedAt, _ := jsonparser.GetString(value, "updated_at")
			version, _ := jsonparser.GetString(value, "latest_version")
			updatedAtTime, _ := time.Parse(time.RFC3339, updatedAt)

			discoveryLag := time.Since(updatedAtTime)
			results = append(results,
				data.PackageVersion{
					Platform:     "hex",
					Name:         name,
					Version:      version,
					CreatedAt:    updatedAtTime,
					DiscoveryLag: discoveryLag,
				})
		},
	)
	if err != nil {
		log.WithFields(log.Fields{"ingestor": ingestor.Name(), "error": err}).Error()
	}

	ingestor.LatestRun = time.Now()

	return results
}
