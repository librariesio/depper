package ingestors

import (
	"fmt"
	"github.com/librariesio/depper/data"
	"github.com/mmcdole/gofeed"
	"log"
	"net/url"
	"strings"
	"time"
)

const elmSchedule = "0 */4 * * *"
const elmFeed = "https://elm-greenwood.com/.rss"

type Elm struct {
	LatestRun time.Time
}

func NewElm() *Elm {
	return &Elm{}
}

func (ingestor *Elm) Schedule() string {
	return elmSchedule
}

func (ingestor *Elm) Ingest() []data.PackageVersion {
	ingestor.LatestRun = time.Now()
	return ingestor.ingestURL(elmFeed)
}

func (ingestor *Elm) ingestURL(feedUrl string) []data.PackageVersion {
	var results []data.PackageVersion

	fp := gofeed.NewParser()

	feed, err := fp.ParseURL(feedUrl)

	if err != nil {
		log.Print(err)
		return results
	}
	for _, item := range feed.Items {
		parsed, _ := url.Parse(item.Link)
		parts := strings.Split(parsed.Path, "/")
		results = append(results,
			data.PackageVersion{
				Platform:  "elm",
				Name:      fmt.Sprintf("%s/%s", parts[2], parts[3]),
				Version:   parts[4],
				CreatedAt: *item.PublishedParsed,
			})
	}
	return results
}
