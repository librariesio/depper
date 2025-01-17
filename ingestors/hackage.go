package ingestors

import (
	"strings"
	"time"

	log "github.com/sirupsen/logrus"

	"github.com/librariesio/depper/data"
)

const hackageSchedule = "*/5 * * * *"
const hackageReleasesUrl = "https://hackage.haskell.org/packages/recent.rss"

type Hackage struct {
	LatestRun time.Time
}

func NewHackage() *Hackage {
	return &Hackage{}
}

func (ingestor *Hackage) Name() string {
	return "hackage"
}

func (ingestor *Hackage) Schedule() string {
	return hackageSchedule
}

func (ingestor *Hackage) Ingest() []data.PackageVersion {
	packages := ingestor.ingestURL(hackageReleasesUrl)
	ingestor.LatestRun = time.Now()
	return packages
}

func (ingestor *Hackage) ingestURL(feedUrl string) []data.PackageVersion {
	var results []data.PackageVersion

	feed, err := depperGetFeed(feedUrl)
	if err != nil {
		log.WithFields(log.Fields{"ingestor": ingestor.Name()}).Error(err)
		return results
	}

	for _, item := range feed.Items {
		nameAndVersion := strings.SplitN(item.Title, " ", 2)
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
