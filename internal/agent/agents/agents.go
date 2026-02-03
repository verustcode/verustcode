// Package agents provides a centralized registry for all Agent implementations.
// This package imports all agent implementations to trigger their init() functions,
// which register the agents in the base.Registry.
//
// By importing this package in main.go instead of individual agent packages,
// we can add new agents without modifying the main entry point.
package agents

import (
	// Import all agent implementations to trigger their init() registration
	_ "github.com/verustcode/verustcode/internal/agent/cursor"
	_ "github.com/verustcode/verustcode/internal/agent/gemini"
	_ "github.com/verustcode/verustcode/internal/agent/mock"
	_ "github.com/verustcode/verustcode/internal/agent/qoder"
	// Add new agent imports here when implementing new agents
)
