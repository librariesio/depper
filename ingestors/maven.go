package ingestors

import (
	"fmt"
	"github.com/librariesio/depper/data"
	log "github.com/sirupsen/logrus"
	"time"
)

const ttl = 168 * time.Hour // 1 Week

type MavenIngestor struct {
	LatestRun time.Time
	FeedName  string
}

func NewMaven(feedName string) *MavenIngestor {
	maven := &MavenIngestor{
		FeedName: feedName,
	}
	if maven.Name() == "" {
		log.Fatalf("Unknown maven ingestor name: %s", feedName)
	}
	return maven
}

func (ingestor *MavenIngestor) Schedule() string {
	return ingestor.getSchedule()
}

func (ingestor *MavenIngestor) Ingest() []data.PackageVersion {
	mp := NewMavenParser(fmt.Sprintf("https://maven.libraries.io/%s/recent", ingestor.FeedName), ingestor.Name())
	results, err := mp.GetPackages()
	if err != nil {
		log.WithFields(log.Fields{"ingestor": ingestor.Name(), "error": err}).Error()
		return results
	}
	ingestor.LatestRun = time.Now()
	return results
}
func (ingestor *MavenIngestor) TTL() time.Duration {
	return ttl
}

func (ingestor *MavenIngestor) getSchedule() string {
	switch ingestor.FeedName {
	case "mavenCentral":
		return "@every 12h"
	case "atlassian":
		return "@every 1h"
	case "hortonworks":
		return "@every 1h"
	case "springLibsRelease":
		return "@every 6h"
	default:
		return "@every 10h"
	}
}
func (ingestor *MavenIngestor) Name() string {
	switch ingestor.FeedName {
	case "mavenCentral":
		return "maven_mavencentral"
	case "atlassian":
		return "maven_atlassian"
	case "hortonworks":
		return "maven_hortonworks"
	case "springLibsRelease":
		return "maven_springlibs"
	default:
		return ""
	}

}
