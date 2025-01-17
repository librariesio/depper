package ingestors

import (
	"context"
	"fmt"
	"time"

	"github.com/librariesio/depper/redis"
)

// Use to set a string bookmark for an ingestor
func setBookmark(ingestor Ingestor, bookmark string) (string, error) {
	key := bookmarkKey(ingestor)

	err := redis.Client.Set(context.Background(), key, bookmark, 0).Err()
	if err != nil {
		return bookmark, fmt.Errorf("Error trying to set %s bookmark to %v - %s", key, bookmark, err)
	}

	return bookmark, nil
}

// Use to set a datetime bookmark time for an ingestor
func setBookmarkTime(ingestor Ingestor, bookmarkTime time.Time) (time.Time, error) {
	if _, err := setBookmark(ingestor, bookmarkTime.Format(time.RFC3339)); err != nil {
		return bookmarkTime, err
	}

	return bookmarkTime, nil
}

// Use to get a string bookmark for an ingestor
func getBookmark(ingestor Ingestor, defaultValue string) (string, error) {
	val, err := redis.Client.Get(context.Background(), bookmarkKey(ingestor)).Result()
	if err == redis.Nil {
		return defaultValue, nil
	} else if err != nil {
		return defaultValue, err
	} else {
		return val, nil
	}
}

// Use to get a bookmark time for an ingestor
func getBookmarkTime(ingestor Ingestor, defaultValue time.Time) (time.Time, error) {
	result, err := getBookmark(ingestor, defaultValue.Format(time.RFC3339))
	parsed, _ := time.Parse(time.RFC3339, result)

	return parsed, err
}

func bookmarkKey(ingestor Ingestor) string {
	return fmt.Sprintf("depper:bookmark:%s", ingestor.Name())
}
