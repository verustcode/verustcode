package dsl

import (
	"testing"
)

func TestParser_Parse_Valid(t *testing.T) {
	yamlContent := `
version: "1.0"
rule_base:
  agent:
    type: cursor
rules:
  - id: security
    description: Reviews code for security issues
    goals:
      areas:
        - injection
        - authentication
      avoid:
        - False positives on test code
  - id: quality
    description: Reviews code for quality and readability
    goals:
      areas:
        - readability
      avoid:
        - Formatting-only nitpicks
`

	parser := NewParser()
	config, err := parser.Parse([]byte(yamlContent))

	if err != nil {
		t.Fatalf("Parse() unexpected error: %v", err)
	}

	if config.Version != "1.0" {
		t.Errorf("Version = %v, want 1.0", config.Version)
	}

	if len(config.Rules) != 2 {
		t.Errorf("len(Rules) = %d, want 2", len(config.Rules))
		return
	}

	// Validate first rule
	rule1 := config.Rules[0]
	if rule1.ID != "security" {
		t.Errorf("Rules[0].ID = %v, want security", rule1.ID)
	}
	if rule1.Description != "Reviews code for security issues" {
		t.Errorf("Rules[0].Description = %v, want 'Reviews code for security issues'", rule1.Description)
	}
	// Validate rule base is applied
	if rule1.Agent.Type != "cursor" {
		t.Errorf("Rules[0].Agent.Type = %v, want cursor (from rule_base)", rule1.Agent.Type)
	}

	// Validate second rule
	rule2 := config.Rules[1]
	if rule2.ID != "quality" {
		t.Errorf("Rules[1].ID = %v, want quality", rule2.ID)
	}
}

func TestParser_Parse_EmptyRules(t *testing.T) {
	yamlContent := `
version: "1.0"
rules: []
`

	parser := NewParser()
	_, err := parser.Parse([]byte(yamlContent))

	if err == nil {
		t.Error("Parse() expected error for empty rules, got nil")
	}
}

func TestParser_Parse_MissingID(t *testing.T) {
	yamlContent := `
version: "1.0"
rules:
  - description: Security Reviewer
    goals:
      areas:
        - security
`

	parser := NewStrictParser()
	_, err := parser.Parse([]byte(yamlContent))

	if err == nil {
		t.Error("Parse() expected error for missing ID, got nil")
	}
}

func TestParser_Parse_DuplicateIDs(t *testing.T) {
	yamlContent := `
version: "1.0"
rules:
  - id: security
    description: Security Reviewer
    goals:
      areas:
        - security
  - id: security
    description: Another Security Reviewer
    goals:
      areas:
        - auth
`

	parser := NewParser()
	_, err := parser.Parse([]byte(yamlContent))

	if err == nil {
		t.Error("Parse() expected error for duplicate IDs, got nil")
	}
}

func TestParser_Parse_InvalidTone(t *testing.T) {
	yamlContent := `
version: "1.0"
rules:
  - id: security
    description: Security Reviewer
    goals:
      areas:
        - security
    output:
      style:
        tone: invalid_tone
`

	parser := NewParser()
	_, err := parser.Parse([]byte(yamlContent))

	if err == nil {
		t.Error("Parse() expected error for invalid tone, got nil")
	}
}

func TestParser_Parse_InvalidSimilarity(t *testing.T) {
	yamlContent := `
version: "1.0"
rules:
  - id: security
    description: Security Reviewer
    goals:
      areas:
        - security
    constraints:
      duplicates:
        suppress_similar: true
        similarity: 1.5
`

	parser := NewParser()
	_, err := parser.Parse([]byte(yamlContent))

	if err == nil {
		t.Error("Parse() expected error for invalid similarity, got nil")
	}
}

func TestParser_Parse_InvalidOutputType(t *testing.T) {
	yamlContent := `
version: "1.0"
rules:
  - id: security
    description: Security Reviewer
    goals:
      areas:
        - security
    output:
      channels:
        - type: invalid_type
`

	parser := NewParser()
	_, err := parser.Parse([]byte(yamlContent))

	if err == nil {
		t.Error("Parse() expected error for invalid output type, got nil")
	}
}

