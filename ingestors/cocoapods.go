package ingestors

import (
	"strings"
	"time"

	log "github.com/sirupsen/logrus"

	"github.com/librariesio/depper/data"
)

const cocoapodsSchedule = "*/5 * * * *"
const cocoapodsReleasesUrl = "https://github.com/CocoaPods/Specs/commits.atom"

type cocoapods struct {
	LatestRun time.Time
}

func NewCocoaPods() *cocoapods {
	return &cocoapods{}
}

func (ingestor *cocoapods) Name() string {
	return "cocoapods"
}

func (ingestor *cocoapods) Schedule() string {
	return cocoapodsSchedule
}

func (ingestor *cocoapods) Ingest() []data.PackageVersion {
	packages := ingestor.ingestURL(cocoapodsReleasesUrl)
	ingestor.LatestRun = time.Now()
	return packages
}

func (ingestor *cocoapods) ingestURL(feedUrl string) []data.PackageVersion {
	var results []data.PackageVersion

	feed, err := depperGetFeed(feedUrl)
	if err != nil {
		log.WithFields(log.Fields{"ingestor": ingestor.Name()}).Error(err)
		return results
	}

	for _, item := range feed.Items {
		nameAndVersion := strings.SplitN(item.Title, " ", 3)
		results = append(results,
			data.PackageVersion{
				Platform:     ingestor.Name(),
				Name:         nameAndVersion[1],
				Version:      nameAndVersion[2],
				CreatedAt:    *item.UpdatedParsed,
				DiscoveryLag: time.Since(*item.UpdatedParsed),
			})
	}

	return results
}
