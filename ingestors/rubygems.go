package ingestors

import (
	"io"
	"time"

	log "github.com/sirupsen/logrus"

	"github.com/buger/jsonparser"
	"github.com/librariesio/depper/data"
)

const rubyGemsSchedule = "*/5 * * * *"
const rubyGemsJustUpdatedURL = "https://rubygems.org/api/v1/activity/just_updated.json"
const rubyGemsLatestURL = "https://rubygems.org/api/v1/activity/latest.json"

type RubyGems struct {
	LatestRun time.Time
}

func NewRubyGems() *RubyGems {
	return &RubyGems{}
}

func (ingestor *RubyGems) Name() string {
	return "rubygems"
}

func (ingestor *RubyGems) Schedule() string {
	return rubyGemsSchedule
}

func (ingestor *RubyGems) Ingest() []data.PackageVersion {
	results := append(
		ingestor.ingestURL(rubyGemsJustUpdatedURL),
		ingestor.ingestURL(rubyGemsLatestURL)...,
	)

	ingestor.LatestRun = time.Now()

	return results
}

func (ingestor *RubyGems) ingestURL(url string) []data.PackageVersion {
	var results []data.PackageVersion

	response, err := depperGetUrl(url)
	if err != nil {
		log.WithFields(log.Fields{"ingestor": ingestor.Name(), "error": err}).Error()
		return results
	}

	defer response.Body.Close()

	body, _ := io.ReadAll(response.Body)

	_, _ = jsonparser.ArrayEach(body, func(value []byte, dataType jsonparser.ValueType, offset int, err error) {
		if err != nil {
			log.WithFields(
				log.Fields{
					"ingestor": ingestor.Name(),
					"error":    err,
					"value":    string(value),
					"dataType": dataType.String(),
					"offset":   offset,
				},
			).Error()
			return
		}

		name, _ := jsonparser.GetString(value, "name")
		version, _ := jsonparser.GetString(value, "version")
		createdAt, _ := jsonparser.GetString(value, "version_created_at")
		createdAtTime, _ := time.Parse(time.RFC3339, createdAt)
		discoveryLag := time.Since(createdAtTime)

		results = append(results,
			data.PackageVersion{
				Platform:     "rubygems",
				Name:         name,
				Version:      version,
				CreatedAt:    createdAtTime,
				DiscoveryLag: discoveryLag,
			})
	})

	return results
}
