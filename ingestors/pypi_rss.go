package ingestors

/*

Load latest releases from the PyPI RSS feeds
(https://warehouse.pypa.io/api-reference/feeds.html)
and return ingestion results.

The RSS feeds provide a combination of new releases for projects, and
new packages added to PyPI. These feeds are checked regularly for updates,
and found releases are returned as ingestion events.

*/

import (
	"fmt"
	"strings"
	"time"

	log "github.com/sirupsen/logrus"

	"github.com/librariesio/depper/data"
	"github.com/mmcdole/gofeed"
)

const pyPiUpdatesFeedUrl = "https://pypi.org/rss/updates.xml"
const pyPiPackagesFeedUrl = "https://pypi.org/rss/packages.xml"
const pyPiReleasesFeedUrl = "https://pypi.org/rss/project/%s/releases.xml"

type PyPiRss struct {
	LatestRun time.Time
}

func NewPyPiRss() *PyPiRss {
	return &PyPiRss{}
}

func (ingestor *PyPiRss) Name() string {
	return "pypiRss"
}

func (ingestor *PyPiRss) Schedule() string {
	return "* * * * *"
}

func (ingestor *PyPiRss) Ingest() []data.PackageVersion {
	packages := append(
		ingestor.getUpdates(),
		ingestor.getNewPackages()...,
	)
	ingestor.LatestRun = time.Now()

	return packages
}

func createUpdateItemPackageVersion(item *gofeed.Item) data.PackageVersion {
	nameAndVersion := strings.SplitN(item.Title, " ", 2)

	return data.PackageVersion{
		Platform:     "pypi",
		Name:         nameAndVersion[0],
		Version:      nameAndVersion[1],
		CreatedAt:    *item.PublishedParsed,
		DiscoveryLag: time.Since(*item.PublishedParsed),
	}
}

// Retrieve the latest release updates
func (ingestor *PyPiRss) getUpdates() []data.PackageVersion {
	var results []data.PackageVersion

	fp := gofeed.NewParser()

	feed, err := fp.ParseURL(pyPiUpdatesFeedUrl)
	if err != nil {
		log.WithFields(log.Fields{"ingestor": ingestor.Name()}).Error(err)
		return results
	}

	for _, item := range feed.Items {
		results = append(results, createUpdateItemPackageVersion(item))
	}

	return results
}

// Retrieve the latest new PyPI packages
func (ingestor *PyPiRss) getNewPackages() []data.PackageVersion {
	var results []data.PackageVersion

	// Get the current bookmark
	bookmark, err := getBookmarkTime(ingestor, time.Now().AddDate(-1, 0, 0))
	if err != nil {
		log.WithFields(log.Fields{"ingestor": ingestor.Name()}).Fatal(err)
	}

	fp := gofeed.NewParser()

	// Get the packages feed
	feed, err := fp.ParseURL(pyPiPackagesFeedUrl)
	if err != nil {
		log.WithFields(log.Fields{"ingestor": ingestor.Name()}).Error(err)
		return results
	}

	// Get releases for items not yet seen
	for _, item := range feed.Items {
		if !item.PublishedParsed.After(bookmark) {
			continue
		}

		linkBits := strings.Split(item.Link, "/")
		packageName := linkBits[len(linkBits)-2]

		results = append(results, ingestor.getReleases(packageName)...)
	}

	if len(results) > 0 {
		if _, err := setBookmarkTime(ingestor, data.MaxCreatedAt(results)); err != nil {
			log.WithFields(log.Fields{"ingestor": ingestor.Name()}).Fatal(err)
		}
	}

	return results
}

func (ingestor *PyPiRss) getReleases(packageName string) []data.PackageVersion {
	var results []data.PackageVersion

	fp := gofeed.NewParser()

	feed, err := fp.ParseURL(fmt.Sprintf(pyPiReleasesFeedUrl, packageName))
	if err != nil {
		log.WithFields(log.Fields{"ingestor": ingestor.Name()}).Error(err)
		return results
	}

	for _, item := range feed.Items {
		results = append(results,
			data.PackageVersion{
				Platform:     "pypi",
				Name:         packageName,
				Version:      item.Title,
				CreatedAt:    *item.PublishedParsed,
				DiscoveryLag: time.Since(*item.PublishedParsed),
			})
	}

	return results
}
