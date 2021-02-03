package ingestors

import (
	"github.com/librariesio/depper/data"
	log "github.com/sirupsen/logrus"
	"time"
)

const mavenAtlassianSchedule = "@every 1h"

const mavenAtlassianUrl = "https://maven.libraries.io/atlassian/recent"

type MavenAtlassian struct {
	LatestRun time.Time
}

func NewMavenAtlassian() *MavenAtlassian {
	return &MavenAtlassian{}
}

func (ingestor *MavenAtlassian) Schedule() string {
	return mavenAtlassianSchedule
}

func (ingestor *MavenAtlassian) Ingest() []data.PackageVersion {
	mp := NewMavenParser(mavenAtlassianUrl, "maven_atlassian")
	results, err := mp.GetPackages()
	if err != nil {
		log.WithFields(log.Fields{"ingestor": "maven_atlassian", "error": err}).Error()
		return results
	}
	ingestor.LatestRun = time.Now()
	return results
}
