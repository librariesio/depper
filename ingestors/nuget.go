package ingestors

import (
	"encoding/json"
	"io"
	"net/http"
	"time"

	log "github.com/sirupsen/logrus"

	"github.com/librariesio/depper/data"
)

const nugetSchedule = "*/5 * * * *"
const nugetIndexUrl = "https://api.nuget.org/v3/catalog0/index.json"
const defaultLatestRun = -120 * time.Minute

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
	// Until we save LatestRun state, we need to set a LatestRun to avoid scanning every single release in the index.
	if ingestor.LatestRun.IsZero() {
		ingestor.LatestRun = time.Now().Add(defaultLatestRun)
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

	var index nugetIndex
	body, _ := io.ReadAll(response.Body)
	err = json.Unmarshal(body, &index)
	if err != nil {
		return results, err
	}

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

	body, _ := io.ReadAll(response.Body)
	var page nugetPage
	err = json.Unmarshal(body, &page)
	if err != nil {
		return results, err
	}

	for _, pkg := range page.Packages {
		pkg.CommitTime, _ = time.Parse(time.RFC3339, pkg.CommitTimeStamp)
		if pkg.CommitTime.After(ingestor.LatestRun) {
			results = append(
				results,
				data.PackageVersion{
					Platform:     "nuget",
					Name:         pkg.Name,
					Version:      pkg.Version,
					CreatedAt:    pkg.CommitTime,
					DiscoveryLag: time.Since(pkg.CommitTime),
				},
			)
		}
	}

	return results, nil
}