func TestParser_Parse_MissingOutputType(t *testing.T) {
	yamlContent := `
version: "1.0"
rules:
  - id: security
    description: Security Reviewer
    goals:
      areas:
        - security
    output:
      channels:
        - dir: reports
`

	parser := NewParser()
	_, err := parser.Parse([]byte(yamlContent))

	if err == nil {
		t.Error("Parse() expected error for missing output type, got nil")
	}
}

func TestParser_Parse_ValidOutputTypes(t *testing.T) {
	// Test new list-based output configuration
	yamlContent := `
version: "1.0"
rules:
  - id: security
    description: Security Reviewer
    goals:
      areas:
        - security
    output:
      format: markdown
      channels:
        - type: file
          dir: reports
        - type: comment
          overwrite: false
`

	parser := NewParser()
	config, err := parser.Parse([]byte(yamlContent))

	if err != nil {
		t.Fatalf("Parse() unexpected error: %v", err)
	}

	if config.Rules[0].Output == nil {
		t.Fatal("Output should not be nil")
	}

	if len(config.Rules[0].Output.Channels) != 2 {
		t.Errorf("len(Output.Channels) = %d, want 2", len(config.Rules[0].Output.Channels))
	}

	// Verify output types
	outputItems := config.Rules[0].Output.Channels
	if outputItems[0].Type != "file" {
		t.Errorf("Output.Channels[0].Type = %v, want file", outputItems[0].Type)
	}
	if outputItems[1].Type != "comment" {
		t.Errorf("Output.Channels[1].Type = %v, want comment", outputItems[1].Type)
	}
	// Note: Format is now per-channel, not at OutputConfig level
}

func TestParser_Parse_ValidFileConfig(t *testing.T) {
	yamlContent := `
version: "1.0"
rules:
  - id: security
    description: Security Reviewer
    goals:
      areas:
        - security
    output:
      channels:
        - type: file
          format: json
          dir: ./reports
          overwrite: true
`

	parser := NewParser()
	config, err := parser.Parse([]byte(yamlContent))

	if err != nil {
		t.Fatalf("Parse() unexpected error: %v", err)
	}

	if config.Rules[0].Output == nil {
		t.Fatal("Output should not be nil")
	}

	if len(config.Rules[0].Output.Channels) != 1 {
		t.Fatalf("len(Output.Channels) = %d, want 1", len(config.Rules[0].Output.Channels))
	}

	if config.Rules[0].Output.Channels[0].Type != "file" {
		t.Errorf("Output.Channels[0].Type = %v, want file", config.Rules[0].Output.Channels[0].Type)
	}
	if config.Rules[0].Output.Channels[0].Format != "json" {
		t.Errorf("Output.Channels[0].Format = %v, want json", config.Rules[0].Output.Channels[0].Format)
	}
	if config.Rules[0].Output.Channels[0].Dir != "./reports" {
		t.Errorf("Output.Channels[0].Dir = %v, want ./reports", config.Rules[0].Output.Channels[0].Dir)
	}
}

func TestParser_Parse_InvalidChannelFormat(t *testing.T) {
	yamlContent := `
version: "1.0"
rules:
  - id: security
    description: Security Reviewer
    goals:
      areas:
        - security
    output:
      channels:
        - type: file
          format: invalid_format
`

	parser := NewParser()
	_, err := parser.Parse([]byte(yamlContent))

	if err == nil {
		t.Error("Parse() expected error for invalid channel format, got nil")
	}
}

