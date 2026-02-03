package prompt

import (
	"strings"
	"testing"

	"github.com/verustcode/verustcode/internal/dsl"
)

func TestRenderer_RenderPRDescription(t *testing.T) {
	t.Run("renders PR description when present", func(t *testing.T) {
		renderer := NewRenderer()

		spec := &Spec{
			SystemRole: SystemRoleSpec{
				Description: "Test Reviewer",
			},
			Goals: GoalsSpec{
				Areas: []AreaItem{{ID: "security", Description: "Security vulnerabilities"}},
			},
			Constraints: ConstraintsSpec{
				SeverityLevels: []string{"critical", "high", "medium", "low", "info"},
			},
			Context: ContextSpec{
				RepoPath:      "/test/repo",
				Ref:           "feature-branch",
				CommitSHA:     "abc123",
				BaseCommitSHA: "def456",
				Commits:       []string{"abc123", "def456"},
				PRNumber:      123,
				PRTitle:       "Add authentication feature",
				PRDescription: "This PR implements JWT-based authentication.\n\nKey changes:\n- Add JWT middleware\n- Update user model\n- Add tests",
			},
		}

		result, err := renderer.Render(spec)
		if err != nil {
			t.Fatalf("Render failed: %v", err)
		}

		// Check that it includes the PR/MR review explanation
		if !strings.Contains(result, "This is a code review for a Pull Request / Merge Request") {
			t.Error("Expected PR/MR review explanation in output")
		}

		// Check that PR/MR Info section is present
		if !strings.Contains(result, "### PR/MR Info") {
			t.Error("Expected '### PR/MR Info' section in output")
		}

		// Check that Branch and Commit Range are in the PR/MR Info section (after PR title)
		if !strings.Contains(result, "Branch: feature-branch") {
			t.Error("Expected 'Branch: feature-branch' in output")
		}

		if !strings.Contains(result, "Commit Range: def456..abc123") {
			t.Error("Expected 'Commit Range: def456..abc123' in output")
		}

		// Check that PR description is included
		if !strings.Contains(result, "Description:") {
			t.Error("Expected 'Description:' label in output")
		}

		if !strings.Contains(result, "This PR implements JWT-based authentication") {
			t.Error("Expected PR description content in output")
		}

		if !strings.Contains(result, "Add JWT middleware") {
			t.Error("Expected PR description details in output")
		}

		// Also check PR title is there
		if !strings.Contains(result, "Add authentication feature") {
			t.Error("Expected PR title in output")
		}

		// RepoPath should NOT be in the output anymore
		if strings.Contains(result, "Repository:") {
			t.Error("Should not include 'Repository:' in output (RepoPath removed)")
		}
	})

	t.Run("omits description section when PR description is empty", func(t *testing.T) {
		renderer := NewRenderer()

		spec := &Spec{
			SystemRole: SystemRoleSpec{
				Description: "Test Reviewer",
			},
			Goals: GoalsSpec{
				Areas: []AreaItem{{ID: "security", Description: "Security vulnerabilities"}},
			},
			Constraints: ConstraintsSpec{
				SeverityLevels: []string{"critical", "high", "medium", "low", "info"},
			},
			Context: ContextSpec{
				RepoPath:      "/test/repo",
				PRNumber:      123,
				PRTitle:       "Add feature",
				PRDescription: "", // Empty description
			},
		}

		result, err := renderer.Render(spec)
		if err != nil {
			t.Fatalf("Render failed: %v", err)
		}

		// Should not have "Description:" label when description is empty
		if strings.Contains(result, "Description:") {
			t.Error("Should not include 'Description:' label when PR description is empty")
		}

		// But PR title should still be there
		if !strings.Contains(result, "Add feature") {
			t.Error("Expected PR title in output")
		}
	})

	t.Run("omits PR section when no PR number", func(t *testing.T) {
		renderer := NewRenderer()

		spec := &Spec{
			SystemRole: SystemRoleSpec{
				Description: "Test Reviewer",
			},
			Goals: GoalsSpec{
				Areas: []AreaItem{{ID: "security", Description: "Security vulnerabilities"}},
			},
			Constraints: ConstraintsSpec{
				SeverityLevels: []string{"critical", "high", "medium", "low", "info"},
			},
			Context: ContextSpec{
				RepoPath:      "/test/repo",
				PRNumber:      0, // No PR
				PRTitle:       "",
				PRDescription: "",
			},
		}

		result, err := renderer.Render(spec)
		if err != nil {
			t.Fatalf("Render failed: %v", err)
		}

		// Should not have PR/MR Info section at all
		if strings.Contains(result, "PR/MR Info") {
			t.Error("Should not include 'PR/MR Info' section when PR number is 0")
		}

		if strings.Contains(result, "Description:") {
			t.Error("Should not include 'Description:' when there is no PR")
		}

		// Should not include the PR/MR review explanation
		if strings.Contains(result, "This is a code review for a Pull Request") {
			t.Error("Should not include PR/MR review explanation when PR number is 0")
		}
	})
}

