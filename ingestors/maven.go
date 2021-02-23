package ingestors

import (
	"time"

	"github.com/librariesio/depper/data"
	log "github.com/sirupsen/logrus"
)

const mavenSchedule = "@every 1h"
const mavenTTL = 240 * time.Hour // 10 days
const (
	MavenAtlassian   MavenRepository = "maven_atlassian"
	MavenHortonworks MavenRepository = "maven_hortonworks"
	MavenCentral     MavenRepository = "maven_mavencentral"
	MavenSpringlibs  MavenRepository = "maven_springlibs"
)

type MavenRepository string

type MavenIngestor struct {
	LatestRun  time.Time
	Repository MavenRepository
}

func NewMaven(repository MavenRepository) *MavenIngestor {
	return &MavenIngestor{
		Repository: repository,
	}
}

func (ingestor *MavenIngestor) Schedule() string {
	return mavenSchedule
}

func (ingestor *MavenIngestor) Ingest() []data.PackageVersion {
	parser := ingestor.GetParser()

	results, err := parser.GetPackages()
	if err != nil {
		log.WithFields(log.Fields{"ingestor": ingestor.Name(), "error": err}).Error()
		return results
	}
	ingestor.LatestRun = time.Now()
	return results
}

func (ingestor *MavenIngestor) TTL() time.Duration {
	return mavenTTL
}

func (ingestor *MavenIngestor) Name() string {
	return string(ingestor.Repository)
}

func (ingestor *MavenIngestor) GetParser() *MavenParser {
	url := map[MavenRepository]string{
		MavenAtlassian:   "https://maven.libraries.io/atlassian/recent",
		MavenHortonworks: "https://maven.libraries.io/hortonworks/recent",
		MavenCentral:     "https://maven.libraries.io/mavenCentral/recent",
		MavenSpringlibs:  "https://maven.libraries.io/springLibsRelease/recent",
	}[ingestor.Repository]

	return NewMavenParser(url, ingestor.Name())
}