func TestParser_Parse_ValidCommentConfig(t *testing.T) {
	yamlContent := `
version: "1.0"
rules:
  - id: security
    description: Security Reviewer
    goals:
      areas:
        - security
    output:
      channels:
        - type: comment
          overwrite: true
          marker_prefix: custom_marker
`

	parser := NewParser()
	config, err := parser.Parse([]byte(yamlContent))

	if err != nil {
		t.Fatalf("Parse() unexpected error: %v", err)
	}

	if config.Rules[0].Output == nil {
		t.Fatal("Output should not be nil")
	}

	if len(config.Rules[0].Output.Channels) != 1 {
		t.Fatalf("len(Output.Channels) = %d, want 1", len(config.Rules[0].Output.Channels))
	}

	if config.Rules[0].Output.Channels[0].Type != "comment" {
		t.Errorf("Output.Channels[0].Type = %v, want comment", config.Rules[0].Output.Channels[0].Type)
	}
	if !config.Rules[0].Output.Channels[0].Overwrite {
		t.Errorf("Output.Channels[0].Overwrite = %v, want true", config.Rules[0].Output.Channels[0].Overwrite)
	}
	if config.Rules[0].Output.Channels[0].MarkerPrefix != "custom_marker" {
		t.Errorf("Output.Channels[0].MarkerPrefix = %v, want custom_marker", config.Rules[0].Output.Channels[0].MarkerPrefix)
	}
}

func TestParser_Parse_CommentAppendMode(t *testing.T) {
	// Test comment channel with overwrite: false (append mode)
	yamlContent := `
version: "1.0"
rules:
  - id: security
    description: Security Reviewer
    goals:
      areas:
        - security
    output:
      channels:
        - type: comment
          overwrite: false
`

	parser := NewParser()
	config, err := parser.Parse([]byte(yamlContent))

	if err != nil {
		t.Fatalf("Parse() unexpected error: %v", err)
	}

	if config.Rules[0].Output.Channels[0].Overwrite {
		t.Errorf("Output.Channels[0].Overwrite = %v, want false", config.Rules[0].Output.Channels[0].Overwrite)
	}
}

func TestParser_Parse_ValidWebhookConfig(t *testing.T) {
	yamlContent := `
version: "1.0"
rules:
  - id: security
    description: Security Reviewer
    goals:
      areas:
        - security
    output:
      channels:
        - type: webhook
          url: https://example.com/callback
          header_secret: my-secret-key
          timeout: 30
          max_retries: 3
`

	parser := NewParser()
	config, err := parser.Parse([]byte(yamlContent))

	if err != nil {
		t.Fatalf("Parse() unexpected error: %v", err)
	}

	if config.Rules[0].Output == nil {
		t.Fatal("Output should not be nil")
	}

	if len(config.Rules[0].Output.Channels) != 1 {
		t.Fatalf("len(Output.Channels) = %d, want 1", len(config.Rules[0].Output.Channels))
	}

	item := config.Rules[0].Output.Channels[0]
	if item.Type != "webhook" {
		t.Errorf("Output.Channels[0].Type = %v, want webhook", item.Type)
	}
	if item.URL != "https://example.com/callback" {
		t.Errorf("Output.Channels[0].URL = %v, want https://example.com/callback", item.URL)
	}
	if item.HeaderSecret != "my-secret-key" {
		t.Errorf("Output.Channels[0].HeaderSecret = %v, want my-secret-key", item.HeaderSecret)
	}
	if item.Timeout != 30 {
		t.Errorf("Output.Channels[0].Timeout = %v, want 30", item.Timeout)
	}
	if item.MaxRetries != 3 {
		t.Errorf("Output.Channels[0].MaxRetries = %v, want 3", item.MaxRetries)
	}
}

func TestParser_Parse_WebhookMissingURL(t *testing.T) {
	yamlContent := `
version: "1.0"
rules:
  - id: security
    description: Security Reviewer
    goals:
      areas:
        - security
    output:
      channels:
        - type: webhook
          header_secret: my-secret
`

	parser := NewParser()
	_, err := parser.Parse([]byte(yamlContent))

	if err == nil {
		t.Error("Parse() expected error for webhook missing URL, got nil")
	}
}

