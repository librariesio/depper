package data

import "time"

type PackageVersion struct {
	Platform     string
	Name         string
	Version      string
	CreatedAt    time.Time
	DiscoveryLag time.Duration // (time of depper discovery) - (creation time, as reported by repository)
}

func MaxCreatedAt(packageVersions []PackageVersion) time.Time {
	var maxCreatedAt time.Time

	for _, packageVersion := range packageVersions {
		if packageVersion.CreatedAt.After(maxCreatedAt) {
			maxCreatedAt = packageVersion.CreatedAt
		}
	}

	return maxCreatedAt
}
