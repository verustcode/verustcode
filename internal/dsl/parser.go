package dsl

import (
	"fmt"
	"strings"

	"go.uber.org/zap"
	"gopkg.in/yaml.v3"

	"github.com/verustcode/verustcode/consts"
	"github.com/verustcode/verustcode/pkg/errors"
	"github.com/verustcode/verustcode/pkg/logger"
)

// Validation constants for webhook configuration
const (
	// MinWebhookTimeout is the minimum timeout in seconds for webhook requests
	MinWebhookTimeout = 30
	// MaxWebhookTimeout is the maximum timeout in seconds for webhook requests
	MaxWebhookTimeout = 300

	// MinWebhookRetries is the minimum number of retry attempts for webhooks
	MinWebhookRetries = 3
	// MaxWebhookRetries is the maximum number of retry attempts for webhooks
	MaxWebhookRetries = 12

	// MinHeaderSecretLength is the minimum length for webhook header secret
	MinHeaderSecretLength = 12
	// MaxHeaderSecretLength is the maximum length for webhook header secret
	MaxHeaderSecretLength = 64
)

// Validation constants for multi-run configuration
const (
	// MinMultiRunRuns is the minimum number of runs for multi-run to be enabled
	MinMultiRunRuns = 2
	// MaxMultiRunRuns is the maximum number of runs allowed
	MaxMultiRunRuns = 3
)

// Validation constants for reference documentation
const (
	// MaxReferenceDocs is the maximum number of reference documentation files allowed
	MaxReferenceDocs = 5
)

// Parser parses and validates DSL configuration
type Parser struct {
	// strict mode enables stricter validation
	strict bool
}

// NewParser creates a new DSL parser
func NewParser() *Parser {
	return &Parser{
		strict: false,
	}
}

// NewStrictParser creates a new DSL parser with strict validation
func NewStrictParser() *Parser {
	return &Parser{
		strict: true,
	}
}

// Parse parses YAML content into ReviewRulesConfig
func (p *Parser) Parse(data []byte) (*ReviewRulesConfig, error) {
	var config ReviewRulesConfig

	if err := yaml.Unmarshal(data, &config); err != nil {
		return nil, errors.Wrap(errors.ErrCodeConfigInvalid, "failed to parse YAML", err)
	}

	// Apply rule base to each review rule
	p.applyRuleBase(&config)

	// Validate the configuration
	if err := p.Validate(&config); err != nil {
		return nil, err
	}

	return &config, nil
}

// applyRuleBase applies base configuration to review rules
func (p *Parser) applyRuleBase(config *ReviewRulesConfig) {
	ruleBase := config.RuleBase
	defaultConfig := DefaultReviewRuleConfig()

	for i := range config.Rules {
		rule := &config.Rules[i]

		// Apply Agent default (merge type and model separately)
		if rule.Agent.Type == "" {
			if ruleBase != nil && ruleBase.Agent.Type != "" {
				rule.Agent.Type = ruleBase.Agent.Type
			} else {
				rule.Agent.Type = defaultConfig.Agent.Type
			}
		}
		if rule.Agent.Model == "" {
			if ruleBase != nil && ruleBase.Agent.Model != "" {
				rule.Agent.Model = ruleBase.Agent.Model
			}
			// Note: defaultConfig.Agent.Model is empty, so no else branch needed
		}

		// Apply Constraints defaults
		if rule.Constraints == nil {
			if ruleBase != nil && ruleBase.Constraints != nil {
				rule.Constraints = p.mergeConstraints(nil, ruleBase.Constraints)
			} else {
				rule.Constraints = defaultConfig.Constraints
			}
		} else {
			// Merge with rule base
			if ruleBase != nil && ruleBase.Constraints != nil {
				rule.Constraints = p.mergeConstraints(rule.Constraints, ruleBase.Constraints)
			} else {
				rule.Constraints = p.mergeConstraints(rule.Constraints, defaultConfig.Constraints)
			}
		}

		// Apply Output defaults
		if rule.Output == nil {
			if ruleBase != nil && ruleBase.Output != nil {
				rule.Output = p.mergeOutput(nil, ruleBase.Output)
			} else {
				rule.Output = defaultConfig.Output
			}
		} else {
			// Merge with rule base
			if ruleBase != nil && ruleBase.Output != nil {
				rule.Output = p.mergeOutput(rule.Output, ruleBase.Output)
			} else {
				rule.Output = p.mergeOutput(rule.Output, defaultConfig.Output)
			}
		}
	}
}

