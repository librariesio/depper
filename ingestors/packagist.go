package ingestors

import (
	"strings"
	"time"

	log "github.com/sirupsen/logrus"

	"github.com/librariesio/depper/data"
	"github.com/mmcdole/gofeed"
)

const packagistSchedule = "*/15 * * * *"
const packagistReleasesUrl = "https://packagist.org/feeds/releases.rss"

type Packagist struct {
	LatestRun time.Time
}

func NewPackagist() *Packagist {
	return &Packagist{}
}

func (ingestor *Packagist) Schedule() string {
	return packagistSchedule
}

func (ingestor *Packagist) Ingest() []data.PackageVersion {
	// Until we save LatestRun state, go back two hours by default.
	if ingestor.LatestRun.IsZero() {
		ingestor.LatestRun = time.Now().Add(-120 * time.Minute)
	}
	packages := ingestor.ingestURL(packagistReleasesUrl)
	ingestor.LatestRun = time.Now()
	return packages
}

func (ingestor *Packagist) ingestURL(feedUrl string) []data.PackageVersion {
	var results []data.PackageVersion

	fp := gofeed.NewParser()

	feed, err := fp.ParseURL(packagistReleasesUrl)
	if err != nil {
		log.WithFields(log.Fields{"ingestor": "packagist"}).Error(err)
		return results
	}

	for _, item := range feed.Items {
		if item.PublishedParsed.After(ingestor.LatestRun) {
			nameAndVersion := strings.SplitN(item.GUID, " ", 2)
			results = append(results,
				data.PackageVersion{
					Platform:  "packagist",
					Name:      nameAndVersion[0],
					Version:   nameAndVersion[1],
					CreatedAt: *item.PublishedParsed,
				})
		}
	}

	return results
}
