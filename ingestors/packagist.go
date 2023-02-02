package ingestors

import (
	"strings"
	"time"

	log "github.com/sirupsen/logrus"

	"github.com/librariesio/depper/data"
	"github.com/mmcdole/gofeed"
)

const packagistSchedule = "*/5 * * * *"
const packagistReleasesUrl = "https://packagist.org/feeds/releases.rss"

type Packagist struct {
	LatestRun time.Time
}

func NewPackagist() *Packagist {
	return &Packagist{}
}

func (ingestor *Packagist) Name() string {
	return "packagist_main"
}

func (ingestor *Packagist) Schedule() string {
	return packagistSchedule
}

func (ingestor *Packagist) Ingest() []data.PackageVersion {
	packages := ingestor.ingestURL(packagistReleasesUrl)
	ingestor.LatestRun = time.Now()
	return packages
}

func (ingestor *Packagist) ingestURL(feedUrl string) []data.PackageVersion {
	var results []data.PackageVersion

	fp := gofeed.NewParser()

	feed, err := fp.ParseURL(packagistReleasesUrl)
	if err != nil {
		log.WithFields(log.Fields{"ingestor": ingestor.Name()}).Error(err)
		return results
	}

	for _, item := range feed.Items {
		nameAndVersion := strings.SplitN(item.GUID, " ", 2)
		results = append(results,
			data.PackageVersion{
				Platform:     ingestor.Name(),
				Name:         nameAndVersion[0],
				Version:      nameAndVersion[1],
				CreatedAt:    *item.PublishedParsed,
				DiscoveryLag: time.Since(*item.PublishedParsed),
			})
	}

	return results
}
