package ingestors

import (
	"encoding/json"
	"fmt"
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
	Items   []struct {
		Url             string `json:"@id"`
		Type            string `json:"@type"`
		CommitId        string `json:"commitId"`
		CommitTimeStamp string `json:"commitTimeStamp"`
		CommitTime      time.Time
	}
}

type nugetPage struct {
	PageId string `json:"@id"`
	Items  []struct {
		Url             string `json:"@id"`
		Type            string `json:"@type"`
		CommitTimeStamp string `json:"commitTimeStamp"`
		CommitTime      time.Time
		NugetId         string `json:"nuget:id"`
		NugetVersion    string `json:"nuget:version"`
	}
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
	ingestor.LatestRun = time.Now().Add(-48 * time.Hour)
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

	for i, item := range index.Items {
		item.CommitTime, _ = time.Parse(time.RFC3339, item.CommitTimeStamp)
		if item.CommitTime.After(ingestor.LatestRun) {
			fmt.Printf("Page %d - %s - %s\n", i, item.Url, item.Type)
			pageResults, err := ingestor.getPage(item.Url)
			if err != nil {
				return results, nil
			}
			results = append(results, pageResults...)
		}
	}

	return results, nil
}

func (ingestor *Nuget) getPage(url string) ([]data.PackageVersion, error) {
	fmt.Printf("Getting page %s\n", url)
	var results []data.PackageVersion

	response, err := http.Get(url)
	if err != nil {
		return []data.PackageVersion{}, err
	}
	defer response.Body.Close()

	body, _ := ioutil.ReadAll(response.Body)
	var page nugetPage
	json.Unmarshal(body, &page)
	fmt.Printf("Page %s %d\n", page.PageId, len(page.Items))

	for _, item := range page.Items {
		item.CommitTime, _ = time.Parse(time.RFC3339, item.CommitTimeStamp)
		if item.CommitTime.After(ingestor.LatestRun) {
			results = append(results,
				data.PackageVersion{
					Platform:  "nuget",
					Name:      item.NugetId,
					Version:   item.NugetVersion,
					CreatedAt: item.CommitTime,
				})
		} else {
			fmt.Printf("Skipping %s\n", item)
		}
	}

	return results, nil
}
