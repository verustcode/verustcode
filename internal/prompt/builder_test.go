package prompt

import (
	"testing"

	"github.com/verustcode/verustcode/internal/dsl"
)

// TestNewBuilder tests creating a new builder
func TestNewBuilder(t *testing.T) {
	builder := NewBuilder()
	if builder == nil {
		t.Error("NewBuilder() returned nil")
	}
}

// TestBuilder_Build tests building a Spec from a ReviewRuleConfig
func TestBuilder_Build(t *testing.T) {
	builder := NewBuilder()

	t.Run("basic rule", func(t *testing.T) {
		rule := &dsl.ReviewRuleConfig{
			ID:          "test-rule",
			Description: "Test Reviewer",
			Goals: dsl.GoalsConfig{
				Areas: []string{"security", "performance"},
				Avoid: []string{"false positives"},
			},
		}

		spec := builder.Build(rule, nil)

		if spec == nil {
			t.Fatal("Build() returned nil")
		}

		if spec.ReviewerID != "test-rule" {
			t.Errorf("ReviewerID = %s, want test-rule", spec.ReviewerID)
		}

		if spec.SystemRole.Description != "Test Reviewer" {
			t.Errorf("SystemRole.Description = %s, want 'Test Reviewer'", spec.SystemRole.Description)
		}

		if len(spec.Goals.Areas) != 2 {
			t.Errorf("len(Goals.Areas) = %d, want 2", len(spec.Goals.Areas))
		}

		// Avoid is now moved to Constraints
		if len(spec.Constraints.Avoid) != 1 {
			t.Errorf("len(Constraints.Avoid) = %d, want 1", len(spec.Constraints.Avoid))
		}
	})

	t.Run("rule with build context", func(t *testing.T) {
		rule := &dsl.ReviewRuleConfig{
			ID:          "test-rule",
			Description: "Test Reviewer",
			Goals: dsl.GoalsConfig{
				Areas: []string{"security"},
			},
		}

		ctx := &BuildContext{
			RepoPath:      "/path/to/repo",
			RepoURL:       "https://github.com/test/repo",
			Ref:           "main",
			CommitSHA:     "abc123",
			BaseCommitSHA: "def456",
			PRNumber:      42,
			PRTitle:       "Test PR",
			PRDescription: "Test description",
			ChangedFiles:  []string{"file1.go", "file2.go"},
			Commits:       []string{"abc123", "def456"},
		}

		spec := builder.Build(rule, ctx)

		if spec.Context.RepoPath != "/path/to/repo" {
			t.Errorf("Context.RepoPath = %s, want '/path/to/repo'", spec.Context.RepoPath)
		}

		if spec.Context.PRNumber != 42 {
			t.Errorf("Context.PRNumber = %d, want 42", spec.Context.PRNumber)
		}

		if len(spec.Context.ChangedFiles) != 2 {
			t.Errorf("len(Context.ChangedFiles) = %d, want 2", len(spec.Context.ChangedFiles))
		}
	})

	t.Run("rule with output style", func(t *testing.T) {
		falseVal := false
		rule := &dsl.ReviewRuleConfig{
			ID:          "test-rule",
			Description: "Test Reviewer",
			Goals: dsl.GoalsConfig{
				Areas: []string{"security"},
			},
			// Note: Constraints must be non-nil for output style language to be applied
			Constraints: &dsl.ConstraintsConfig{},
			Output: &dsl.OutputConfig{
				Style: &dsl.OutputStyleConfig{
					Tone:     "strict",
					Concise:  &falseVal,
					Language: "Chinese",
				},
			},
		}

		spec := builder.Build(rule, nil)

		// Tone is now in Constraints (moved from SystemRole)
		if spec.Constraints.Tone != "strict" {
			t.Errorf("Constraints.Tone = %s, want 'strict'", spec.Constraints.Tone)
		}

		// Concise is in Constraints
		if spec.Constraints.Concise != false {
			t.Errorf("Constraints.Concise = %v, want false", spec.Constraints.Concise)
		}

		if spec.Constraints.Language != "Chinese" {
			t.Errorf("Constraints.Language = %s, want 'Chinese'", spec.Constraints.Language)
		}
	})

	t.Run("rule with constraints", func(t *testing.T) {
		rule := &dsl.ReviewRuleConfig{
			ID:          "test-rule",
			Description: "Test Reviewer",
			Goals: dsl.GoalsConfig{
				Areas: []string{"security"},
			},
			Constraints: &dsl.ConstraintsConfig{
				ScopeControl: []string{"Review only changed code"},
				Severity: &dsl.SeverityConfig{
					MinReport: "medium",
				},
			},
		}

		spec := builder.Build(rule, nil)

		if len(spec.Constraints.ScopeControl) != 1 {
			t.Errorf("len(Constraints.ScopeControl) = %d, want 1", len(spec.Constraints.ScopeControl))
		}

		// Severity levels are now system constants
		if len(spec.Constraints.SeverityLevels) != 5 {
			t.Errorf("len(Constraints.SeverityLevels) = %d, want 5", len(spec.Constraints.SeverityLevels))
		}

		if spec.Constraints.MinSeverity != "medium" {
			t.Errorf("Constraints.MinSeverity = %s, want 'medium'", spec.Constraints.MinSeverity)
		}
	})
}

