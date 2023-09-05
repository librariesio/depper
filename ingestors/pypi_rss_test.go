package ingestors

import (
	"testing"
	"time"

	"github.com/mmcdole/gofeed"
)

func TestCreateUpdateItemPackageVersion(t *testing.T) {
	timeNow := time.Now()

	feedItem := gofeed.Item{
		Title:           "wow cool whoa",
		PublishedParsed: &timeNow,
	}

	result := createUpdateItemPackageVersion(&feedItem)

	if result.Name != "wow" {
		t.Errorf("expect name of %s, got %s", "wow", result.Name)
	}

	if result.Version != "cool whoa" {
		t.Errorf("expect version of %s, got %s", "cool whoa", result.Version)
	}

	if result.CreatedAt != timeNow {
		t.Errorf("expect time of %#v, got %#v", timeNow, result.CreatedAt)
	}
}
