package main

import (
	"io"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/librariesio/depper/data"
	"github.com/librariesio/depper/ingestors"
	"github.com/librariesio/depper/publishers"
	"github.com/librariesio/depper/redis"
	"github.com/robfig/cron/v3"
	"github.com/sirupsen/logrus/hooks/writer"

	logrus_bugsnag "github.com/Shopify/logrus-bugsnag"
	bugsnag "github.com/bugsnag/bugsnag-go"
	log "github.com/sirupsen/logrus"
	"gopkg.in/DataDog/dd-trace-go.v1/ddtrace/tracer"
)

const defaultTTL = 24 * time.Hour

type Depper struct {
	// Place onto which jobs are placed for Libraries.io to further examine a package manager's package
	pipeline           *publishers.Pipeline
	signalHandler      chan os.Signal
	streamingIngestors []*ingestors.StreamingIngestor
}

func waitForExitSignal(signalHandler chan os.Signal) os.Signal {
	signal.Notify(signalHandler, syscall.SIGINT, syscall.SIGTERM)
	sig := <-signalHandler
	signal.Stop(signalHandler)

	return sig
}

func main() {
	defer func() {
		// This defer will run when SIGINT is caught, but not for SIGKILL/SIGTERM/SIGHUP/SIGSTOP or os.Exit().
		log.Info("Stopping Depper")
	}()

	if val := os.Getenv("DD_AGENT_HOST"); val != "" {
		log.Info("Connecting to Datadog")
		tracer.Start(
			tracer.WithService(os.Getenv("DD_SERVICE")),
			// The DD agent already does sampling, but sampling on the client will help reduce overhead in Go.
			// tracer.WithSampler(tracer.NewRateSampler(0.1)),
		)
		defer tracer.Stop()
	}

	setupLogger()
	redis.Connect()

	log.Info("Starting Depper")
	depper := &Depper{
		pipeline:      createPipeline(),
		signalHandler: make(chan os.Signal, 1),
	}
	depper.registerIngestors()

	sig := waitForExitSignal(depper.signalHandler)

	log.WithFields(log.Fields{"signal": sig}).Info("Exiting")
}

func createPipeline() *publishers.Pipeline {
	pipeline := publishers.NewPipeline()
	pipeline.Register(&publishers.LoggingPublisher{})
	pipeline.Register(publishers.NewSidekiq())
	return pipeline
}

func (depper *Depper) registerIngestors() {
	depper.registerIngestor(ingestors.NewCocoaPods())
	depper.registerIngestor(ingestors.NewRubyGems())
	depper.registerIngestor(ingestors.NewElm())
	depper.registerIngestor(ingestors.NewGo())
	depper.registerIngestor(ingestors.NewMaven(ingestors.MavenCentral))
	depper.registerIngestor(ingestors.NewMaven(ingestors.GoogleMaven))
	depper.registerIngestor(ingestors.NewCargo())
	depper.registerIngestor(ingestors.NewNuget())
	depper.registerIngestor(ingestors.NewPackagist())
	depper.registerIngestor(ingestors.NewCPAN())
	depper.registerIngestor(ingestors.NewHex())
	depper.registerIngestor(ingestors.NewHackage())
	depper.registerIngestor(ingestors.NewPub())
	// drupal is returning a 403 as of 2024-05-06. need to investigate but let's
	// not block other ingestion as we do
	//	depper.registerIngestor(ingestors.NewDrupal())
	depper.registerIngestor(ingestors.NewPyPiRss())
	depper.registerIngestor(ingestors.NewPyPiXmlRpc())
	depper.registerIngestor(ingestors.NewConda(ingestors.CondaForge))
	depper.registerIngestor(ingestors.NewConda(ingestors.CondaMain))

	depper.registerIngestorStream(ingestors.NewNPM())
}

func (depper *Depper) registerIngestor(ingestor ingestors.Ingestor) {
	c := cron.New()
	ingestAndPublish := func() {
		ttl := defaultTTL

		if ttler, ok := ingestor.(ingestors.TTLer); ok {
			ttl = ttler.TTL()
		}

		span := tracer.StartSpan("ingest")
		// TODO: add ingestor name to span here
		// span.SetTag(ingestor.Name())
		packageVersions := ingestor.Ingest()
		span.Finish()

		for _, packageVersion := range packageVersions {
			depper.pipeline.Publish(ttl, packageVersion)
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
			depper.pipeline.Publish(defaultTTL, packageVersion)
		}
	}()
}

func setupLogger() {
	log.SetFormatter(&log.TextFormatter{
		FullTimestamp: true,
		ForceQuote:    true,
	})

	// Configure bugsnag
	bugsnag.Configure(bugsnag.Configuration{
		APIKey:          os.Getenv("BUGSNAG_API_KEY"),
		AppVersion:      os.Getenv("GIT_COMMIT"),
		ProjectPackages: []string{"main", "github.com/librariesio/depper"},
	})
	hook, _ := logrus_bugsnag.NewBugsnagHook()
	log.AddHook(hook)

	// Send error-y logs to stderr and info-y logs to stdout
	log.SetOutput(io.Discard)
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
