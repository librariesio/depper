package publishers

import (
	log "github.com/sirupsen/logrus"

	"github.com/librariesio/depper/data"
)

type LoggingPublisher struct{}

func (publisher *LoggingPublisher) Publish(packageVersion data.PackageVersion) {
	log.
		WithFields(log.Fields{
			"platform":     packageVersion.Platform,
			"name":         packageVersion.Name,
			"version":      packageVersion.Version,
			"created":      packageVersion.CreatedAt,
			"discoveryLag": packageVersion.DiscoveryLag.Milliseconds(),
		}).
		Info("Depper publish")
}
