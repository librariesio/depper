package ingestors

import (
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/buger/jsonparser"
	"github.com/librariesio/depper/data"
)

var architectures = []string{"linux-32", "linux-64", "linux-aarch64", "linux-armv6l", "linux-armv7l", "linux-ppc64le", "osx-64", "win-64", "win-32", "noarch", "zos-z"}

type CondaParser struct {
	URL      string
	Platform string
}

func NewCondaParser(url string, platform string) *CondaParser {
	return &CondaParser{
		URL:      url,
		Platform: platform,
	}
}

func (parser *CondaParser) GetPackages(lastRun time.Time) ([]data.PackageVersion, error) {
	var results []data.PackageVersion
	for _, arch := range architectures {
		response, err := http.Get(fmt.Sprintf("%s/%s/repodata.json", parser.URL, arch))
		if err != nil {
			return results, err
		}
		defer response.Body.Close()
		jsonBody, _ := io.ReadAll(response.Body)
		packages, _, _, err := jsonparser.Get(jsonBody, "packages")
		if err != nil {
			return results, err
		}

		err = jsonparser.ObjectEach(packages, func(key []byte, value []byte, dataType jsonparser.ValueType, offset int) error {
			name, _ := jsonparser.GetString(value, "name")
			version, _ := jsonparser.GetString(value, "version")
			timestamp, _ := jsonparser.GetInt(value, "timestamp")
			timeCode := time.Unix(0, timestamp*int64(time.Millisecond))
			if timeCode.Before(lastRun) {
				return nil
			}
			discoveryLag := time.Since(timeCode)

			results = append(results,
				data.PackageVersion{
					Platform:     parser.Platform,
					Name:         name,
					Version:      version,
					CreatedAt:    timeCode,
					DiscoveryLag: discoveryLag,
				})
			return nil
		})
		if err != nil {
			return results, err
		}
	}
	return results, nil
}
