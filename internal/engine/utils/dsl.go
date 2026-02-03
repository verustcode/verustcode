// Package utils provides utility functions for the engine.
// This file contains DSL configuration conversion utilities.
package utils

import (
	"encoding/json"
	"fmt"

	"github.com/verustcode/verustcode/internal/dsl"
	"github.com/verustcode/verustcode/internal/model"
)

// DSLConfigToJSONMap converts DSL config to JSONMap for storage.
func DSLConfigToJSONMap(config *dsl.ReviewRulesConfig) (model.JSONMap, error) {
	data, err := json.Marshal(config)
	if err != nil {
		return nil, err
	}
	var result model.JSONMap
	if err := json.Unmarshal(data, &result); err != nil {
		return nil, err
	}
	return result, nil
}

// RuleConfigToJSONMap converts a single rule config to JSONMap for storage.
func RuleConfigToJSONMap(rule *dsl.ReviewRuleConfig) (model.JSONMap, error) {
	data, err := json.Marshal(rule)
	if err != nil {
		return nil, err
	}
	var result model.JSONMap
	if err := json.Unmarshal(data, &result); err != nil {
		return nil, err
	}
	return result, nil
}

// JSONMapToDSLConfig converts JSONMap back to DSL config (for recovery).
func JSONMapToDSLConfig(data model.JSONMap) (*dsl.ReviewRulesConfig, error) {
	if len(data) == 0 {
		return nil, fmt.Errorf("empty DSL config data")
	}

	// Marshal JSONMap to JSON bytes
	jsonBytes, err := json.Marshal(data)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal JSONMap: %w", err)
	}

	// Unmarshal to DSL config
	var config dsl.ReviewRulesConfig
	if err := json.Unmarshal(jsonBytes, &config); err != nil {
		return nil, fmt.Errorf("failed to unmarshal DSL config: %w", err)
	}

	return &config, nil
}
