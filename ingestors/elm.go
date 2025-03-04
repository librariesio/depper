package ingestors

import (
	"fmt"
	"net/url"
	"strings"
	"time"

	log "github.com/sirupsen/logrus"

	"github.com/librariesio/depper/data"
)

const elmSchedule = "0 */4 * * *"
const elmFeed = "https://releases.elm.dmy.fr/.rss"

type Elm struct {
	LatestRun time.Time
}

func NewElm() *Elm {
	return &Elm{}
}

func (ingestor *Elm) Name() string {
	return "elm"
}

func (ingestor *Elm) Schedule() string {
	return elmSchedule
}

func (ingestor *Elm) Ingest() []data.PackageVersion {
	packages := ingestor.ingestURL(elmFeed)
	ingestor.LatestRun = time.Now()
	return packages
}

func (ingestor *Elm) ingestURL(feedUrl string) []data.PackageVersion {
	var results []data.PackageVersion

	feed, err := depperGetFeed(feedUrl)

	if err != nil {
		log.WithFields(log.Fields{"ingestor": ingestor.Name()}).Error(err)
		return results
	}
	for _, item := range feed.Items {
		parsed, _ := url.Parse(item.Link)
		parts := strings.Split(parsed.Path, "/")
		discoveryLag := time.Since(*item.PublishedParsed)
		results = append(results,
			data.PackageVersion{
				Platform:     "elm",
				Name:         fmt.Sprintf("%s/%s", parts[2], parts[3]),
				Version:      parts[4],
				CreatedAt:    *item.PublishedParsed,
				DiscoveryLag: discoveryLag,
			})
	}
	return results
}
