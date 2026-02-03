// Package executor handles review rule execution.
// This file contains format instruction building logic.
package executor

import (
	"go.uber.org/zap"

	"github.com/verustcode/verustcode/consts"
	"github.com/verustcode/verustcode/internal/dsl"
	"github.com/verustcode/verustcode/internal/prompt"
	"github.com/verustcode/verustcode/pkg/logger"
)

// BuildFormatInstructions generates format instructions for LLM output.
// LLM always returns JSON structured data; output channels handle format conversion.
func BuildFormatInstructions(rule *dsl.ReviewRuleConfig, buildCtx *prompt.BuildContext) string {
	if rule == nil || rule.Output == nil {
		logger.Debug("Rule or Output is nil, skipping format instructions",
			zap.String("rule_id", func() string {
				if rule != nil {
					return rule.ID
				}
				return ""
			}()),
		)
		return ""
	}

	// LLM always returns JSON - format conversion happens at channel level
	format := consts.OutputFormatJSON

	// Get language from output style or build context
	language := ""
	if rule.Output.Style != nil && rule.Output.Style.Language != "" {
		language = rule.Output.Style.Language
	} else if buildCtx != nil && buildCtx.OutputLanguage != "" {
		language = buildCtx.OutputLanguage
	}

	// Log schema configuration
	extraFieldsCount := 0
	if rule.Output.Schema != nil && len(rule.Output.Schema.ExtraFields) > 0 {
		extraFieldsCount = len(rule.Output.Schema.ExtraFields)
	}

	// Check if history_compare is enabled
	historyCompareEnabled := rule.HistoryCompare != nil && rule.HistoryCompare.Enabled

	logger.Debug("Building format instructions for rule (LLM always returns JSON)",
		zap.String("rule_id", rule.ID),
		zap.String("format", format),
		zap.String("language", language),
		zap.Int("extra_fields_count", extraFieldsCount),
		zap.Bool("has_schema", rule.Output.Schema != nil),
		zap.Bool("history_compare_enabled", historyCompareEnabled),
	)

	// Build format instructions using the FormatInstructionBuilder
	// Always use JSON format for LLM output
	// When history_compare is enabled, status field becomes required
	formatBuilder := prompt.NewFormatInstructionBuilder()
	result := formatBuilder.BuildWithOptions(format, rule.Output.Schema, language, historyCompareEnabled)

	logger.Debug("Format instructions generated",
		zap.String("rule_id", rule.ID),
		zap.Int("instructions_length", len(result)),
		zap.Bool("is_empty", result == ""),
	)

	return result
}