func TestParser_Parse_WebhookInvalidTimeout(t *testing.T) {
	// Test timeout too small (less than 30)
	yamlContent := `
version: "1.0"
rules:
  - id: security
    description: Security Reviewer
    goals:
      areas:
        - security
    output:
      channels:
        - type: webhook
          url: https://example.com/callback
          timeout: 10
`

	parser := NewParser()
	_, err := parser.Parse([]byte(yamlContent))

	if err == nil {
		t.Error("Parse() expected error for webhook timeout too small, got nil")
	}

	// Test timeout too large (more than 300)
	yamlContent2 := `
version: "1.0"
rules:
  - id: security
    description: Security Reviewer
    goals:
      areas:
        - security
    output:
      channels:
        - type: webhook
          url: https://example.com/callback
          timeout: 500
`

	_, err = parser.Parse([]byte(yamlContent2))

	if err == nil {
		t.Error("Parse() expected error for webhook timeout too large, got nil")
	}
}

func TestParser_Parse_WebhookInvalidMaxRetries(t *testing.T) {
	// Test max_retries too small (less than 3)
	yamlContent := `
version: "1.0"
rules:
  - id: security
    description: Security Reviewer
    goals:
      areas:
        - security
    output:
      channels:
        - type: webhook
          url: https://example.com/callback
          max_retries: 1
`

	parser := NewParser()
	_, err := parser.Parse([]byte(yamlContent))

	if err == nil {
		t.Error("Parse() expected error for webhook max_retries too small, got nil")
	}

	// Test max_retries too large (more than 12)
	yamlContent2 := `
version: "1.0"
rules:
  - id: security
    description: Security Reviewer
    goals:
      areas:
        - security
    output:
      channels:
        - type: webhook
          url: https://example.com/callback
          max_retries: 20
`

	_, err = parser.Parse([]byte(yamlContent2))

	if err == nil {
		t.Error("Parse() expected error for webhook max_retries too large, got nil")
	}
}

func TestParser_Parse_WebhookInvalidHeaderSecret(t *testing.T) {
	// Test header_secret too short (less than 12 characters)
	yamlContent := `
version: "1.0"
rules:
  - id: security
    description: Security Reviewer
    goals:
      areas:
        - security
    output:
      channels:
        - type: webhook
          url: https://example.com/callback
          header_secret: short
`

	parser := NewParser()
	_, err := parser.Parse([]byte(yamlContent))

	if err == nil {
		t.Error("Parse() expected error for webhook header_secret too short, got nil")
	}

	// Test header_secret too long (more than 64 characters)
	yamlContent2 := `
version: "1.0"
rules:
  - id: security
    description: Security Reviewer
    goals:
      areas:
        - security
    output:
      channels:
        - type: webhook
          url: https://example.com/callback
          header_secret: this-is-a-very-long-secret-key-that-exceeds-the-maximum-allowed-length-of-64-characters
`

	_, err = parser.Parse([]byte(yamlContent2))

	if err == nil {
		t.Error("Parse() expected error for webhook header_secret too long, got nil")
	}
}

func TestParser_Parse_WebhookValidHeaderSecretEmpty(t *testing.T) {
	// Empty header_secret should be valid
	yamlContent := `
version: "1.0"
rules:
  - id: security
    description: Security Reviewer
    goals:
      areas:
        - security
    output:
      channels:
        - type: webhook
          url: https://example.com/callback
`

	parser := NewParser()
	_, err := parser.Parse([]byte(yamlContent))

	if err != nil {
		t.Errorf("Parse() unexpected error for webhook with empty header_secret: %v", err)
	}
}

func TestParser_Parse_ValidTones(t *testing.T) {
	tones := []string{"strict", "constructive", "neutral", "friendly", "professional"}

	for _, tone := range tones {
		t.Run(tone, func(t *testing.T) {
			yamlContent := `
version: "1.0"
rules:
  - id: test
    description: Test Reviewer
    goals:
      areas:
        - test
    output:
      style:
        tone: ` + tone + `
`
			parser := NewParser()
			_, err := parser.Parse([]byte(yamlContent))

			if err != nil {
				t.Errorf("Parse() unexpected error for tone %s: %v", tone, err)
			}
		})
	}
}

