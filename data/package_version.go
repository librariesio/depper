package data

import "time"

type PackageVersion struct {
	Platform  string
	Name      string
	Version   string
	CreatedAt time.Time
}
