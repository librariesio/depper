package ingestors

import (
	"github.com/librariesio/depper/data"
)

type StreamingIngestor interface {
	Ingest(chan data.PackageVersion)
}
