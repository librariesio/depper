package publishers

import (
	"log"

	"github.com/librariesio/depper/data"
)

type LoggingPublisher struct{}

func (publisher *LoggingPublisher) Publish(packageVersion data.PackageVersion) {
	log.Printf("Depper Publishing %s %s %s %s",
		packageVersion.Platform,
		packageVersion.Name,
		packageVersion.Version,
		packageVersion.CreatedAt)
}
