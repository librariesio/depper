package ingestors

import (
	"fmt"
	"io"
	"strconv"

	"github.com/buger/jsonparser"
	log "github.com/sirupsen/logrus"

	"github.com/librariesio/depper/data"
)

const npmSchedule = "*/5 * * * *"
const npmIndexUrl = "https://replicate.npmjs.com/registry"
const npmChangesUrl = npmIndexUrl + "/_changes"

// Current limit is 1 page per run, but if we need to do a backfill or catch up
// we could increase this to > 1 pages.
const pages = 1
const perPage = 10000

type NPM struct {
}

func NewNPM() *NPM {
	return &NPM{}
}

func (ingestor *NPM) Schedule() string {
	return npmSchedule
}

func (ingestor *NPM) Name() string {
	return "npm"
}

func (ingestor *NPM) Ingest() []data.PackageVersion {
	currentSequence := ingestor.getCurrentSequence()

	var results []data.PackageVersion
	for page := 0; page < pages; page++ {
		lastSequence, lastResults := ingestor.getPage(currentSequence)
		if len(lastResults) == 0 {
			break
		}
		results = append(results, lastResults...)

		if lastSequence > currentSequence {
			currentSequence = lastSequence
		}
	}

	if _, err := setBookmark(ingestor, strconv.FormatInt(currentSequence, 10)); err != nil {
		log.WithFields(log.Fields{"ingestor": ingestor.Name()}).Fatal(err)
	}

	return results
}

func (ingestor *NPM) getPage(sequence int64) (int64, []data.PackageVersion) {
	var results []data.PackageVersion

	// The header enables the new API changes and can be removed May 29th, 2025:
	// https://github.blog/changelog/2025-02-27-changes-and-deprecation-notice-for-npm-replication-apis/
	response, err := depperGetUrlWithHeaders(
		fmt.Sprintf("%s?since=%d&limit=%d", npmChangesUrl, sequence, perPage),
		map[string]string{"npm-replication-opt-in": "true"},
	)
	if err != nil {
		log.WithFields(log.Fields{"ingestor": ingestor.Name(), "error": err}).Error()
		return sequence, results
	}
	defer response.Body.Close()

	body, _ := io.ReadAll(response.Body)
	sequenceList, _, _, _ := jsonparser.Get(body, "results")
	_, _ = jsonparser.ArrayEach(sequenceList, func(value []byte, dataType jsonparser.ValueType, offset int, err error) {
		name, _ := jsonparser.GetString(value, "id")
		seq, _ := jsonparser.GetInt(value, "seq")

		// The new NPM feed only provides names and sequences, so we don't get Version, CreatedAt or DiscoveryLag.
		results = append(results,
			data.PackageVersion{
				Platform: "npm",
				Name:     name,
				Sequence: strconv.FormatInt(seq, 10),
			})
	})
	lastSequence, err := jsonparser.GetInt(body, "last_seq")
	if err != nil {
		log.WithFields(log.Fields{"ingestor": ingestor.Name(), "error": err}).Error()
		lastSequence = sequence
	}

	return lastSequence, results
}

func (ingestor *NPM) getCurrentSequence() int64 {
	bookmark, err := getBookmark(ingestor, "")
	if err != nil {
		log.WithFields(log.Fields{"ingestor": ingestor.Name(), "error": err}).Fatal()
	}

	var currentSequence int64
	if bookmark != "" {
		currentSequence, _ = strconv.ParseInt(bookmark, 10, 64)
	} else if currentSequence == 0 {
		currentSequence = ingestor.getLatestSequence()
		log.WithFields(log.Fields{"ingestor": ingestor.Name(), "msg": fmt.Sprintf("No NPM bookmark saved, using latest published sequence %d", currentSequence)}).Info()
	}

	return currentSequence
}

// As a fallback, fetch the latest published sequence from https://replicate.npmjs.com/registry/.
func (ingestor *NPM) getLatestSequence() int64 {
	response, err := depperGetUrl(npmIndexUrl)
	if err != nil {
		log.WithFields(log.Fields{"ingestor": ingestor.Name(), "error": err}).Fatal()
	}
	defer response.Body.Close()
	body, _ := io.ReadAll(response.Body)
	latestSequence, err := jsonparser.GetInt(body, "update_seq")
	if err != nil {
		log.WithFields(log.Fields{"ingestor": ingestor.Name(), "error": err}).Fatal()
	}

	return latestSequence
}
