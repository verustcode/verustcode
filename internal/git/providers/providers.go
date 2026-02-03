// Package providers provides a centralized registry for all Git provider implementations.
// This package imports all provider implementations to trigger their init() functions,
// which register the providers in the provider.Registry.
//
// By importing this package in main.go instead of individual provider packages,
// we can add new providers without modifying the main entry point.
package providers

import (
	// Import all provider implementations to trigger their init() registration
	_ "github.com/verustcode/verustcode/internal/git/gitea"
	_ "github.com/verustcode/verustcode/internal/git/github"
	_ "github.com/verustcode/verustcode/internal/git/gitlab"
	// Add new provider imports here when implementing new providers
)
