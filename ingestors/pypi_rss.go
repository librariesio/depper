package ingestors

import (
	"strings"
	"time"

	log "github.com/sirupsen/logrus"

	"github.com/librariesio/depper/data"
	"github.com/mmcdole/gofeed"
)

//const pyPiPackagesFeedUrl = "https://pypi.org/rss/packages.xml"
const pyPiUpdatesFeedUrl = "https://pypi.org/rss/updates.xml"

type PyPiRss struct {
	LatestRun time.Time
}

func NewPyPiRss() *PyPiRss {
	return &PyPiRss{}
}

func (ingestor *PyPiRss) Name() string {
	return "npm"
}

func (ingestor *PyPiRss) Schedule() string {
	return "* * * * *"
}

func (ingestor *PyPiRss) Ingest() []data.PackageVersion {
	packages := ingestor.ingestURL(pyPiUpdatesFeedUrl)

	ingestor.LatestRun = time.Now()

	return packages
}

func (ingestor *PyPiRss) ingestURL(feedUrl string) []data.PackageVersion {
	var results []data.PackageVersion

	fp := gofeed.NewParser()

	feed, err := fp.ParseURL(feedUrl)
	if err != nil {
		log.WithFields(log.Fields{"ingestor": "pypiRss"}).Error(err)
		return results
	}

	for _, item := range feed.Items {
		nameAndVersion := strings.SplitN(item.Title, " ", 2)
		results = append(results,
			data.PackageVersion{
				Platform:  "pypi",
				Name:      nameAndVersion[0],
				Version:   nameAndVersion[1],
				CreatedAt: *item.PublishedParsed,
			})
	}

	return results
}
