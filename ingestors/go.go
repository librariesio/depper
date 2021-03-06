package ingestors

import (
	"bufio"
	"fmt"
	"net/http"
	"net/url"
	"regexp"
	"time"

	log "github.com/sirupsen/logrus"

	"github.com/buger/jsonparser"
	"github.com/librariesio/depper/data"
)

const goSchedule = "2-59/5 * * * *"
const goIndexUrl = "https://index.golang.org/index"
const vendoredVersionRegexp = `^v0.0.0\-\d{14}\-[0-9a-f]{12}$` // We'll avoid publishing vendored modules

type Go struct {
	LatestRun time.Time
}

func NewGo() *Go {
	return &Go{}
}

func (ingestor *Go) Schedule() string {
	return goSchedule
}

func (ingestor *Go) Ingest() []data.PackageVersion {
	// Currently the index only shows the last <=2000 package version releases from the date given. (https://proxy.golang.org/)
	oneDayAgo := url.QueryEscape(time.Now().AddDate(0, 0, -1).Format(time.RFC3339))
	url := fmt.Sprintf("%s?since=%s&limit=2000", goIndexUrl, oneDayAgo)

	var results []data.PackageVersion

	response, err := http.Get(url)
	if err != nil {
		log.WithFields(log.Fields{"ingestor": "go", "error": err}).Error()
		return results
	}

	defer response.Body.Close()

	re, _ := regexp.Compile(vendoredVersionRegexp)
	scanner := bufio.NewScanner(response.Body) // Each line is valid json, but the body as a whole is not

	for scanner.Scan() {
		name, _ := jsonparser.GetString(scanner.Bytes(), "Path")
		version, _ := jsonparser.GetString(scanner.Bytes(), "Version")
		createdAt, _ := jsonparser.GetString(scanner.Bytes(), "Timestamp")
		createdAtTime, _ := time.Parse(time.RFC3339, createdAt)

		if re.MatchString(version) {
			continue
		}

		results = append(results,
			data.PackageVersion{
				Platform:  "go",
				Name:      name,
				Version:   version,
				CreatedAt: createdAtTime,
			})
	}
	if err := scanner.Err(); err != nil {
		log.WithFields(log.Fields{"ingestor": "go", "error": err}).Error()
	}

	ingestor.LatestRun = time.Now()

	return results
}