// TestBuilder_BuildSystemRole tests building the system role specification
func TestBuilder_BuildSystemRole(t *testing.T) {
	builder := NewBuilder()

	t.Run("description only", func(t *testing.T) {
		rule := &dsl.ReviewRuleConfig{
			Description: "Test Reviewer",
		}

		spec := builder.Build(rule, nil)

		// SystemRole now only contains Description
		if spec.SystemRole.Description != "Test Reviewer" {
			t.Errorf("SystemRole.Description = %s, want 'Test Reviewer'", spec.SystemRole.Description)
		}

		// Tone is now in Constraints, with default "constructive"
		if spec.Constraints.Tone != "constructive" {
			t.Errorf("Constraints.Tone = %s, want 'constructive'", spec.Constraints.Tone)
		}

		// Concise is in Constraints, default true
		if spec.Constraints.Concise != true {
			t.Errorf("Constraints.Concise = %v, want true", spec.Constraints.Concise)
		}
	})

	t.Run("custom tone in constraints", func(t *testing.T) {
		falseVal := false
		rule := &dsl.ReviewRuleConfig{
			Description: "Strict Reviewer",
			Output: &dsl.OutputConfig{
				Style: &dsl.OutputStyleConfig{
					Tone:    "strict",
					Concise: &falseVal,
				},
			},
		}

		spec := builder.Build(rule, nil)

		// Tone is now in Constraints
		if spec.Constraints.Tone != "strict" {
			t.Errorf("Constraints.Tone = %s, want 'strict'", spec.Constraints.Tone)
		}

		if spec.Constraints.Concise != false {
			t.Errorf("Constraints.Concise = %v, want false", spec.Constraints.Concise)
		}
	})
}

// TestBuilder_BuildGoals tests building the goals specification
func TestBuilder_BuildGoals(t *testing.T) {
	builder := NewBuilder()

	t.Run("with areas - avoid moved to constraints", func(t *testing.T) {
		rule := &dsl.ReviewRuleConfig{
			ID:          "test",
			Description: "Reviewer",
			Goals: dsl.GoalsConfig{
				Areas: []string{"security", "performance", "readability"},
				Avoid: []string{"formatting", "stylistic issues"},
			},
		}

		spec := builder.Build(rule, nil)

		// Goals now only contains Areas
		if len(spec.Goals.Areas) != 3 {
			t.Errorf("len(Goals.Areas) = %d, want 3", len(spec.Goals.Areas))
		}

		// Avoid is now in Constraints
		if len(spec.Constraints.Avoid) != 2 {
			t.Errorf("len(Constraints.Avoid) = %d, want 2", len(spec.Constraints.Avoid))
		}
	})

	t.Run("empty goals", func(t *testing.T) {
		rule := &dsl.ReviewRuleConfig{
			ID:          "test",
			Description: "Reviewer",
			Goals:       dsl.GoalsConfig{},
		}

		spec := builder.Build(rule, nil)

		if len(spec.Goals.Areas) != 0 {
			t.Errorf("Goals.Areas should be empty, got %v", spec.Goals.Areas)
		}
	})
}