// mergeConstraints merges two ConstraintsConfig, with override taking precedence
func (p *Parser) mergeConstraints(override, base *ConstraintsConfig) *ConstraintsConfig {
	if override == nil {
		return base
	}
	if base == nil {
		return override
	}

	result := &ConstraintsConfig{}

	// Merge ScopeControl
	if len(override.ScopeControl) > 0 {
		result.ScopeControl = override.ScopeControl
	} else {
		result.ScopeControl = base.ScopeControl
	}

	// Merge Severity
	if override.Severity != nil {
		result.Severity = p.mergeSeverity(override.Severity, base.Severity)
	} else {
		result.Severity = base.Severity
	}

	// Merge Duplicates
	if override.Duplicates != nil {
		result.Duplicates = override.Duplicates
	} else {
		result.Duplicates = base.Duplicates
	}

	return result
}

// mergeSeverity merges two SeverityConfig, with override taking precedence
func (p *Parser) mergeSeverity(override, base *SeverityConfig) *SeverityConfig {
	if override == nil {
		return base
	}
	if base == nil {
		return override
	}

	return &SeverityConfig{
		MinReport: mergeString(override.MinReport, base.MinReport),
	}
}

// mergeOutput merges two OutputConfig, with override taking precedence
// Note: Schema is NOT inherited from base - it can only be defined at rule level
func (p *Parser) mergeOutput(override, base *OutputConfig) *OutputConfig {
	if override == nil {
		// When no override, copy base but clear Schema (no inheritance)
		if base == nil {
			return nil
		}
		return &OutputConfig{
			Style:    base.Style,
			Channels: base.Channels,
			Schema:   nil, // Schema is not inherited
		}
	}
	if base == nil {
		return override
	}

	// Merge Style: deep merge
	var style *OutputStyleConfig
	if override.Style != nil {
		style = p.mergeOutputStyle(override.Style, base.Style)
	} else {
		style = base.Style
	}

	// Schema: only use override's schema (no inheritance from base)
	// extra_fields can only be defined at rule level, not in rule_base
	var schema *OutputSchemaConfig
	if override.Schema != nil {
		schema = override.Schema
	}
	// Note: intentionally not falling back to base.Schema

	return &OutputConfig{
		Style:    style,
		Channels: mergeOutputChannels(override.Channels, base.Channels),
		Schema:   schema,
	}
}

// mergeOutputChannels merges output channel lists
// Override completely replaces base (no merging)
func mergeOutputChannels(override, base []OutputItemConfig) []OutputItemConfig {
	if len(override) > 0 {
		return override
	}
	return base
}

// mergeOutputStyle merges two OutputStyleConfig, with override taking precedence
func (p *Parser) mergeOutputStyle(override, base *OutputStyleConfig) *OutputStyleConfig {
	if override == nil {
		return base
	}
	if base == nil {
		return override
	}

	return &OutputStyleConfig{
		Tone:     mergeString(override.Tone, base.Tone),
		Concise:  mergeBoolPtr(override.Concise, base.Concise),
		NoEmoji:  mergeBoolPtr(override.NoEmoji, base.NoEmoji),
		NoDate:   mergeBoolPtr(override.NoDate, base.NoDate),
		Language: mergeString(override.Language, base.Language),
	}
}

