# depper

> A dapper consumer of ecosystem APIs.

Depper is an ingestor of package releases from multiple ecosystems (each ecosystem is found in [ingestors/](ingestors/)).

When new package releases are found, they are pushed to a shared redis queue for [Libraries.io](https://libraries.io) to process.

## Ingestor interface

Ingestors must satisfy the `ingestors.PollingIngestor` interface. It is currently our only ingestor interface, and
schedules ingestion of new versions at specific intervals (`ingestor.Schedule()`).

## TTLer interface

By default a unique `PackageVersion` will get limited to one published event per 24 hours. This duration can be overridden by
implementing the TTLer interface.

## Ingestor Cursor Patterns

Depper has to know where to pick up once it restarts, so there are several methods for storing such a cursor:

- `ingestors.setBookmarkTime()` + `ingestor.getBookmarkTime()` [RECOMMENDED] : reads/sets a `time.Time` to redis (persistent)
- `ingestors.setBookmark()` + `ingestor.getBookmark()`: reads/sets an arbitrary string to redis (persistent)
- `LatestRun`: reads/sets a `time.Time` on the ingestor instance (non-persistent)

## Running Locally

`go run main.go`

## Running Tests

`go test -v ./...`

## Running the Linter

You'll need the same version of our linter as CI, so reference the `".circleci/config.yml"` for the installation command.

`golangci-lint run`: this will run the linter.

`golangci-lint run --fix`: this will run the linter and autofix any autofix-able linter errors.

## Deploying

1. merge PR into `main` branch
2. `tl setenv libraries`
3. `./bin/deploy.sh`
