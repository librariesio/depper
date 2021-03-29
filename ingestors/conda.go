package ingestors

import (
	"time"

	"github.com/librariesio/depper/data"
	log "github.com/sirupsen/logrus"
)

const condaSchedule = "*/30 * * * *"

const (
	CondaForge CondaRepository = "conda_forge"
	CondaMain  CondaRepository = "conda_main"
)

type CondaRepository string

type CondaIngestor struct {
	LatestRun  time.Time
	Repository CondaRepository
}

func NewConda(repository CondaRepository) *CondaIngestor {
	return &CondaIngestor{
		Repository: repository,
	}
}

func (ingestor *CondaIngestor) Schedule() string {
	return condaSchedule
}

func (ingestor *CondaIngestor) Name() string {
	return string(ingestor.Repository)
}

func (ingestor *CondaIngestor) Ingest() []data.PackageVersion {
	// Until we save LatestRun state, we need to set a LatestRun to avoid scanning every single release in the index.
	bookmark, err := getBookmarkTime(ingestor, time.Now().AddDate(-1, 0, 0))
	if err != nil {
		log.WithFields(log.Fields{"ingestor": ingestor.Name(), "error": err}).Fatal()
	}
	parser := ingestor.GetParser()

	results, err := parser.GetPackages(bookmark)
	if err != nil {
		log.WithFields(log.Fields{"ingestor": ingestor.Name(), "error": err}).Error()
		return results
	}
	if len(results) > 0 {
		if _, err := setBookmarkTime(ingestor, data.MaxCreatedAt(results)); err != nil {
			log.WithFields(log.Fields{"ingestor": ingestor.Name()}).Fatal(err)
		}
	}

	return results
}

func (ingestor *CondaIngestor) GetParser() *CondaParser {
	url := map[CondaRepository]string{
		CondaMain:  "https://repo.anaconda.com/pkgs/main",
		CondaForge: "https://conda.anaconda.org/conda-forge",
	}[ingestor.Repository]

	return NewCondaParser(url, ingestor.Name())
}