// TestBuilder_BuildConstraints tests building the constraints specification
func TestBuilder_BuildConstraints(t *testing.T) {
	builder := NewBuilder()

	t.Run("default constraints", func(t *testing.T) {
		rule := &dsl.ReviewRuleConfig{
			ID:          "test",
			Description: "Reviewer",
		}

		spec := builder.Build(rule, nil)

		// Check default values - severity levels are now system constants (5 levels)
		if len(spec.Constraints.SeverityLevels) != 5 {
			t.Errorf("len(Constraints.SeverityLevels) = %d, want 5", len(spec.Constraints.SeverityLevels))
		}

		// MinSeverity should be empty by default (no filtering)
		if spec.Constraints.MinSeverity != "" {
			t.Errorf("Constraints.MinSeverity = %s, want empty", spec.Constraints.MinSeverity)
		}

		if spec.Constraints.Concise != true {
			t.Errorf("Constraints.Concise = %v, want true", spec.Constraints.Concise)
		}

		if spec.Constraints.NoEmoji != true {
			t.Errorf("Constraints.NoEmoji = %v, want true", spec.Constraints.NoEmoji)
		}

		if spec.Constraints.NoDate != true {
			t.Errorf("Constraints.NoDate = %v, want true", spec.Constraints.NoDate)
		}
	})

	t.Run("custom constraints with min_report", func(t *testing.T) {
		rule := &dsl.ReviewRuleConfig{
			ID:          "test",
			Description: "Reviewer",
			Constraints: &dsl.ConstraintsConfig{
				ScopeControl: []string{"Only review changed lines"},
				Severity: &dsl.SeverityConfig{
					MinReport: "high",
				},
			},
		}

		spec := builder.Build(rule, nil)

		if len(spec.Constraints.ScopeControl) != 1 {
			t.Errorf("len(Constraints.ScopeControl) = %d, want 1", len(spec.Constraints.ScopeControl))
		}

		// Severity levels are system constants
		if len(spec.Constraints.SeverityLevels) != 5 {
			t.Errorf("len(Constraints.SeverityLevels) = %d, want 5", len(spec.Constraints.SeverityLevels))
		}

		if spec.Constraints.MinSeverity != "high" {
			t.Errorf("Constraints.MinSeverity = %s, want 'high'", spec.Constraints.MinSeverity)
		}
	})

	t.Run("language from context fallback", func(t *testing.T) {
		// Note: Constraints must be non-nil for language from context to be applied
		rule := &dsl.ReviewRuleConfig{
			ID:          "test",
			Description: "Reviewer",
			Constraints: &dsl.ConstraintsConfig{},
		}

		ctx := &BuildContext{
			OutputLanguage: "English",
		}

		spec := builder.Build(rule, ctx)

		if spec.Constraints.Language != "English" {
			t.Errorf("Constraints.Language = %s, want 'English'", spec.Constraints.Language)
		}
	})

	t.Run("language from output style takes precedence", func(t *testing.T) {
		// Note: Constraints must be non-nil for language to be applied
		rule := &dsl.ReviewRuleConfig{
			ID:          "test",
			Description: "Reviewer",
			Constraints: &dsl.ConstraintsConfig{},
			Output: &dsl.OutputConfig{
				Style: &dsl.OutputStyleConfig{
					Language: "Chinese",
				},
			},
		}

		ctx := &BuildContext{
			OutputLanguage: "English",
		}

		spec := builder.Build(rule, ctx)

		// Output style language should take precedence over context
		if spec.Constraints.Language != "Chinese" {
			t.Errorf("Constraints.Language = %s, want 'Chinese'", spec.Constraints.Language)
		}
	})

	t.Run("nil constraints still applies context language", func(t *testing.T) {
		rule := &dsl.ReviewRuleConfig{
			ID:          "test",
			Description: "Reviewer",
			// Constraints is nil
		}

		ctx := &BuildContext{
			OutputLanguage: "English",
		}

		spec := builder.Build(rule, ctx)

		// Language from context should be applied even if rule.Constraints is nil
		// This ensures consistent behavior - context provides runtime configuration
		if spec.Constraints.Language != "English" {
			t.Errorf("Constraints.Language = %s, want 'English' (from context)", spec.Constraints.Language)
		}
	})

	t.Run("FocusOnIssuesOnly defaults to true", func(t *testing.T) {
		rule := &dsl.ReviewRuleConfig{
			ID:          "test",
			Description: "Reviewer",
		}

		spec := builder.Build(rule, nil)

		// Default should be true
		if spec.Constraints.FocusOnIssuesOnly != true {
			t.Errorf("Constraints.FocusOnIssuesOnly = %v, want true", spec.Constraints.FocusOnIssuesOnly)
		}
	})

	t.Run("FocusOnIssuesOnly can be disabled", func(t *testing.T) {
		falseVal := false
		rule := &dsl.ReviewRuleConfig{
			ID:          "test",
			Description: "Reviewer",
			Constraints: &dsl.ConstraintsConfig{
				FocusOnIssuesOnly: &falseVal,
			},
		}

		spec := builder.Build(rule, nil)

		// Should be false when explicitly set
		if spec.Constraints.FocusOnIssuesOnly != false {
			t.Errorf("Constraints.FocusOnIssuesOnly = %v, want false", spec.Constraints.FocusOnIssuesOnly)
		}
	})

	t.Run("FocusOnIssuesOnly explicit true", func(t *testing.T) {
		trueVal := true
		rule := &dsl.ReviewRuleConfig{
			ID:          "test",
			Description: "Reviewer",
			Constraints: &dsl.ConstraintsConfig{
				FocusOnIssuesOnly: &trueVal,
			},
		}

		spec := builder.Build(rule, nil)

		// Should be true when explicitly set
		if spec.Constraints.FocusOnIssuesOnly != true {
			t.Errorf("Constraints.FocusOnIssuesOnly = %v, want true", spec.Constraints.FocusOnIssuesOnly)
		}
	})
}

