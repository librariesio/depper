package ingestors

import (
	"bufio"
	"fmt"
	"net/http"
	"net/url"
	"time"

	log "github.com/sirupsen/logrus"

	"github.com/buger/jsonparser"
	"github.com/librariesio/depper/data"
)

const golangSchedule = "15 */6 * * *"
const golangIndexUrl = "https://index.golang.org/index"

type Golang struct {
	LatestRun time.Time
}

func NewGolang() *Golang {
	return &Golang{}
}

func (ingestor *Golang) Schedule() string {
	return golangSchedule
}

func (ingestor *Golang) Ingest() []data.PackageVersion {
	// Currently the index only shows the last <=2000 package version releases from the date given. (https://proxy.golang.org/)
	oneDayAgo := url.QueryEscape(time.Now().AddDate(0, 0, -1).Format(time.RFC3339))
	url := fmt.Sprintf("%s?since=%s&limit=2000", golangIndexUrl, oneDayAgo)


	var results []data.PackageVersion

	response, err := http.Get(url)
	if err != nil {
		log.WithFields(log.Fields{"ingestor": "golang", "error": err}).Error()
		return results
	}

	defer response.Body.Close()

	scanner := bufio.NewScanner(response.Body) // Each line is valid json, but the body as a whole is not
	for scanner.Scan() {
		name, _ := jsonparser.GetString(scanner.Bytes(), "Path")
		version, _ := jsonparser.GetString(scanner.Bytes(), "Version")
		createdAt, _ := jsonparser.GetString(scanner.Bytes(), "Timestamp")
		createdAtTime, _ := time.Parse(time.RFC3339, createdAt)

		results = append(results,
			data.PackageVersion{
				Platform:  "golang",
				Name:      name,
				Version:   version,
				CreatedAt: createdAtTime,
			})
	}
	if err := scanner.Err(); err != nil {
		log.WithFields(log.Fields{"ingestor": "golang", "error": err}).Error()
	}

	ingestor.LatestRun = time.Now()

	return results
}
