package publishers

import "github.com/librariesio/depper/data"

type Publisher interface {
	Publish(data.PackageVersion)
}
