// Package model defines the data models for the application.
// This file contains unit tests for model types.
package model

import (
	"encoding/json"
	"testing"
)

// TestStringArrayValue tests StringArray.Value() method
func TestStringArrayValue(t *testing.T) {
	tests := []struct {
		name    string
		input   StringArray
		want    string
		wantErr bool
	}{
		{
			name:  "empty array",
			input: StringArray{},
			want:  "[]",
		},
		{
			name:  "nil array",
			input: nil,
			want:  "[]",
		},
		{
			name:  "single element",
			input: StringArray{"hello"},
			want:  `["hello"]`,
		},
		{
			name:  "multiple elements",
			input: StringArray{"a", "b", "c"},
			want:  `["a","b","c"]`,
		},
		{
			name:  "elements with special characters",
			input: StringArray{"hello world", "foo\"bar", "test\nline"},
			want:  `["hello world","foo\"bar","test\nline"]`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := tt.input.Value()
			if (err != nil) != tt.wantErr {
				t.Errorf("StringArray.Value() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("StringArray.Value() = %v, want %v", got, tt.want)
			}
		})
	}
}

// TestStringArrayScan tests StringArray.Scan() method
func TestStringArrayScan(t *testing.T) {
	tests := []struct {
		name    string
		input   interface{}
		want    StringArray
		wantErr bool
	}{
		{
			name:  "nil value",
			input: nil,
			want:  StringArray{},
		},
		{
			name:  "empty array as string",
			input: "[]",
			want:  StringArray{},
		},
		{
			name:  "empty array as bytes",
			input: []byte("[]"),
			want:  StringArray{},
		},
		{
			name:  "single element as string",
			input: `["hello"]`,
			want:  StringArray{"hello"},
		},
		{
			name:  "multiple elements as string",
			input: `["a","b","c"]`,
			want:  StringArray{"a", "b", "c"},
		},
		{
			name:  "multiple elements as bytes",
			input: []byte(`["a","b","c"]`),
			want:  StringArray{"a", "b", "c"},
		},
		{
			name:    "invalid JSON",
			input:   "not json",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var s StringArray
			err := s.Scan(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("StringArray.Scan() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && len(s) != len(tt.want) {
				t.Errorf("StringArray.Scan() length = %d, want %d", len(s), len(tt.want))
				return
			}
			for i := range tt.want {
				if s[i] != tt.want[i] {
					t.Errorf("StringArray.Scan()[%d] = %v, want %v", i, s[i], tt.want[i])
				}
			}
		})
	}
}

