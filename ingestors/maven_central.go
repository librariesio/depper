package ingestors

import (
	"time"

	"github.com/librariesio/depper/data"
	log "github.com/sirupsen/logrus"
)

const mavenCentralSchedule = "@every 12h"
const ttlHours = 168 * time.Hour // 1 Week

const mavenCentralUrl = "https://maven.libraries.io/mavenCentral/recent"

type MavenCentral struct {
	LatestRun time.Time
}

func NewMavenCentral() *MavenCentral {
	return &MavenCentral{}
}

func (ingestor *MavenCentral) Schedule() string {
	return mavenCentralSchedule
}

func (ingestor *MavenCentral) TTLHours() time.Duration {
	return ttlHours
}

func (ingestor *MavenCentral) Ingest() []data.PackageVersion {
	mp := NewMavenParser(mavenCentralUrl, "maven_mavencentral")
	results, err := mp.GetPackages()
	if err != nil {
		log.WithFields(log.Fields{"ingestor": "maven_mavencentral", "error": err}).Error()
		return results
	}
	ingestor.LatestRun = time.Now()
	return results
}
