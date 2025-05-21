package ingestors

import (
	"bufio"
	"fmt"
	"net/url"
	"time"

	log "github.com/sirupsen/logrus"

	"github.com/buger/jsonparser"
	"github.com/librariesio/depper/data"
)

const goSchedule = "2-59/5 * * * *"
const goIndexUrl = "https://index.golang.org/index"

type Go struct {
	LatestRun time.Time
}

func NewGo() *Go {
	return &Go{}
}

func (ingestor *Go) Schedule() string {
	return goSchedule
}

func (ingestor *Go) Name() string {
	return "go"
}

func (ingestor *Go) Ingest() []data.PackageVersion {
	bookmarkTime, err := getBookmarkTime(ingestor, time.Now().AddDate(0, 0, -1)) // fallback to 1 day ago
	if err != nil {
		log.WithFields(log.Fields{"ingestor": ingestor.Name(), "error": err}).Fatal()
	}

	// Currently the index only shows the last <=2000 package release from the
	// date given. (https://proxy.golang.org/)
	url := fmt.Sprintf(
		"%s?since=%s&limit=2000",
		goIndexUrl,
		url.QueryEscape(bookmarkTime.Format(time.RFC3339)),
	)

	var results []data.PackageVersion

	response, err := depperGetUrl(url, map[string]string{})
	if err != nil {
		log.WithFields(log.Fields{"ingestor": ingestor.Name(), "error": err}).Error()
		return results
	}

	defer response.Body.Close()

	scanner := bufio.NewScanner(response.Body) // Each line is valid json, but the body as a whole is not

	for scanner.Scan() {
		name, _ := jsonparser.GetString(scanner.Bytes(), "Path")
		version, _ := jsonparser.GetString(scanner.Bytes(), "Version")
		createdAt, _ := jsonparser.GetString(scanner.Bytes(), "Timestamp")
		createdAtTime, _ := time.Parse(time.RFC3339, createdAt)

		// TODO: undoing this change from 2022 and monitoring it. Pseudoversions can
		// be used legitimately by other packages, so let's monitor the traffic and
		// see if it's not too noisy.
		//
		// Avoid publishing pseudo-versions, which are revisions for which no semver tag exists.
		// if module.IsPseudoVersion(version) {
		// 	continue
		// }

		discoveryLag := time.Since(createdAtTime)

		results = append(results,
			data.PackageVersion{
				Platform:     "go",
				Name:         name,
				Version:      version,
				CreatedAt:    createdAtTime,
				DiscoveryLag: discoveryLag,
			})

		if createdAtTime.After(bookmarkTime) {
			bookmarkTime = createdAtTime
		}
	}

	if _, err := setBookmarkTime(ingestor, bookmarkTime); err != nil {
		log.WithFields(log.Fields{"ingestor": ingestor.Name()}).Fatal(err)
	}

	if err := scanner.Err(); err != nil {
		log.WithFields(log.Fields{"ingestor": ingestor.Name(), "error": err}).Error()
	}

	ingestor.LatestRun = time.Now()

	return results
}
