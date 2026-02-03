package output

import (
	"fmt"
	"strings"

	"github.com/verustcode/verustcode/internal/config"
)

// BuildMetadataString builds the metadata string based on configuration.
// It formats agent name, model name, and custom text according to the metadata config.
// Returns an empty string if no metadata should be displayed.
func BuildMetadataString(cfg *config.OutputMetadataConfig, agentName, modelName string) string {
	if cfg == nil {
		return ""
	}

	var parts []string

	// Add custom text first if provided
	if cfg.CustomText != "" {
		parts = append(parts, cfg.CustomText)
	}

	// Add agent name if enabled
	if cfg.ShowAgent == nil || *cfg.ShowAgent {
		if agentName != "" {
			parts = append(parts, fmt.Sprintf("Agent: %s", agentName))
		}
	}

	// Add model name if enabled
	if cfg.ShowModel == nil || *cfg.ShowModel {
		if modelName != "" {
			parts = append(parts, fmt.Sprintf("Model: %s", modelName))
		}
	}

	if len(parts) == 0 {
		return ""
	}

	return "*" + strings.Join(parts, " || ") + "*"
}