// Validate validates the ReviewRulesConfig
func (p *Parser) Validate(config *ReviewRulesConfig) error {
	if len(config.Rules) == 0 {
		return errors.New(errors.ErrCodeConfigInvalid, "at least one review rule is required")
	}

	// Track rule IDs for uniqueness check
	ids := make(map[string]bool)

	for i, rule := range config.Rules {
		if err := p.validateRule(&rule, i); err != nil {
			return err
		}

		// Check for duplicate IDs
		if ids[rule.ID] {
			return errors.New(errors.ErrCodeConfigInvalid,
				fmt.Sprintf("duplicate review rule ID: %s", rule.ID))
		}
		ids[rule.ID] = true
	}

	return nil
}

// validateRule validates a single review rule configuration
func (p *Parser) validateRule(rule *ReviewRuleConfig, index int) error {
	prefix := fmt.Sprintf("rule[%d]", index)

	// ID is required
	if rule.ID == "" {
		return errors.New(errors.ErrCodeConfigInvalid,
			fmt.Sprintf("%s: id is required", prefix))
	}

	// Validate Goals: at least one area is required in strict mode
	if len(rule.Goals.Areas) == 0 {
		if p.strict {
			return errors.New(errors.ErrCodeConfigInvalid,
				fmt.Sprintf("%s (%s): at least one goal area is required", prefix, rule.ID))
		}
	}

	// Validate area definitions (warn if unknown areas are used)
	if err := p.validateAreas(rule.Goals.Areas, prefix, rule.ID); err != nil {
		return err
	}

	// Validate ReferenceDocs: maximum 5 files allowed
	if len(rule.ReferenceDocs) > MaxReferenceDocs {
		return errors.New(errors.ErrCodeConfigInvalid,
			fmt.Sprintf("%s (%s): reference_docs cannot exceed %d files, got %d",
				prefix, rule.ID, MaxReferenceDocs, len(rule.ReferenceDocs)))
	}

	// Validate Constraints
	if rule.Constraints != nil {
		if err := p.validateConstraints(rule.Constraints, prefix, rule.ID); err != nil {
			return err
		}
	}

	// Validate Output
	if rule.Output != nil {
		if err := p.validateOutput(rule.Output, prefix, rule.ID); err != nil {
			return err
		}
	}

	// Validate MultiRun
	if rule.MultiRun != nil {
		if err := p.validateMultiRun(rule.MultiRun, prefix, rule.ID); err != nil {
			return err
		}
	}

	return nil
}

// validateConstraints validates constraints configuration
func (p *Parser) validateConstraints(constraints *ConstraintsConfig, prefix, id string) error {
	// Validate Severity
	if constraints.Severity != nil {
		if err := p.validateSeverity(constraints.Severity, prefix, id); err != nil {
			return err
		}
	}

	// Validate Duplicates similarity threshold
	if constraints.Duplicates != nil {
		sim := constraints.Duplicates.Similarity
		if sim != 0 && (sim < 0 || sim > 1) {
			return errors.New(errors.ErrCodeConfigInvalid,
				fmt.Sprintf("%s (%s): similarity must be between 0 and 1, got: %f",
					prefix, id, sim))
		}
	}

	return nil
}

// validateSeverity validates severity configuration
func (p *Parser) validateSeverity(severity *SeverityConfig, prefix, id string) error {
	// Validate min_report severity against system-defined levels
	if severity.MinReport != "" && !containsString(SeverityLevels, severity.MinReport) {
		return errors.New(errors.ErrCodeConfigInvalid,
			fmt.Sprintf("%s (%s): invalid min_report severity: %s (valid: %s)",
				prefix, id, severity.MinReport, strings.Join(SeverityLevels, ", ")))
	}

	return nil
}

