// Package engine provides the core review engine for VerustCode.
// This file defines the interfaces used for dependency injection between engine sub-modules.
package engine

import (
	"github.com/verustcode/verustcode/internal/agent/base"
	"github.com/verustcode/verustcode/internal/git/provider"
)

// TaskEnqueuer allows components to enqueue tasks to the review queue.
// Used by RecoveryService and other components that need to submit tasks.
type TaskEnqueuer interface {
	// Enqueue adds a task to the queue for its repository.
	// Returns true if the task was successfully enqueued.
	Enqueue(task *Task) bool

	// EnqueueAsRunning adds a task to the queue and marks its repo as running.
	// Used for recovery scenarios where tasks were already in progress.
	// Returns true if the task was successfully enqueued.
	EnqueueAsRunning(task *Task) bool
}

// ProviderResolver provides access to git providers.
// Used by components that need to interact with git hosting services.
type ProviderResolver interface {
	// Get returns a provider by name.
	// Returns nil if the provider is not found.
	Get(name string) provider.Provider

	// DetectFromURL detects the provider name from a repository URL.
	// Returns empty string if the provider cannot be detected.
	DetectFromURL(url string) string
}

// AgentResolver provides access to AI agents.
// Used by components that need to execute AI-powered reviews.
type AgentResolver interface {
	// Get returns an agent by name.
	// Returns nil if the agent is not found.
	Get(name string) base.Agent
}
