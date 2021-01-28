package publishers

import (
	"fmt"
	"time"

	"github.com/librariesio/depper/data"
)

type publishing struct {
	data.PackageVersion
	ttl time.Duration
}

func (p *publishing) Key() string {
	return fmt.Sprintf("depper:ingest:%s:%s:%s", p.Platform, p.Name, p.Version)
}
