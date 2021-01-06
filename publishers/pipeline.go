package publishers

import (
	"time"

	"github.com/librariesio/depper/data"
)

const maxQueueSize = 1000

type Publisher interface {
	Publish(data.PackageVersion)
}

type Pipeline struct {
	publishers      []Publisher
	LastPublishedAt time.Time
	queue           chan data.PackageVersion
}

func NewPipeline() *Pipeline {
	pipeline := &Pipeline{}
	go pipeline.run()

	return pipeline
}

func (pipeline *Pipeline) Publish(packageVersion data.PackageVersion) {
	pipeline.queue <- packageVersion
}

func (pipeline *Pipeline) run() {
	pipeline.queue = make(chan data.PackageVersion, maxQueueSize)

	for packageVersion := range pipeline.queue {
		pipeline.process(packageVersion)
	}
}

func (pipeline *Pipeline) process(packageVersion data.PackageVersion) {
	// TODO move deduping code here
	for _, publisher := range pipeline.publishers {
		// Publish each packageversion to all publishers
		publisher.Publish(packageVersion)
	}
}

// Registers a publisher on the pipeline
func (pipeline *Pipeline) Register(publisher Publisher) {
	pipeline.publishers = append(pipeline.publishers, publisher)
}
