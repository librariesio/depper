package ingestors

import (
	"encoding/json"
	"io/ioutil"
	"net/http"
	"time"

	log "github.com/sirupsen/logrus"

	"github.com/librariesio/depper/data"
)

const nugetSchedule = "18 * * * *"
const nugetIndexUrl = "https://api.nuget.org/v3/catalog0/index.json"

type nugetIndex struct {
	IndexId string `json:"@id"`
	Pages   []struct {
		Url             string `json:"@id"`
		CommitTimeStamp string `json:"commitTimeStamp"`
		CommitTime      time.Time
	} `json:"items"`
}

type nugetPage struct {
	PageId   string `json:"@id"`
	Packages []struct {
		Url             string `json:"@id"`
		Type            string `json:"@type"`
		CommitTimeStamp string `json:"commitTimeStamp"`
		CommitTime      time.Time
		Name            string `json:"nuget:id"`
		Version         string `json:"nuget:version"`
	} `json:"items"`
}

type Nuget struct {
	LatestRun time.Time
}

func NewNuget() *Nuget {
	return &Nuget{}
}

func (ingestor *Nuget) Schedule() string {
	return nugetSchedule
}

func (ingestor *Nuget) Ingest() []data.PackageVersion {
	// Until we save LatestRun state, begin with the last 24 hours.
	if ingestor.LatestRun.IsZero() {
		ingestor.LatestRun = time.Now().Add(-24 * time.Hour)
	}
	packages := ingestor.ingestURL(nugetIndexUrl)
	ingestor.LatestRun = time.Now()
	return packages
}

func (ingestor *Nuget) ingestURL(url string) []data.PackageVersion {
	var results []data.PackageVersion

	results, err := ingestor.getIndex(url)
	if err != nil {
		log.WithFields(log.Fields{"ingestor": "nuget", "error": err}).Error()
	}

	return results
}

func (ingestor *Nuget) getIndex(url string) ([]data.PackageVersion, error) {
	var results []data.PackageVersion

	response, err := http.Get(url)
	if err != nil {
		return results, err
	}
	defer response.Body.Close()

	body, _ := ioutil.ReadAll(response.Body)
	var index nugetIndex
	json.Unmarshal(body, &index)

	for _, page := range index.Pages {
		page.CommitTime, _ = time.Parse(time.RFC3339, page.CommitTimeStamp)
		if page.CommitTime.After(ingestor.LatestRun) {
			pageResults, err := ingestor.getPage(page.Url)
			if err != nil {
				return results, nil
			}
			results = append(results, pageResults...)
		}
	}

	return results, nil
}

func (ingestor *Nuget) getPage(url string) ([]data.PackageVersion, error) {
	var results []data.PackageVersion

	response, err := http.Get(url)
	if err != nil {
		return []data.PackageVersion{}, err
	}
	defer response.Body.Close()

	body, _ := ioutil.ReadAll(response.Body)
	var page nugetPage
	json.Unmarshal(body, &page)

	for _, pkg := range page.Packages {
		pkg.CommitTime, _ = time.Parse(time.RFC3339, pkg.CommitTimeStamp)
		if pkg.CommitTime.After(ingestor.LatestRun) {
			results = append(
				results,
				data.PackageVersion{
					Platform:  "nuget",
					Name:      pkg.Name,
					Version:   pkg.Version,
					CreatedAt: pkg.CommitTime,
				},
			)
		}
	}

	return results, nil
}
