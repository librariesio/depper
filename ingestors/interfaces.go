package ingestors

import (
	"time"

	"github.com/librariesio/depper/data"
)

type Ingestor interface {
	Name() string
}

// Regular ingestors provide an API we can poll for changes. This polling
// is done on a regular schedule.
type PollingIngestor interface {
	Ingestor

	Schedule() string
	Ingest() []data.PackageVersion
}

type TTLer interface {
	TTL() time.Duration
}
