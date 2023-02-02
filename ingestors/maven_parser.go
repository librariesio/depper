package ingestors

import (
	"encoding/json"
	"io"
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

func (parser *MavenParser) GetPackages() ([]data.PackageVersion, error) {
	var results []data.PackageVersion

	response, err := http.Get(parser.URL)
	if err != nil {
		return results, err
	}
	defer response.Body.Close()

	body, _ := io.ReadAll(response.Body)
	var mavens []mavenUpdate
	err = json.Unmarshal(body, &mavens)
	if err != nil {
		return results, err
	}

	for _, maven := range mavens {
		createdAt := time.Unix(0, maven.LastModified*int64(time.Millisecond))
		discoveryLag := time.Since(createdAt)

		results = append(results,
			data.PackageVersion{
				Platform:     parser.Platform,
				Name:         maven.Name,
				Version:      maven.Version,
				CreatedAt:    createdAt,
				DiscoveryLag: discoveryLag,
			})
	}

	return results, nil
}