// validateOutput validates output configuration
func (p *Parser) validateOutput(output *OutputConfig, prefix, id string) error {
	validFormats := []string{consts.OutputFormatMarkdown, consts.OutputFormatJSON}
	validTypes := []string{"file", "comment", "webhook"}
	validTones := []string{"strict", "constructive", "neutral", "friendly", "professional"}

	// Validate Style
	if output.Style != nil {
		if output.Style.Tone != "" {
			if !containsString(validTones, output.Style.Tone) {
				return errors.New(errors.ErrCodeConfigInvalid,
					fmt.Sprintf("%s (%s): invalid output style tone: %s (valid: %s)",
						prefix, id, output.Style.Tone, strings.Join(validTones, ", ")))
			}
		}
	}

	// Validate Channels
	for i, item := range output.Channels {
		// Type is required
		if item.Type == "" {
			return errors.New(errors.ErrCodeConfigInvalid,
				fmt.Sprintf("%s (%s): output.channels[%d]: type is required", prefix, id, i))
		}

		// Validate type
		if !containsString(validTypes, item.Type) {
			return errors.New(errors.ErrCodeConfigInvalid,
				fmt.Sprintf("%s (%s): output.channels[%d]: invalid type: %s (valid: %s)",
					prefix, id, i, item.Type, strings.Join(validTypes, ", ")))
		}

		// Validate channel format if specified
		if item.Format != "" {
			if !containsString(validFormats, item.Format) {
				return errors.New(errors.ErrCodeConfigInvalid,
					fmt.Sprintf("%s (%s): output.channels[%d]: invalid format: %s (valid: %s)",
						prefix, id, i, item.Format, strings.Join(validFormats, ", ")))
			}
		}

		// Validate webhook-specific options
		if item.Type == "webhook" {
			// URL is required for webhook type
			if item.URL == "" {
				return errors.New(errors.ErrCodeConfigInvalid,
					fmt.Sprintf("%s (%s): output.channels[%d]: url is required for webhook type",
						prefix, id, i))
			}

			// Validate timeout: 0 means default, otherwise must be within valid range
			if item.Timeout != 0 && (item.Timeout < MinWebhookTimeout || item.Timeout > MaxWebhookTimeout) {
				return errors.New(errors.ErrCodeConfigInvalid,
					fmt.Sprintf("%s (%s): output.channels[%d]: timeout must be between %d and %d seconds, got: %d",
						prefix, id, i, MinWebhookTimeout, MaxWebhookTimeout, item.Timeout))
			}

			// Validate max_retries: 0 means default, otherwise must be within valid range
			if item.MaxRetries != 0 && (item.MaxRetries < MinWebhookRetries || item.MaxRetries > MaxWebhookRetries) {
				return errors.New(errors.ErrCodeConfigInvalid,
					fmt.Sprintf("%s (%s): output.channels[%d]: max_retries must be between %d and %d, got: %d",
						prefix, id, i, MinWebhookRetries, MaxWebhookRetries, item.MaxRetries))
			}

			// Validate header_secret: empty is allowed, otherwise must be within valid length range
			secretLen := len(item.HeaderSecret)
			if secretLen > 0 && (secretLen < MinHeaderSecretLength || secretLen > MaxHeaderSecretLength) {
				return errors.New(errors.ErrCodeConfigInvalid,
					fmt.Sprintf("%s (%s): output.channels[%d]: header_secret must be %d-%d characters, got: %d",
						prefix, id, i, MinHeaderSecretLength, MaxHeaderSecretLength, secretLen))
			}
		}
	}

	// Validate Schema extra_fields
	if output.Schema != nil && len(output.Schema.ExtraFields) > 0 {
		if err := p.validateExtraFields(output.Schema.ExtraFields, prefix, id); err != nil {
			return err
		}
	}

	return nil
}

// validateMultiRun validates multi-run configuration
// Multi-run is automatically enabled when Runs >= 2
func (p *Parser) validateMultiRun(multiRun *MultiRunConfig, prefix, id string) error {
	// Skip validation if runs < MinMultiRunRuns (multi-run not enabled)
	if multiRun.Runs < MinMultiRunRuns {
		return nil
	}

	// Validate runs: must not exceed maximum
	if multiRun.Runs > MaxMultiRunRuns {
		return errors.New(errors.ErrCodeConfigInvalid,
			fmt.Sprintf("%s (%s): multi_run.runs must be at most %d, got: %d",
				prefix, id, MaxMultiRunRuns, multiRun.Runs))
	}

	return nil
}

