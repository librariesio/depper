package main

import (
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/librariesio/depper/ingestors"
	"github.com/librariesio/depper/publishers"
	"github.com/robfig/cron/v3"
)

type Depper struct {
	pipeline *publishers.Pipeline

	signalHandler chan os.Signal
}

func main() {
	fmt.Println("Starting Depper...")

	depper := &Depper{
		pipeline:      createPipeline(),
		signalHandler: make(chan os.Signal, 1),
	}

	depper.registerIngestors()
	signal.Notify(depper.signalHandler, syscall.SIGINT, syscall.SIGTERM)

	sig := <-depper.signalHandler
	signal.Stop(depper.signalHandler)
	fmt.Printf("Caught signal: %s.\n", sig)
}

func createPipeline() *publishers.Pipeline {
	pipeline := publishers.NewPipeline()
	pipeline.Register(&publishers.LoggingPublisher{})

	return pipeline
}

func (depper *Depper) registerIngestors() {
	depper.registerIngestor(&ingestors.RubyGems{})
}

func (depper *Depper) registerIngestor(ingestor ingestors.Ingestor) {
	c := cron.New()
	_, err := c.AddFunc(ingestor.Schedule(), func() {
		for _, packageVersion := range ingestor.Ingest() {
			depper.pipeline.Publish(packageVersion)
		}
	})

	if err != nil {
		log.Fatalf("Error: %v", err)
	}

	c.Start()

	// for _, packageVersion := range ingestor.Ingest() {
	// 	depper.pipeline.Publish(packageVersion)
	// }
}