// TestBuilder_BuildContext tests building the context specification
func TestBuilder_BuildContext(t *testing.T) {
	builder := NewBuilder()

	t.Run("nil context", func(t *testing.T) {
		rule := &dsl.ReviewRuleConfig{
			ID:          "test",
			Description: "Reviewer",
		}

		spec := builder.Build(rule, nil)

		// Context should have zero values
		if spec.Context.RepoPath != "" {
			t.Errorf("Context.RepoPath = %s, want empty", spec.Context.RepoPath)
		}

		if spec.Context.PRNumber != 0 {
			t.Errorf("Context.PRNumber = %d, want 0", spec.Context.PRNumber)
		}
	})

	t.Run("context with PreviousReviewForComparison", func(t *testing.T) {
		rule := &dsl.ReviewRuleConfig{
			ID:          "test",
			Description: "Reviewer",
		}

		ctx := &BuildContext{
			PRNumber:                    123,
			PreviousReviewForComparison: "## Previous Issues\n\n1. SQL injection found",
		}

		spec := builder.Build(rule, ctx)

		if spec.Context.PreviousReviewForComparison != "## Previous Issues\n\n1. SQL injection found" {
			t.Errorf("Context.PreviousReviewForComparison = %s, want previous review content", spec.Context.PreviousReviewForComparison)
		}
	})

	t.Run("full context", func(t *testing.T) {
		rule := &dsl.ReviewRuleConfig{
			ID:          "test",
			Description: "Reviewer",
		}

		ctx := &BuildContext{
			RepoPath:      "/repo/path",
			RepoURL:       "https://github.com/test/repo",
			Ref:           "feature-branch",
			CommitSHA:     "abc123def",
			BaseCommitSHA: "base456",
			PRNumber:      123,
			PRTitle:       "Feature: Add new feature",
			PRDescription: "This PR adds a new feature",
			ChangedFiles:  []string{"main.go", "util.go", "test.go"},
			Commits:       []string{"abc123", "def456", "ghi789"},
		}

		spec := builder.Build(rule, ctx)

		if spec.Context.RepoPath != "/repo/path" {
			t.Errorf("Context.RepoPath = %s, want '/repo/path'", spec.Context.RepoPath)
		}

		if spec.Context.RepoURL != "https://github.com/test/repo" {
			t.Errorf("Context.RepoURL = %s, want 'https://github.com/test/repo'", spec.Context.RepoURL)
		}

		if spec.Context.Ref != "feature-branch" {
			t.Errorf("Context.Ref = %s, want 'feature-branch'", spec.Context.Ref)
		}

		if spec.Context.CommitSHA != "abc123def" {
			t.Errorf("Context.CommitSHA = %s, want 'abc123def'", spec.Context.CommitSHA)
		}

		if spec.Context.BaseCommitSHA != "base456" {
			t.Errorf("Context.BaseCommitSHA = %s, want 'base456'", spec.Context.BaseCommitSHA)
		}

		if spec.Context.PRNumber != 123 {
			t.Errorf("Context.PRNumber = %d, want 123", spec.Context.PRNumber)
		}

		if spec.Context.PRTitle != "Feature: Add new feature" {
			t.Errorf("Context.PRTitle = %s, want 'Feature: Add new feature'", spec.Context.PRTitle)
		}

		if len(spec.Context.ChangedFiles) != 3 {
			t.Errorf("len(Context.ChangedFiles) = %d, want 3", len(spec.Context.ChangedFiles))
		}

		if len(spec.Context.Commits) != 3 {
			t.Errorf("len(Context.Commits) = %d, want 3", len(spec.Context.Commits))
		}

		if spec.Context.PRDescription != "This PR adds a new feature" {
			t.Errorf("Context.PRDescription = %s, want 'This PR adds a new feature'", spec.Context.PRDescription)
		}
	})
}

