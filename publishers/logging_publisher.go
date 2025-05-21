package publishers

import (
	log "github.com/sirupsen/logrus"

	"github.com/librariesio/depper/data"
)

type LoggingPublisher struct{}

func (publisher *LoggingPublisher) Publish(packageVersion data.PackageVersion) {
	field := log.Fields{
		"platform":     packageVersion.Platform,
		"name":         packageVersion.Name,
		"version":      packageVersion.Version,
		"created":      packageVersion.CreatedAt,
		"discoveryLag": packageVersion.DiscoveryLag.Milliseconds(),
	}

	if packageVersion.Sequence != "" {
		field["sequence"] = packageVersion.Sequence
	}

	log.
		WithFields(field).
		Info("Depper publish")
}
