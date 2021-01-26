package maven

import (
	"encoding/json"
	"github.com/librariesio/depper/data"
	"io/ioutil"
	"net/http"
	"time"
)

type MavenParser struct {
	URL      string
	Platform string
}
type mavenUpdate struct {
	Name    string
	Version string
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

	body, _ := ioutil.ReadAll(response.Body)
	var mavens []mavenUpdate
	json.Unmarshal(body, &mavens)

	for _, maven := range mavens {
		results = append(results,
			data.PackageVersion{
				Platform:  parser.Platform,
				Name:      maven.Name,
				Version:   maven.Version,
				CreatedAt: time.Now(),
			})
	}

	return results, nil
}
