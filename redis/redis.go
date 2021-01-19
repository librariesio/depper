package redis

import (
	"context"
	"os"

	log "github.com/sirupsen/logrus"

	redis "github.com/go-redis/redis/v8"
)

var Client *redis.Client
var Nil = redis.Nil

func init() {
	address := "localhost:6379"
	if envVal, envFound := os.LookupEnv("REDISCLOUD_URL"); envFound {
		address = envVal
	}

	Client = redis.NewClient(&redis.Options{
		Addr:     address,
		Password: "",
		DB:       0,
	})
	if err := Client.Ping(context.Background()).Err(); err != nil {
		log.Fatalf("Error connecting to redis: %s", err.Error())
	}
}
