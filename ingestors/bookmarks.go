package ingestors

import (
	"context"
	"fmt"
	"time"

	"github.com/librariesio/depper/redis"
)

// Use to set a string bookmark for an ingestor
func setBookmark(namer Namer, bookmark string) (string, error) {
	key := bookmarkKey(namer)

	err := redis.Client.Set(context.Background(), key, bookmark, 0).Err()
	if err != nil {
		return bookmark, fmt.Errorf("Error trying to set %s bookmark to %v - %s", key, bookmark, err)
	}

	return bookmark, nil
}

// Use to set a datetime bookmark time for an ingestor
func setBookmarkTime(namer Namer, bookmarkTime time.Time) (time.Time, error) {
	if _, err := setBookmark(namer, bookmarkTime.Format(time.RFC3339)); err != nil {
		return bookmarkTime, err
	}

	return bookmarkTime, nil
}

// Use to get a string bookmark for an ingestor
func getBookmark(namer Namer, defaultValue string) (string, error) {
	val, err := redis.Client.Get(context.Background(), bookmarkKey(namer)).Result()
	if err == redis.Nil {
		return defaultValue, nil
	} else if err != nil {
		return defaultValue, err
	} else {
		return val, nil
	}
}

// Use to get a bookmark time for an ingestor
func getBookmarkTime(namer Namer, defaultValue time.Time) (time.Time, error) {
	result, err := getBookmark(namer, defaultValue.Format(time.RFC3339))
	parsed, _ := time.Parse(time.RFC3339, result)

	return parsed, err
}

func bookmarkKey(namer Namer) string {
	return fmt.Sprintf("depper:bookmark:%s", namer.Name())
}
