package ingestors

import (
	"encoding/json"
	"io/ioutil"
	"net/http"
	"time"

	"github.com/librariesio/depper/data"
)

type MavenParser struct {
	URL      string
	Platform string
}
type mavenUpdate struct {
	Name         string
	Version      string
	LastModified int64
	Size         int64
}

func NewMavenParser(url string, platform string) *MavenParser {
	return &MavenParser{
		URL:      url,
		Platform: platform,
	}
}

func (parser *MavenParser) GetPackages(lastRun time.Time) ([]data.PackageVersion, error) {
	var results []data.PackageVersion

	response, err := http.Get(parser.URL)
	if err != nil {
		return results, err
	}
	defer response.Body.Close()

	body, _ := ioutil.ReadAll(response.Body)
	var mavens []mavenUpdate
	err = json.Unmarshal(body, &mavens)
	if err != nil {
		return results, err
	}

	for _, maven := range mavens {
		createdAt := time.Unix(0, maven.LastModified*int64(time.Millisecond))

		if createdAt.After(lastRun) {
			results = append(results,
				data.PackageVersion{
					Platform:  parser.Platform,
					Name:      maven.Name,
					Version:   maven.Version,
					CreatedAt: createdAt,
				})
		}
	}

	return results, nil
}
