package ingestors

import (
	"fmt"
	"github.com/librariesio/depper/data"
	log "github.com/sirupsen/logrus"
	"time"
)

type MavenIngestor struct {
	LatestRun time.Time
	FeedName  string
}

func NewMaven(feedName string) *MavenIngestor {
	return &MavenIngestor{
		FeedName: feedName,
	}
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
		log.Fatal(fmt.Sprintf("Unknown maven ingestor name: %s", ingestor.FeedName))
		return ""
	}

}