func TestParser_StrictMode(t *testing.T) {
	// In strict mode, goal areas are required
	yamlContent := `
version: "1.0"
rules:
  - id: test
    description: Test Reviewer
    goals: {}
`

	strictParser := NewStrictParser()
	_, err := strictParser.Parse([]byte(yamlContent))

	if err == nil {
		t.Error("StrictParser.Parse() expected error for empty goal areas, got nil")
	}

	// Non-strict mode should pass
	normalParser := NewParser()
	_, err = normalParser.Parse([]byte(yamlContent))

	if err != nil {
		t.Errorf("Parser.Parse() unexpected error: %v", err)
	}
}

func TestParser_ApplyDefaults(t *testing.T) {
	yamlContent := `
version: "1.0"
rule_base:
  agent:
    type: gemini
  constraints:
    severity:
      min_report: high
rules:
  - id: test
    description: Test Reviewer
    goals:
      areas:
        - test
`

	parser := NewParser()
	config, err := parser.Parse([]byte(yamlContent))

	if err != nil {
		t.Fatalf("Parse() unexpected error: %v", err)
	}

	rule := config.Rules[0]

	// Validate rule base is correctly applied
	if rule.Agent.Type != "gemini" {
		t.Errorf("Agent.Type = %v, want gemini", rule.Agent.Type)
	}
	if rule.Constraints == nil {
		t.Error("Constraints should not be nil")
		return
	}
	if rule.Constraints.Severity == nil {
		t.Error("Constraints.Severity should not be nil")
		return
	}
	if rule.Constraints.Severity.MinReport != "high" {
		t.Errorf("Constraints.Severity.MinReport = %v, want high", rule.Constraints.Severity.MinReport)
	}
}

func TestParser_Parse_OutputWithLanguage(t *testing.T) {
	yamlContent := `
version: "1.0"
rules:
  - id: test
    description: Test Reviewer
    goals:
      areas:
        - test
    constraints:
      scope_control:
        - Review only code changed in this PR
    output:
      style:
        tone: constructive
        language: Chinese
`

	parser := NewParser()
	config, err := parser.Parse([]byte(yamlContent))

	if err != nil {
		t.Fatalf("Parse() unexpected error: %v", err)
	}

	rule := config.Rules[0]

	if rule.Constraints == nil {
		t.Error("Constraints should not be nil")
		return
	}

	if len(rule.Constraints.ScopeControl) != 1 {
		t.Errorf("len(Constraints.ScopeControl) = %d, want 1", len(rule.Constraints.ScopeControl))
	}

	if rule.Output == nil {
		t.Error("Output should not be nil")
		return
	}

	if rule.Output.Style == nil {
		t.Error("Output.Style should not be nil")
		return
	}

	if rule.Output.Style.Language != "Chinese" {
		t.Errorf("Output.Style.Language = %v, want Chinese", rule.Output.Style.Language)
	}
}

func TestParser_Parse_ValidExtraFields(t *testing.T) {
	yamlContent := `
version: "1.0"
rules:
  - id: security
    description: Security review
    goals:
      areas:
        - security-vulnerabilities
    output:
      schema:
        extra_fields:
          - name: vulnerability_type
            type: string
            description: "Type of security vulnerability"
            required: true
            enum: ["sql_injection", "xss", "csrf"]
          - name: cve_id
            type: string
            description: "CVE identifier if applicable"
`
	parser := NewParser()
	config, err := parser.Parse([]byte(yamlContent))

	if err != nil {
		t.Fatalf("Parse() unexpected error: %v", err)
	}

	rule := config.Rules[0]
	if rule.Output == nil || rule.Output.Schema == nil {
		t.Fatal("Output.Schema should not be nil")
	}

	if len(rule.Output.Schema.ExtraFields) != 2 {
		t.Errorf("len(ExtraFields) = %d, want 2", len(rule.Output.Schema.ExtraFields))
	}

	// Check first field
	if rule.Output.Schema.ExtraFields[0].Name != "vulnerability_type" {
		t.Errorf("ExtraFields[0].Name = %v, want vulnerability_type", rule.Output.Schema.ExtraFields[0].Name)
	}
	if !rule.Output.Schema.ExtraFields[0].Required {
		t.Error("ExtraFields[0].Required should be true")
	}
	if len(rule.Output.Schema.ExtraFields[0].Enum) != 3 {
		t.Errorf("len(ExtraFields[0].Enum) = %d, want 3", len(rule.Output.Schema.ExtraFields[0].Enum))
	}
}