// TestJSONMapValue tests JSONMap.Value() method
func TestJSONMapValue(t *testing.T) {
	tests := []struct {
		name    string
		input   JSONMap
		wantErr bool
	}{
		{
			name:  "nil map",
			input: nil,
		},
		{
			name:  "empty map",
			input: JSONMap{},
		},
		{
			name: "simple map",
			input: JSONMap{
				"key": "value",
			},
		},
		{
			name: "nested map",
			input: JSONMap{
				"key1": "value1",
				"key2": 42,
				"key3": true,
				"nested": map[string]interface{}{
					"inner": "value",
				},
			},
		},
		{
			name: "map with array",
			input: JSONMap{
				"items": []interface{}{"a", "b", "c"},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := tt.input.Value()
			if (err != nil) != tt.wantErr {
				t.Errorf("JSONMap.Value() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			// Value should be a valid JSON string
			if got != nil {
				if str, ok := got.(string); ok {
					var m map[string]interface{}
					if err := json.Unmarshal([]byte(str), &m); err != nil {
						t.Errorf("JSONMap.Value() returned invalid JSON: %v", err)
					}
				}
			}
		})
	}
}

// TestJSONMapScan tests JSONMap.Scan() method
func TestJSONMapScan(t *testing.T) {
	tests := []struct {
		name     string
		input    interface{}
		wantKeys []string
		wantErr  bool
	}{
		{
			name:     "nil value",
			input:    nil,
			wantKeys: []string{},
		},
		{
			name:     "empty object as string",
			input:    "{}",
			wantKeys: []string{},
		},
		{
			name:     "empty object as bytes",
			input:    []byte("{}"),
			wantKeys: []string{},
		},
		{
			name:     "simple object as string",
			input:    `{"key":"value"}`,
			wantKeys: []string{"key"},
		},
		{
			name:     "simple object as bytes",
			input:    []byte(`{"key":"value"}`),
			wantKeys: []string{"key"},
		},
		{
			name:     "nested object",
			input:    `{"key1":"value1","nested":{"inner":"value"}}`,
			wantKeys: []string{"key1", "nested"},
		},
		{
			name:    "invalid JSON",
			input:   "not json",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var m JSONMap
			err := m.Scan(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("JSONMap.Scan() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr {
				for _, key := range tt.wantKeys {
					if _, ok := m[key]; !ok {
						t.Errorf("JSONMap.Scan() missing key: %s", key)
					}
				}
			}
		})
	}
}

// TestReviewStatus tests ReviewStatus constants
func TestReviewStatus(t *testing.T) {
	statuses := []ReviewStatus{
		ReviewStatusPending,
		ReviewStatusRunning,
		ReviewStatusCompleted,
		ReviewStatusFailed,
		ReviewStatusCancelled,
	}

	expectedValues := []string{
		"pending",
		"running",
		"completed",
		"failed",
		"cancelled",
	}

	for i, status := range statuses {
		if string(status) != expectedValues[i] {
			t.Errorf("ReviewStatus = %s, want %s", status, expectedValues[i])
		}
	}
}

// TestRuleStatus tests RuleStatus constants
func TestRuleStatus(t *testing.T) {
	statuses := []RuleStatus{
		RuleStatusPending,
		RuleStatusRunning,
		RuleStatusCompleted,
		RuleStatusFailed,
		RuleStatusSkipped,
	}

	expectedValues := []string{
		"pending",
		"running",
		"completed",
		"failed",
		"skipped",
	}

	for i, status := range statuses {
		if string(status) != expectedValues[i] {
			t.Errorf("RuleStatus = %s, want %s", status, expectedValues[i])
		}
	}
}

// TestRunStatus tests RunStatus constants
func TestRunStatus(t *testing.T) {
	statuses := []RunStatus{
		RunStatusPending,
		RunStatusRunning,
		RunStatusCompleted,
		RunStatusFailed,
	}

	expectedValues := []string{
		"pending",
		"running",
		"completed",
		"failed",
	}

	for i, status := range statuses {
		if string(status) != expectedValues[i] {
			t.Errorf("RunStatus = %s, want %s", status, expectedValues[i])
		}
	}
}

// TestWebhookStatus tests WebhookStatus constants
func TestWebhookStatus(t *testing.T) {
	statuses := []WebhookStatus{
		WebhookStatusPending,
		WebhookStatusSuccess,
		WebhookStatusFailed,
	}

	expectedValues := []string{
		"pending",
		"success",
		"failed",
	}

	for i, status := range statuses {
		if string(status) != expectedValues[i] {
			t.Errorf("WebhookStatus = %s, want %s", status, expectedValues[i])
		}
	}
}

// TestAllModels tests the AllModels function
func TestAllModels(t *testing.T) {
	models := AllModels()
	if len(models) == 0 {
		t.Error("AllModels() returned empty slice")
	}

	// Check that all expected base models are included
	hasReview := false
	hasReviewRule := false
	hasReviewRuleRun := false
	hasReviewResult := false
	hasWebhookLog := false
	hasRepoConfig := false

	for _, model := range models {
		switch model.(type) {
		case *Review:
			hasReview = true
		case *ReviewRule:
			hasReviewRule = true
		case *ReviewRuleRun:
			hasReviewRuleRun = true
		case *ReviewResult:
			hasReviewResult = true
		case *ReviewResultWebhookLog:
			hasWebhookLog = true
		case *RepositoryReviewConfig:
			hasRepoConfig = true
		}
	}

	if !hasReview {
		t.Error("AllModels() missing Review")
	}
	if !hasReviewRule {
		t.Error("AllModels() missing ReviewRule")
	}
	if !hasReviewRuleRun {
		t.Error("AllModels() missing ReviewRuleRun")
	}
	if !hasReviewResult {
		t.Error("AllModels() missing ReviewResult")
	}
	if !hasWebhookLog {
		t.Error("AllModels() missing ReviewResultWebhookLog")
	}
	if !hasRepoConfig {
		t.Error("AllModels() missing RepositoryReviewConfig")
	}
}

// TestWeeklyStatsStructs tests the weekly stats structs
func TestWeeklyStatsStructs(t *testing.T) {
	t.Run("WeeklyDeliveryTime", func(t *testing.T) {
		stat := WeeklyDeliveryTime{
			Week: "2024-W01",
			P50:  1.5,
			P90:  3.5,
			P95:  5.0,
		}
		if stat.Week != "2024-W01" {
			t.Error("WeeklyDeliveryTime.Week not set correctly")
		}
	})

	t.Run("WeeklyCodeChange", func(t *testing.T) {
		stat := WeeklyCodeChange{
			Week:         "2024-W01",
			LinesAdded:   100,
			LinesDeleted: 50,
			NetChange:    50,
		}
		if stat.NetChange != 50 {
			t.Error("WeeklyCodeChange.NetChange not set correctly")
		}
	})

	t.Run("WeeklyFileChange", func(t *testing.T) {
		stat := WeeklyFileChange{
			Week:         "2024-W01",
			FilesChanged: 10,
		}
		if stat.FilesChanged != 10 {
			t.Error("WeeklyFileChange.FilesChanged not set correctly")
		}
	})

	t.Run("WeeklyMRCount", func(t *testing.T) {
		stat := WeeklyMRCount{
			Week:  "2024-W01",
			Count: 5,
		}
		if stat.Count != 5 {
			t.Error("WeeklyMRCount.Count not set correctly")
		}
	})

	t.Run("WeeklyRevision", func(t *testing.T) {
		stat := WeeklyRevision{
			Week:           "2024-W01",
			AvgRevisions:   2.5,
			TotalRevisions: 25,
			MRCount:        10,
			Layer1Count:    5,
			Layer2Count:    3,
			Layer3Count:    2,
			Layer1Label:    "1-2次",
			Layer2Label:    "3-4次",
			Layer3Label:    "5+次",
		}
		if stat.AvgRevisions != 2.5 {
			t.Error("WeeklyRevision.AvgRevisions not set correctly")
		}
	})

	t.Run("IssueSeverityStats", func(t *testing.T) {
		stat := IssueSeverityStats{
			Severity: "high",
			Count:    10,
		}
		if stat.Severity != "high" || stat.Count != 10 {
			t.Error("IssueSeverityStats not set correctly")
		}
	})

	t.Run("IssueCategoryStats", func(t *testing.T) {
		stat := IssueCategoryStats{
			Category: "security",
			Count:    5,
		}
		if stat.Category != "security" || stat.Count != 5 {
			t.Error("IssueCategoryStats not set correctly")
		}
	})

	t.Run("RepoStats", func(t *testing.T) {
		stat := RepoStats{
			DeliveryTimeStats: []WeeklyDeliveryTime{},
			CodeChangeStats:   []WeeklyCodeChange{},
			FileChangeStats:   []WeeklyFileChange{},
			MRCountStats:      []WeeklyMRCount{},
			RevisionStats:     []WeeklyRevision{},
		}
		if stat.DeliveryTimeStats == nil {
			t.Error("RepoStats.DeliveryTimeStats should not be nil")
		}
	})
}

// TestStringArrayRoundTrip tests saving and loading StringArray
func TestStringArrayRoundTrip(t *testing.T) {
	original := StringArray{"hello", "world", "test"}

	// Convert to driver.Value
	value, err := original.Value()
	if err != nil {
		t.Fatalf("Value() error: %v", err)
	}

	// Scan back
	var restored StringArray
	if err := restored.Scan(value); err != nil {
		t.Fatalf("Scan() error: %v", err)
	}

	// Compare
	if len(restored) != len(original) {
		t.Fatalf("Restored length = %d, want %d", len(restored), len(original))
	}
	for i := range original {
		if restored[i] != original[i] {
			t.Errorf("Restored[%d] = %s, want %s", i, restored[i], original[i])
		}
	}
}

// TestJSONMapRoundTrip tests saving and loading JSONMap
func TestJSONMapRoundTrip(t *testing.T) {
	original := JSONMap{
		"string": "value",
		"number": float64(42),
		"bool":   true,
		"nested": map[string]interface{}{
			"inner": "value",
		},
	}

	// Convert to driver.Value
	value, err := original.Value()
	if err != nil {
		t.Fatalf("Value() error: %v", err)
	}

	// Scan back
	var restored JSONMap
	if err := restored.Scan(value); err != nil {
		t.Fatalf("Scan() error: %v", err)
	}

	// Compare string value
	if restored["string"] != original["string"] {
		t.Errorf("Restored[string] = %v, want %v", restored["string"], original["string"])
	}

	// Compare number value
	if restored["number"] != original["number"] {
		t.Errorf("Restored[number] = %v, want %v", restored["number"], original["number"])
	}

	// Compare bool value
	if restored["bool"] != original["bool"] {
		t.Errorf("Restored[bool] = %v, want %v", restored["bool"], original["bool"])
	}
}
