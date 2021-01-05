package ingestors

import (
	"github.com/librariesio/depper/data"
)

type Ingestor interface {
	Schedule() string
	Ingest() []*data.PackageVersion
}