func TestBuilder_BuildWithPRDescription(t *testing.T) {
	t.Run("builds spec with PR description", func(t *testing.T) {
		builder := NewBuilder()

		rule := &dsl.ReviewRuleConfig{
			ID:          "test",
			Description: "Security Reviewer",
			Goals: dsl.GoalsConfig{
				Areas: []string{"security", "best-practices"},
			},
		}

		ctx := &BuildContext{
			RepoPath:      "/test/repo",
			PRNumber:      123,
			PRTitle:       "Add authentication feature",
			PRDescription: "This PR implements JWT-based authentication.\n\nKey changes:\n- Add JWT middleware\n- Update user model\n- Add tests",
		}

		spec := builder.Build(rule, ctx)
		renderer := NewRenderer()
		promptText, err := renderer.Render(spec)
		if err != nil {
			t.Fatalf("Failed to render prompt: %v", err)
		}

		// Verify the full flow: Build -> Render produces correct output
		if !strings.Contains(promptText, "PR #123") {
			t.Error("Expected PR number in rendered prompt")
		}

		if !strings.Contains(promptText, "Add authentication feature") {
			t.Error("Expected PR title in rendered prompt")
		}

		if !strings.Contains(promptText, "Description:") {
			t.Error("Expected 'Description:' label in rendered prompt")
		}

		if !strings.Contains(promptText, "This PR implements JWT-based authentication") {
			t.Error("Expected PR description content in rendered prompt")
		}
	})
}

func TestQuote(t *testing.T) {
	t.Run("formats single line as blockquote", func(t *testing.T) {
		result := quote("Hello world")
		expected := "> Hello world"
		if result != expected {
			t.Errorf("Expected %q, got %q", expected, result)
		}
	})

	t.Run("formats multi-line as blockquote", func(t *testing.T) {
		result := quote("Line 1\nLine 2\nLine 3")
		expected := "> Line 1\n> Line 2\n> Line 3"
		if result != expected {
			t.Errorf("Expected %q, got %q", expected, result)
		}
	})

	t.Run("handles empty string", func(t *testing.T) {
		result := quote("")
		if result != "" {
			t.Errorf("Expected empty string, got %q", result)
		}
	})

	t.Run("handles markdown content with code blocks", func(t *testing.T) {
		input := "## Title\n\n```go\nfunc main() {}\n```\n\n- Item 1"
		result := quote(input)
		// Each line should start with "> "
		lines := strings.Split(result, "\n")
		for i, line := range lines {
			if !strings.HasPrefix(line, "> ") {
				t.Errorf("Line %d should start with '> ', got %q", i, line)
			}
		}
	})
}

