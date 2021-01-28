package publishers

import (
	"context"
	"time"

	"github.com/librariesio/depper/data"
	"github.com/librariesio/depper/redis"
	log "github.com/sirupsen/logrus"
)

const maxQueueSize = 1000

type Pipeline struct {
	publishers      []Publisher
	LastPublishedAt time.Time
	queue           chan publishing
}

func NewPipeline() *Pipeline {
	pipeline := &Pipeline{}
	go pipeline.run()

	return pipeline
}

func (pipeline *Pipeline) Publish(ttl time.Duration, packageVersion data.PackageVersion) {
	pipeline.queue <- publishing{PackageVersion: packageVersion, ttl: ttl}
}

func (pipeline *Pipeline) run() {
	pipeline.queue = make(chan publishing, maxQueueSize)

	for publishing := range pipeline.queue {
		pipeline.process(publishing)
	}
}

func (pipeline *Pipeline) process(publishing publishing) {
	if !pipeline.shouldPublish(publishing) {
		return
	}

	for _, publisher := range pipeline.publishers {
		// Publish each packageversion to all publishers
		publisher.Publish(publishing.PackageVersion)
	}
}

func (pipeline *Pipeline) shouldPublish(publishing publishing) bool {
	wasSet, err := redis.Client.SetNX(context.Background(), publishing.Key(), true, publishing.ttl).Result()

	if err != nil {
		log.WithFields(log.Fields{"publisher": "pipeline"}).Error(err)
		return false
	}

	return wasSet
}

// Registers a publisher on the pipeline
func (pipeline *Pipeline) Register(publisher Publisher) {
	pipeline.publishers = append(pipeline.publishers, publisher)
}
