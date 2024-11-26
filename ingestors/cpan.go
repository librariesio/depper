package ingestors

import (
	"strings"
	"time"

	log "github.com/sirupsen/logrus"

	"github.com/librariesio/depper/data"
)

const cpanSchedule = "*/5 * * * *"
const cpanReleasesUrl = "https://metacpan.org/recent.rss"

type CPAN struct {
	LatestRun time.Time
}

func NewCPAN() *CPAN {
	return &CPAN{}
}

func (ingestor *CPAN) Name() string {
	return "cpan"
}

func (ingestor *CPAN) Schedule() string {
	return cpanSchedule
}

func (ingestor *CPAN) Ingest() []data.PackageVersion {
	packages := ingestor.ingestURL(cpanReleasesUrl)
	ingestor.LatestRun = time.Now()
	return packages
}

func (ingestor *CPAN) ingestURL(feedUrl string) []data.PackageVersion {
	var results []data.PackageVersion

	feed, err := depperGetFeed(feedUrl)
	if err != nil {
		log.WithFields(log.Fields{"ingestor": ingestor.Name()}).Error(err)
		return results
	}

	for _, item := range feed.Items {
		pieces := strings.Split(item.Title, "-")
		results = append(results,
			data.PackageVersion{
				Platform:     ingestor.Name(),
				Name:         strings.Join(pieces[0:len(pieces)-1], "-"),
				Version:      pieces[len(pieces)-1],
				CreatedAt:    *item.PublishedParsed,
				DiscoveryLag: time.Since(*item.PublishedParsed),
			})
	}

	return results
}
