package ingestors

import "time"

type TTLer interface {
	TTL() time.Duration
}