// TestBuilder_BuildAll tests building specs for all rules
func TestBuilder_BuildAll(t *testing.T) {
	builder := NewBuilder()

	t.Run("multiple rules", func(t *testing.T) {
		config := &dsl.ReviewRulesConfig{
			Version: "1.0",
			Rules: []dsl.ReviewRuleConfig{
				{
					ID:          "rule1",
					Description: "Security Reviewer",
					Goals: dsl.GoalsConfig{
						Areas: []string{"security"},
					},
				},
				{
					ID:          "rule2",
					Description: "Performance Reviewer",
					Goals: dsl.GoalsConfig{
						Areas: []string{"performance"},
					},
				},
				{
					ID:          "rule3",
					Description: "Quality Reviewer",
					Goals: dsl.GoalsConfig{
						Areas: []string{"readability"},
					},
				},
			},
		}

		specs := builder.BuildAll(config, nil)

		if len(specs) != 3 {
			t.Errorf("BuildAll() returned %d specs, want 3", len(specs))
		}

		// Verify each spec
		expectedIDs := []string{"rule1", "rule2", "rule3"}
		for i, spec := range specs {
			if spec.ReviewerID != expectedIDs[i] {
				t.Errorf("specs[%d].ReviewerID = %s, want %s", i, spec.ReviewerID, expectedIDs[i])
			}
		}
	})

	t.Run("empty rules", func(t *testing.T) {
		config := &dsl.ReviewRulesConfig{
			Version: "1.0",
			Rules:   []dsl.ReviewRuleConfig{},
		}

		specs := builder.BuildAll(config, nil)

		if len(specs) != 0 {
			t.Errorf("BuildAll() returned %d specs, want 0", len(specs))
		}
	})

	t.Run("with context", func(t *testing.T) {
		config := &dsl.ReviewRulesConfig{
			Version: "1.0",
			Rules: []dsl.ReviewRuleConfig{
				{
					ID:          "rule1",
					Description: "Reviewer",
					Goals: dsl.GoalsConfig{
						Areas: []string{"security"},
					},
				},
			},
		}

		ctx := &BuildContext{
			PRNumber: 42,
		}

		specs := builder.BuildAll(config, ctx)

		if len(specs) != 1 {
			t.Fatalf("BuildAll() returned %d specs, want 1", len(specs))
		}

		if specs[0].Context.PRNumber != 42 {
			t.Errorf("specs[0].Context.PRNumber = %d, want 42", specs[0].Context.PRNumber)
		}
	})
}

// TestNewReviewResult tests creating a new review result
func TestNewReviewResult(t *testing.T) {
	result := NewReviewResult("test-reviewer")

	if result == nil {
		t.Fatal("NewReviewResult() returned nil")
	}

	if result.ReviewerID != "test-reviewer" {
		t.Errorf("ReviewerID = %s, want 'test-reviewer'", result.ReviewerID)
	}

	if result.Data == nil {
		t.Error("Data should not be nil")
	}

	if len(result.Data) != 0 {
		t.Errorf("len(Data) = %d, want 0", len(result.Data))
	}
}
