package data

import "time"

// The information necessary for Libraries.to to look up a project and
// retrieve additional, package manager-specific information.
type PackageVersion struct {
	Platform     string
	Name         string
	Version      string
	CreatedAt    time.Time
	DiscoveryLag time.Duration // (time of depper discovery) - (creation time, as reported by repository)
	Sequence     string        // arbitrary field for tracking the order of events and debugging
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
