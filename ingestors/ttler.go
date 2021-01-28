package ingestors

import "time"

type TTLer interface {
	TTLHours() time.Duration
}