func TestParser_Parse_ExtraFieldsEmptyName(t *testing.T) {
	yamlContent := `
version: "1.0"
rules:
  - id: test
    description: Test rule
    goals:
      areas:
        - security-vulnerabilities
    output:
      schema:
        extra_fields:
          - name: ""
            type: string
            description: "Empty name field"
`
	parser := NewParser()
	_, err := parser.Parse([]byte(yamlContent))

	if err == nil {
		t.Error("Parse() should return error for extra field with empty name")
	}
}

func TestParser_Parse_ExtraFieldsInvalidType(t *testing.T) {
	yamlContent := `
version: "1.0"
rules:
  - id: test
    description: Test rule
    goals:
      areas:
        - security-vulnerabilities
    output:
      schema:
        extra_fields:
          - name: test_field
            type: invalid_type
            description: "Invalid type field"
`
	parser := NewParser()
	_, err := parser.Parse([]byte(yamlContent))

	if err == nil {
		t.Error("Parse() should return error for extra field with invalid type")
	}
}

func TestParser_Parse_ExtraFieldsReservedName(t *testing.T) {
	yamlContent := `
version: "1.0"
rules:
  - id: test
    description: Test rule
    goals:
      areas:
        - security-vulnerabilities
    output:
      schema:
        extra_fields:
          - name: severity
            type: string
            description: "Trying to override reserved field"
`
	parser := NewParser()
	_, err := parser.Parse([]byte(yamlContent))

	if err == nil {
		t.Error("Parse() should return error for extra field with reserved name")
	}
}

func TestParser_Parse_ExtraFieldsDuplicateName(t *testing.T) {
	yamlContent := `
version: "1.0"
rules:
  - id: test
    description: Test rule
    goals:
      areas:
        - security-vulnerabilities
    output:
      schema:
        extra_fields:
          - name: custom_field
            type: string
            description: "First field"
          - name: custom_field
            type: string
            description: "Duplicate field"
`
	parser := NewParser()
	_, err := parser.Parse([]byte(yamlContent))

	if err == nil {
		t.Error("Parse() should return error for duplicate extra field names")
	}
}

func TestParser_Parse_ExtraFieldsEnumOnNonString(t *testing.T) {
	yamlContent := `
version: "1.0"
rules:
  - id: test
    description: Test rule
    goals:
      areas:
        - security-vulnerabilities
    output:
      schema:
        extra_fields:
          - name: count_field
            type: integer
            description: "Integer field with enum"
            enum: ["one", "two", "three"]
`
	parser := NewParser()
	_, err := parser.Parse([]byte(yamlContent))

	if err == nil {
		t.Error("Parse() should return error for enum on non-string type")
	}
}

func TestParser_Parse_ExtraFieldsMissingDescription(t *testing.T) {
	yamlContent := `
version: "1.0"
rules:
  - id: test
    description: Test rule
    goals:
      areas:
        - security-vulnerabilities
    output:
      schema:
        extra_fields:
          - name: test_field
            type: string
`
	parser := NewParser()
	_, err := parser.Parse([]byte(yamlContent))

	if err == nil {
		t.Error("Parse() should return error for extra field without description")
	}
}

func TestParser_Parse_SchemaNotInheritedFromRuleBase(t *testing.T) {
	yamlContent := `
version: "1.0"
rule_base:
  output:
    schema:
      extra_fields:
        - name: base_field
          type: string
          description: "Field from rule_base"
rules:
  - id: test
    description: Test rule
    goals:
      areas:
        - security-vulnerabilities
`
	parser := NewParser()
	config, err := parser.Parse([]byte(yamlContent))

	if err != nil {
		t.Fatalf("Parse() unexpected error: %v", err)
	}

	rule := config.Rules[0]
	// Schema should NOT be inherited from rule_base
	if rule.Output != nil && rule.Output.Schema != nil && len(rule.Output.Schema.ExtraFields) > 0 {
		t.Error("Schema.ExtraFields should NOT be inherited from rule_base")
	}
}