// GetRuleByID returns a review rule by ID
func (config *ReviewRulesConfig) GetRuleByID(id string) *ReviewRuleConfig {
	for i := range config.Rules {
		if config.Rules[i].ID == id {
			return &config.Rules[i]
		}
	}
	return nil
}

// GetRuleIDs returns all review rule IDs
func (config *ReviewRulesConfig) GetRuleIDs() []string {
	ids := make([]string, len(config.Rules))
	for i, r := range config.Rules {
		ids[i] = r.ID
	}
	return ids
}

// validateAreas validates if all areas are defined
// In both strict and non-strict modes, only logs warnings without returning errors
// This maintains flexibility and allows users to use custom areas
func (p *Parser) validateAreas(areas []string, prefix, ruleID string) error {
	for _, area := range areas {
		if !IsValidArea(area) {
			logger.Warn("undefined area used in configuration",
				zap.String("rule_id", ruleID),
				zap.String("area", area),
				zap.String("hint", "this might be a custom area or a typo"),
			)
		}
	}
	return nil
}

// reservedFindingFields are base schema fields that cannot be overridden by extra_fields
var reservedFindingFields = []string{
	"severity", "title", "description", "category", "location", "suggestion", "code_snippet",
}

// validExtraFieldTypes are the allowed types for extra_fields
var validExtraFieldTypes = []string{"string", "integer", "boolean", "array"}

// validateExtraFields validates the extra_fields configuration in schema
func (p *Parser) validateExtraFields(fields []ExtraFieldConfig, prefix, id string) error {
	seenNames := make(map[string]bool)

	for i, field := range fields {
		// Name is required
		if field.Name == "" {
			return errors.New(errors.ErrCodeConfigInvalid,
				fmt.Sprintf("%s (%s): output.schema.extra_fields[%d]: name is required", prefix, id, i))
		}

		// Check for duplicate field names
		if seenNames[field.Name] {
			return errors.New(errors.ErrCodeConfigInvalid,
				fmt.Sprintf("%s (%s): output.schema.extra_fields[%d]: duplicate field name: %s", prefix, id, i, field.Name))
		}
		seenNames[field.Name] = true

		// Check if field name conflicts with reserved base schema fields
		if containsString(reservedFindingFields, field.Name) {
			return errors.New(errors.ErrCodeConfigInvalid,
				fmt.Sprintf("%s (%s): output.schema.extra_fields[%d]: cannot override reserved field: %s (reserved: %s)",
					prefix, id, i, field.Name, strings.Join(reservedFindingFields, ", ")))
		}

		// Type is required
		if field.Type == "" {
			return errors.New(errors.ErrCodeConfigInvalid,
				fmt.Sprintf("%s (%s): output.schema.extra_fields[%d] (%s): type is required", prefix, id, i, field.Name))
		}

		// Validate type
		if !containsString(validExtraFieldTypes, field.Type) {
			return errors.New(errors.ErrCodeConfigInvalid,
				fmt.Sprintf("%s (%s): output.schema.extra_fields[%d] (%s): invalid type: %s (valid: %s)",
					prefix, id, i, field.Name, field.Type, strings.Join(validExtraFieldTypes, ", ")))
		}

		// Description is required
		if field.Description == "" {
			return errors.New(errors.ErrCodeConfigInvalid,
				fmt.Sprintf("%s (%s): output.schema.extra_fields[%d] (%s): description is required", prefix, id, i, field.Name))
		}

		// Enum is only valid for string type
		if len(field.Enum) > 0 && field.Type != "string" {
			return errors.New(errors.ErrCodeConfigInvalid,
				fmt.Sprintf("%s (%s): output.schema.extra_fields[%d] (%s): enum is only valid for string type, got: %s",
					prefix, id, i, field.Name, field.Type))
		}
	}

	return nil
}
