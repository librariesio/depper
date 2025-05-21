package ingestors

import (
	"context"
	"net/http"
	"time"

	"github.com/mmcdole/gofeed"
)

func depperGetUrl(url string) (*http.Response, error) {
	return depperGetUrlWithHeaders(url, map[string]string{})
}

func depperGetUrlWithHeaders(url string, headers map[string]string) (*http.Response, error) {
	client := &http.Client{}
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}

	req.Header.Set("User-Agent", UserAgent)
	for key, value := range headers {
		req.Header.Set(key, value)
	}

	return client.Do(req)
}

func depperGetFeed(url string) (feed *gofeed.Feed, err error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	fp := gofeed.NewParser()
	fp.UserAgent = UserAgent

	return fp.ParseURLWithContext(url, ctx)
}
