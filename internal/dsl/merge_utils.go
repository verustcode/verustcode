package dsl

// merge_utils.go provides utility functions for merging configuration structs
// and other common utility functions to reduce code duplication

// mergeString merges two string values, with override taking precedence
// Returns override if non-empty, otherwise returns base
func mergeString(override, base string) string {
	if override != "" {
		return override
	}
	return base
}

// mergeBoolPtr merges two *bool values, with override taking precedence
// Returns override if non-nil, otherwise returns base
func mergeBoolPtr(override, base *bool) *bool {
	if override != nil {
		return override
	}
	return base
}

// mergeStringSlice merges two string slices
// Returns override if non-empty, otherwise returns base
func mergeStringSlice(override, base []string) []string {
	if len(override) > 0 {
		return override
	}
	return base
}

// mergeStringMap merges two string maps
// Returns override if non-nil, otherwise returns base
func mergeStringMap(override, base map[string]string) map[string]string {
	if override != nil {
		return override
	}
	return base
}

// mergeFloat64 merges two float64 values
// Returns override if non-zero, otherwise returns base
func mergeFloat64(override, base float64) float64 {
	if override != 0 {
		return override
	}
	return base
}

// containsString checks if a slice contains a string
// This is a common utility used throughout the DSL package
func containsString(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}
