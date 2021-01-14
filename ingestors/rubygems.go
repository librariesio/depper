package ingestors

import (
	"io/ioutil"
	"net/http"
	"time"

	log "github.com/sirupsen/logrus"

	"github.com/buger/jsonparser"
	"github.com/librariesio/depper/data"
)

const RubyGemsSchedule = "*/5 * * * *"
const RubyGemsJustUpdatedURL = "https://rubygems.org/api/v1/activity/just_updated.json"
const RubyGemsLatestURL = "https://rubygems.org/api/v1/activity/latest.json"

type RubyGems struct {
	LatestRun time.Time
}

func NewRubyGems() *RubyGems {
	return &RubyGems{}
}

func (ingestor *RubyGems) Schedule() string {
	return RubyGemsSchedule
}

func (ingestor *RubyGems) Ingest() []data.PackageVersion {
	log.WithFields(log.Fields{"platform": "rubygems"}).Info("Depper ingest")
	results := append(
		ingestor.ingestURL(RubyGemsJustUpdatedURL),
		ingestor.ingestURL(RubyGemsLatestURL)...,
	)

	ingestor.LatestRun = time.Now()

	return results
}

func (ingestor *RubyGems) ingestURL(url string) []data.PackageVersion {
	var results []data.PackageVersion

	response, err := http.Get(url)
	if err != nil {
		log.WithFields(log.Fields{"ingestor": "rubygems", "error": err}).Error()
		return results
	}

	defer response.Body.Close()

	body, _ := ioutil.ReadAll(response.Body)

	_, err = jsonparser.ArrayEach(body, func(value []byte, dataType jsonparser.ValueType, offset int, err error) {
		name, _ := jsonparser.GetString(value, "name")
		version, _ := jsonparser.GetString(value, "version")
		createdAt, _ := jsonparser.GetString(value, "version_created_at")
		createdAtTime, _ := time.Parse(time.RFC3339, createdAt)

		results = append(results,
			data.PackageVersion{
				Platform:  "rubygems",
				Name:      name,
				Version:   version,
				CreatedAt: createdAtTime,
			})
	})

	if err != nil {
		log.WithFields(log.Fields{"ingestor": "rubygems", "error": err}).Error()
	}

	return results
}
