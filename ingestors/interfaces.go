package ingestors

import (
	"time"

	"github.com/librariesio/depper/data"
)

type Ingestor interface {
	Schedule() string
	Ingest() []data.PackageVersion
}

type StreamingIngestor interface {
	Ingest(chan data.PackageVersion)
}

type TTLer interface {
	TTL() time.Duration
}

type Namer interface {
	Name() string
}
