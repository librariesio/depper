package main

import (
	"io/ioutil"
	"os"
	"os/signal"
	"syscall"

	"github.com/librariesio/depper/data"
	"github.com/librariesio/depper/ingestors"
	"github.com/librariesio/depper/publishers"
	"github.com/robfig/cron/v3"
	log "github.com/sirupsen/logrus"
	"github.com/sirupsen/logrus/hooks/writer"
)

type Depper struct {
	pipeline           *publishers.Pipeline
	signalHandler      chan os.Signal
	streamingIngestors []*ingestors.StreamingIngestor
}

func main() {
	setupLogger()
	log.Info("Starting Depper")
	depper := &Depper{
		pipeline:      createPipeline(),
		signalHandler: make(chan os.Signal, 1),
	}
	depper.registerIngestors()

	signal.Notify(depper.signalHandler, syscall.SIGINT, syscall.SIGTERM)
	sig := <-depper.signalHandler
	signal.Stop(depper.signalHandler)
	log.WithFields(log.Fields{"signal": sig}).Info("Exiting")
}

func createPipeline() *publishers.Pipeline {
	pipeline := publishers.NewPipeline()
	pipeline.Register(&publishers.LoggingPublisher{})
	pipeline.Register(publishers.NewSidekiq())
	return pipeline
}

func (depper *Depper) registerIngestors() {
	depper.registerIngestor(ingestors.NewRubyGems())
	depper.registerIngestorStream(ingestors.NewNPM())
	depper.registerIngestor(ingestors.NewElm())
	depper.registerIngestor(ingestors.NewGo())
	depper.registerIngestor(ingestors.NewMavenCentral())
}

func (depper *Depper) registerIngestor(ingestor ingestors.Ingestor) {
	c := cron.New()
	ingestAndPublish := func() {
		for _, packageVersion := range ingestor.Ingest() {
			depper.pipeline.Publish(packageVersion)
		}
	}

	_, err := c.AddFunc(ingestor.Schedule(), ingestAndPublish)
	if err != nil {
		log.Fatal(err)
	}

	c.Start()

	// For now we'll run once upon registration
	ingestAndPublish()
}

func (depper *Depper) registerIngestorStream(ingestor ingestors.StreamingIngestor) {
	depper.streamingIngestors = append(depper.streamingIngestors, &ingestor)

	// Unbuffered channel so that the StreamingIngestor will block while pulling
	// next updates until Publish() has grabbed the last one.
	packageVersions := make(chan data.PackageVersion)

	go ingestor.Ingest(packageVersions)
	go func() {
		for packageVersion := range packageVersions {
			depper.pipeline.Publish(packageVersion)
		}
	}()
}

func setupLogger() {
	log.SetFormatter(&log.TextFormatter{
		FullTimestamp: true,
	})

	// Send error-y logs to stderr and info-y logs to stdout
	log.SetOutput(ioutil.Discard)
	log.AddHook(&writer.Hook{
		Writer: os.Stderr,
		LogLevels: []log.Level{
			log.PanicLevel,
			log.FatalLevel,
			log.ErrorLevel,
			log.WarnLevel,
		},
	})
	log.AddHook(&writer.Hook{
		Writer: os.Stdout,
		LogLevels: []log.Level{
			log.InfoLevel,
			log.DebugLevel,
		},
	})

	if os.Getenv("DEBUG") == "1" {
		log.SetLevel(log.DebugLevel)
	} else {
		log.SetLevel(log.InfoLevel)
	}
}
