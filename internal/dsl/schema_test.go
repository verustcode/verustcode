package dsl

import (
	"testing"
)

func TestDefaultReviewRuleConfig(t *testing.T) {
	config := DefaultReviewRuleConfig()

	// Validate default agent
	if config.Agent.Type != "cursor" {
		t.Errorf("Agent.Type = %v, want cursor", config.Agent.Type)
	}
	if config.Agent.Model != "" {
		t.Errorf("Agent.Model = %v, want empty (default)", config.Agent.Model)
	}

	// Validate constraints
	if config.Constraints == nil {
		t.Error("Constraints should not be nil")
		return
	}

	// Validate severity in constraints - should be nil by default (no filtering)
	if config.Constraints.Severity != nil {
		t.Error("Constraints.Severity should be nil by default (no min_report filtering)")
	}

	// Validate output style in output
	if config.Output == nil {
		t.Error("Output should not be nil")
		return
	}
	if config.Output.Style == nil {
		t.Error("Output.Style should not be nil")
		return
	}
	if config.Output.Style.Tone != "constructive" {
		t.Errorf("Output.Style.Tone = %v, want constructive", config.Output.Style.Tone)
	}

	// Validate duplicates in constraints
	if config.Constraints.Duplicates == nil {
		t.Error("Constraints.Duplicates should not be nil")
		return
	}
	if config.Constraints.Duplicates.SuppressSimilar == nil || !*config.Constraints.Duplicates.SuppressSimilar {
		t.Error("Constraints.Duplicates.SuppressSimilar should be true")
	}
	if config.Constraints.Duplicates.Similarity != 0.88 {
		t.Errorf("Constraints.Duplicates.Similarity = %v, want 0.88", config.Constraints.Duplicates.Similarity)
	}

	// Validate default output - no default channels (user must configure)
	if config.Output == nil {
		t.Error("Output should not be nil")
		return
	}
	if len(config.Output.Channels) != 0 {
		t.Errorf("len(Output.Channels) = %d, want 0 (no default channels)", len(config.Output.Channels))
	}
	// Note: User must explicitly configure at least one output channel
}

func TestReviewRulesConfig_GetRuleByID(t *testing.T) {
	config := &ReviewRulesConfig{
		Rules: []ReviewRuleConfig{
			{ID: "rule1", Description: "Security Reviewer"},
			{ID: "rule2", Description: "Code Quality Reviewer"},
			{ID: "rule3", Description: "Performance Reviewer"},
		},
	}

	tests := []struct {
		name   string
		id     string
		found  bool
		wantID string
	}{
		{
			name:   "find existing rule",
			id:     "rule2",
			found:  true,
			wantID: "rule2",
		},
		{
			name:  "rule not found",
			id:    "nonexistent",
			found: false,
		},
		{
			name:   "find first rule",
			id:     "rule1",
			found:  true,
			wantID: "rule1",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rule := config.GetRuleByID(tt.id)
			if tt.found {
				if rule == nil {
					t.Errorf("GetRuleByID(%s) returned nil, want rule", tt.id)
					return
				}
				if rule.ID != tt.wantID {
					t.Errorf("GetRuleByID(%s).ID = %v, want %v", tt.id, rule.ID, tt.wantID)
				}
			} else {
				if rule != nil {
					t.Errorf("GetRuleByID(%s) = %v, want nil", tt.id, rule)
				}
			}
		})
	}
}

func TestReviewRulesConfig_GetRuleIDs(t *testing.T) {
	config := &ReviewRulesConfig{
		Rules: []ReviewRuleConfig{
			{ID: "security"},
			{ID: "quality"},
			{ID: "performance"},
		},
	}

	ids := config.GetRuleIDs()

	if len(ids) != 3 {
		t.Errorf("GetRuleIDs() returned %d IDs, want 3", len(ids))
		return
	}

	expected := []string{"security", "quality", "performance"}
	for i, id := range ids {
		if id != expected[i] {
			t.Errorf("GetRuleIDs()[%d] = %v, want %v", i, id, expected[i])
		}
	}
}

func TestReviewRulesConfig_EmptyRules(t *testing.T) {
	config := &ReviewRulesConfig{
		Rules: []ReviewRuleConfig{},
	}

	ids := config.GetRuleIDs()
	if len(ids) != 0 {
		t.Errorf("GetRuleIDs() returned %d IDs for empty rules, want 0", len(ids))
	}

	rule := config.GetRuleByID("any")
	if rule != nil {
		t.Errorf("GetRuleByID() returned %v for empty rules, want nil", rule)
	}
}
