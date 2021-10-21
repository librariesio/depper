# depper

> A dapper consumer of ecosystem APIs.

Depper is an ingestor of package releases from multiple ecosystems (each ecosystem is found in [ingestors/](ingestors/)).

When new package releases are found, they are pushed to a shared redis queue for [Libraries.io](https://libraries.io) to process.

#### Ingestor Types

* `ingestors.Ingestor`: these are scheduled to ingest new versions at specific intervals (`ingestor.Schedule()`).
* `ingestors.StreamingIngestor`: these are always running in a goroutine, ingesting new releases via a channel

#### Ingestor Cursor Patterns

Depper has to know where to pick up once it restarts, so there are several methods for storing such a cursor:

* `ingestors.setBookmarkTime()` + `ingestor.getBookmarkTime()` [RECOMMENDED] : reads/sets a `time.Time` to redis (persistent)
* `ingestors.setBookmark()` + `ingestor.getBookmark()`: reads/sets an arbitrary string to redis (persistent)
* `LatestRun`: reads/sets a `time.Time` on the ingestor instance (non-persistent)

# Running Locally

`go run main.go`

# Deploying

1) merge PR into `main` branch
2) `tl setenv libraries`
3) `./bin/deploy.sh`


