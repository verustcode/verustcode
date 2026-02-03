// Package consts defines cross-module constants used throughout the application.
package consts

import (
	"sync"
	"time"
)

// ServiceName is the application service name
const ServiceName = "verustcode"

// Output format constants
const (
	// OutputFormatMarkdown represents markdown output format
	OutputFormatMarkdown = "markdown"

	// OutputFormatJSON represents JSON output format
	OutputFormatJSON = "json"
)

// Project information constants
const (
	// ProjectName is the display name of the project
	ProjectName = "VerustCode"

	// ProjectURL is the GitHub repository URL
	ProjectURL = "https://github.com/verustcode/verustcode"
)

// Build information - set via ldflags during build or programmatically
var (
	// Version is the application version
	Version = "dev"

	// BuildTime is the build timestamp
	BuildTime = "unknown"

	// GitCommit is the git commit hash
	GitCommit = "unknown"
)

// Server runtime information
var (
	startedAt   time.Time
	startedOnce sync.Once
)

// SetStartedAt records the server start time (can only be called once)
func SetStartedAt(t time.Time) {
	startedOnce.Do(func() {
		startedAt = t
	})
}

// GetStartedAt returns the server start time
func GetStartedAt() time.Time {
	return startedAt
}

// GetUptime returns the duration since server started
func GetUptime() time.Duration {
	if startedAt.IsZero() {
		return 0
	}
	return time.Since(startedAt)
}
