package ingestors

import (
	"context"
	"fmt"

	"github.com/librariesio/depper/redis"
)

// Use to set a date/string bookmark for an ingestor
func setBookmark(namer Namer, bookmark interface{}) (interface{}, error) {
	err := redis.Client.Set(context.Background(), bookmarkKey(namer), bookmark, 0).Err()
	if err != nil {
		return bookmark, fmt.Errorf("Error trying to set %s bookmark to %v - %s", bookmark, err)
	}

	return bookmark, nil
}

// Use to get a date/string bookmark for an ingestor
func getBookmark(namer Namer, defaultValue interface{}) (interface{}, error) {
	val, err := redis.Client.Get(context.Background(), bookmarkKey(namer)).Result()
	if err == redis.Nil {
		return defaultValue, nil
	} else if err != nil {
		return defaultValue, err
	} else {
		return val, nil
	}
}

func bookmarkKey(namer Namer) string {
	return fmt.Sprintf("depper:bookmark:%s", namer.Name())
}