func TestRenderer_RenderPRDescriptionAsBlockquote(t *testing.T) {
	t.Run("renders PR description as blockquote", func(t *testing.T) {
		renderer := NewRenderer()

		spec := &Spec{
			SystemRole: SystemRoleSpec{
				Description: "Test Reviewer",
			},
			Goals: GoalsSpec{
				Areas: []AreaItem{{ID: "security", Description: "Security vulnerabilities"}},
			},
			Constraints: ConstraintsSpec{
				SeverityLevels: []string{"critical", "high", "medium", "low", "info"},
			},
			Context: ContextSpec{
				RepoPath:      "/test/repo",
				PRNumber:      123,
				PRTitle:       "Add feature",
				PRDescription: "Line 1\nLine 2",
			},
		}

		result, err := renderer.Render(spec)
		if err != nil {
			t.Fatalf("Render failed: %v", err)
		}

		// Check that PR description is formatted as blockquote
		if !strings.Contains(result, "> Line 1") {
			t.Error("Expected first line to be blockquoted")
		}

		if !strings.Contains(result, "> Line 2") {
			t.Error("Expected second line to be blockquoted")
		}
	})
}

func TestRenderer_RenderPreviousReviewForComparison(t *testing.T) {
	t.Run("renders previous review when present", func(t *testing.T) {
		renderer := NewRenderer()

		spec := &Spec{
			SystemRole: SystemRoleSpec{
				Description: "Test Reviewer",
			},
			Goals: GoalsSpec{
				Areas: []AreaItem{{ID: "security", Description: "Security vulnerabilities"}},
			},
			Constraints: ConstraintsSpec{
				SeverityLevels: []string{"critical", "high", "medium", "low", "info"},
			},
			Context: ContextSpec{
				PRNumber:                    123,
				PRTitle:                     "Add feature",
				PreviousReviewForComparison: "## Previous Issues\n\n1. SQL injection vulnerability in user input\n2. Missing null check",
			},
		}

		result, err := renderer.Render(spec)
		if err != nil {
			t.Fatalf("Render failed: %v", err)
		}

		// Check that the historical comparison section is present
		if !strings.Contains(result, "### Previous Review Result (Historical Comparison)") {
			t.Error("Expected '### Previous Review Result (Historical Comparison)' section in output")
		}

		// Check that status indicators are documented
		if !strings.Contains(result, "[FIXED]") {
			t.Error("Expected '[FIXED]' status indicator in output")
		}

		if !strings.Contains(result, "[NEW]") {
			t.Error("Expected '[NEW]' status indicator in output")
		}

		if !strings.Contains(result, "[PERSISTS]") {
			t.Error("Expected '[PERSISTS]' status indicator in output")
		}

		// Check that previous review content is blockquoted
		if !strings.Contains(result, "> ## Previous Issues") {
			t.Error("Expected previous review content to be blockquoted")
		}

		if !strings.Contains(result, "> 1. SQL injection vulnerability") {
			t.Error("Expected previous review issue to be blockquoted")
		}
	})

	t.Run("omits previous review section when empty", func(t *testing.T) {
		renderer := NewRenderer()

		spec := &Spec{
			SystemRole: SystemRoleSpec{
				Description: "Test Reviewer",
			},
			Goals: GoalsSpec{
				Areas: []AreaItem{{ID: "security", Description: "Security vulnerabilities"}},
			},
			Constraints: ConstraintsSpec{
				SeverityLevels: []string{"critical", "high", "medium", "low", "info"},
			},
			Context: ContextSpec{
				PRNumber:                    123,
				PRTitle:                     "Add feature",
				PreviousReviewForComparison: "", // Empty - no previous review
			},
		}

		result, err := renderer.Render(spec)
		if err != nil {
			t.Fatalf("Render failed: %v", err)
		}

		// Should not have previous review section
		if strings.Contains(result, "Previous Review Result") {
			t.Error("Should not include 'Previous Review Result' section when previous review is empty")
		}

		if strings.Contains(result, "[FIXED]") {
			t.Error("Should not include status indicators when previous review is empty")
		}
	})
}
