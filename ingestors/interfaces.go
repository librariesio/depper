package ingestors

import (
	"time"

	"github.com/librariesio/depper/data"
)

// Regular ingestors provide an API we can poll for changes. This polling
// is done on a regular schedule.
type Ingestor interface {
	Schedule() string
	Ingest() []data.PackageVersion
}

// Streaming Ingestors continually pull new release information from a
// persistent source. NPM is an example of this, as it provides a
// CouchDB API endpoint from which we can continually pull new
// package data.
type StreamingIngestor interface {
	Ingest(chan data.PackageVersion)
}

type TTLer interface {
	TTL() time.Duration
}

type Namer interface {
	Name() string
}
