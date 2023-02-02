package ingestors

import (
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/PuerkitoBio/goquery"
	log "github.com/sirupsen/logrus"

	"github.com/librariesio/depper/data"
	"github.com/mmcdole/gofeed"
)

const drupalSchedule = "0 */4 * * *"
const drupalModulesUrl = "https://www.drupal.org/project/project_module?page=%d&solrsort=ds_project_latest_release+desc"
const drupalReleasesUrl = "https://www.drupal.org/node/%s/release/feed"

type Drupal struct {
	LatestRun time.Time
}

func NewDrupal() *Drupal {
	return &Drupal{}
}

func (ingestor *Drupal) Schedule() string {
	return drupalSchedule
}

func (ingestor *Drupal) Name() string {
	return "packagist_drupal"
}

func (ingestor *Drupal) Ingest() []data.PackageVersion {
	var results []data.PackageVersion

	bookmark, err := getBookmarkTime(ingestor, time.Now().AddDate(-1, 0, 0))
	if err != nil {
		log.WithFields(log.Fields{"ingestor": ingestor.Name(), "error": err}).Fatal()
	}

	page := 0
	done := false
	// 100 is an arbitrary limit to ensure we don't scrape all ~2k pages of packages
	for page < 100 && !done {
		doc, err := getHtmlDocument(fmt.Sprintf(drupalModulesUrl, page))
		if err != nil {
			log.WithFields(log.Fields{"ingestor": ingestor.Name(), "error": err}).Fatal()
		}

		doc.Find(".node-project-module").Each(func(i int, s *goquery.Selection) {
			if !done {
				var id string
				if idAttr, exists := s.Attr("id"); exists {
					parts := strings.SplitN(idAttr, "-", 2) // e.g. "node-1234"
					id = parts[1]
				}
				packageResults := ingestor.getVersions(id, bookmark)
				if len(packageResults) == 0 { // last page didn't have any new versions, which means we don't have to keep looking at older packages
					done = true
				} else {
					results = append(results, packageResults...)
				}
			}
		})
		page++
		time.Sleep(100 * time.Millisecond)
	}

	if len(results) > 0 {
		if _, err := setBookmarkTime(ingestor, data.MaxCreatedAt(results)); err != nil {
			log.WithFields(log.Fields{"ingestor": ingestor.Name()}).Fatal(err)
		}
	}

	return results
}

func (ingestor *Drupal) getVersions(id string, bookmark time.Time) []data.PackageVersion {
	var results []data.PackageVersion
	fp := gofeed.NewParser()

	feed, err := fp.ParseURL(fmt.Sprintf(drupalReleasesUrl, id))
	if err != nil {
		log.WithFields(log.Fields{"ingestor": ingestor.Name()}).Error(err)
		return results
	}

	for _, item := range feed.Items {
		createdAtTime, _ := time.Parse(time.RFC1123, item.Published)
		nameAndVersion := strings.SplitN(item.Title, " ", 2) // e.g. ctools 7.x-1.19
		if createdAtTime.After(bookmark) {
			discoveryLag := time.Since(createdAtTime)
			results = append(results,
				data.PackageVersion{
					Platform:     ingestor.Name(),
					Name:         nameAndVersion[0],
					Version:      nameAndVersion[1],
					CreatedAt:    createdAtTime,
					DiscoveryLag: discoveryLag,
				})
		}
	}

	return results
}

func getHtmlDocument(url string) (*goquery.Document, error) {
	res, err := http.Get(url)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()
	if res.StatusCode != 200 {
		return nil, fmt.Errorf("Status code error for %s: %d %s", url, res.StatusCode, res.Status)
	}

	doc, err := goquery.NewDocumentFromReader(res.Body)
	if err != nil {
		return nil, err
	}

	return doc, nil
}
