// Package report provides report generation functionality.
package report

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"go.uber.org/zap"

	"github.com/verustcode/verustcode/internal/agent/base"
	"github.com/verustcode/verustcode/internal/config"
	"github.com/verustcode/verustcode/internal/dsl"
	"github.com/verustcode/verustcode/internal/llm"
	"github.com/verustcode/verustcode/internal/model"
	"github.com/verustcode/verustcode/pkg/logger"
)

// ReportStructure represents the generated report structure (Phase 1 output).
// This is the fixed JSON schema that AI must return in Phase 1.
type ReportStructure struct {
	Title    string             `json:"title"`    // Report title
	Summary  string             `json:"summary"`  // Brief summary
	Sections []GeneratedSection `json:"sections"` // Generated sections
}

// GeneratedSection represents a section in the generated structure.
// Sections are dynamically determined by AI based on project analysis.
type GeneratedSection struct {
	ID          string             `json:"id"`                    // Section identifier
	Title       string             `json:"title"`                 // Section title
	Description string             `json:"description"`           // What this section should cover
	Subsections []GeneratedSection `json:"subsections,omitempty"` // Optional subsections (for hierarchical structure)
}

// StructureGenerator generates report structure (Phase 1)
type StructureGenerator struct {
	agent          base.Agent
	configProvider config.ConfigProvider
}

// NewStructureGenerator creates a new structure generator
func NewStructureGenerator(agent base.Agent, configProvider config.ConfigProvider) *StructureGenerator {
	return &StructureGenerator{
		agent:          agent,
		configProvider: configProvider,
	}
}

// Generate generates the report structure for a given report
func (g *StructureGenerator) Generate(ctx context.Context, report *model.Report, repoPath string) (*ReportStructure, error) {
	logger.Info("Generating report structure",
		zap.String("report_id", report.ID),
		zap.String("report_type", report.ReportType),
		zap.String("repo_path", repoPath),
	)

	// Get report type definition and DSL config (dynamically load from filesystem)
	typeDef, err := ScanReportType(report.ReportType)
	if err != nil {
		return nil, fmt.Errorf("unknown report type: %w", err)
	}

	config := typeDef.Config
	if config == nil {
		return nil, fmt.Errorf("report type '%s' has no DSL configuration", report.ReportType)
	}

	// Build prompt for structure generation using DSL config
	// The agent will analyze the project autonomously
	prompt := g.buildStructurePrompt(typeDef, config)

	// Log complete prompt for debugging (Phase 1: structure generation)
	logger.Debug("Structure generation prompt",
		zap.String("report_type", report.ReportType),
		zap.String("prompt", prompt),
	)

	// Call agent to generate structure
	logger.Info("Calling agent to generate report structure",
		zap.String("report_type", report.ReportType),
		zap.String("agent", config.Agent.GetType()),
		zap.String("model", config.Agent.Model),
	)

	req := &base.ReviewRequest{
		RepoPath:  repoPath,
		RequestID: fmt.Sprintf("structure-%s", report.ID),
		Model:     config.Agent.Model, // Use model from DSL configuration
	}
	result, err := g.agent.ExecuteWithPrompt(ctx, req, prompt)
	if err != nil {
		logger.Error("Agent call failed for structure generation",
			zap.String("report_id", report.ID),
			zap.String("report_type", report.ReportType),
			zap.Error(err),
		)
		return nil, fmt.Errorf("agent call failed: %w", err)
	}
	if !result.Success {
		// Log the error response from agent
		logger.Error("Agent returned error for structure generation",
			zap.String("report_id", report.ID),
			zap.String("report_type", report.ReportType),
			zap.String("agent_error", result.Error),
			zap.String("raw_response", result.Text),
			zap.Int("response_length", len(result.Text)),
		)
		return nil, fmt.Errorf("agent returned error: %s (raw response length: %d)", result.Error, len(result.Text))
	}

	// Get response text
	response := result.Text

	// Parse response
	structure, err := g.parseStructureResponse(response)
	if err != nil {
		// Log the raw response for debugging
		logger.Error("Failed to parse structure response",
			zap.String("report_id", report.ID),
			zap.String("report_type", report.ReportType),
			zap.Error(err),
			zap.String("raw_response", response),
			zap.Int("response_length", len(response)),
		)
		return nil, fmt.Errorf("failed to parse structure response (raw response length: %d): %w", len(response), err)
	}

	logger.Info("Report structure generated",
		zap.String("title", structure.Title),
		zap.Int("sections", len(structure.Sections)),
	)

	return structure, nil
}

// buildStructurePrompt builds the prompt for structure generation using DSL config.
// Phase 1 prompt composition:
// 1. Role (structure.description)
// 2. Report type information
// 3. Reference documents hints (from structure.reference_docs)
// 4. Goals (structure.goals.areas and structure.goals.avoid)
// 5. Constraints (structure.constraints)
// 6. Output format instructions (fixed JSON schema)
func (g *StructureGenerator) buildStructurePrompt(typeDef *ReportTypeDefinition, reportConfig *dsl.ReportConfig) string {
	var sb strings.Builder

	// Role description from DSL (first section)
	if reportConfig.Structure.Description != "" {
		sb.WriteString("## Role\n")
		sb.WriteString(reportConfig.Structure.Description)
		sb.WriteString("\n\n")
	}

	// Report type information (after Role)
	sb.WriteString("## Report Type\n")
	sb.WriteString(fmt.Sprintf("- **Type**: %s\n", typeDef.Name))
	sb.WriteString(fmt.Sprintf("- **Description**: %s\n\n", typeDef.Description))

	// Reference documents hints from DSL config
	if len(reportConfig.Structure.ReferenceDocs) > 0 {
		sb.WriteString("**Suggested reference documents:**\n")
		for _, docPath := range reportConfig.Structure.ReferenceDocs {
			sb.WriteString(fmt.Sprintf("- %s\n", docPath))
		}
		sb.WriteString("\n")
	}

	// Goals section (combining topics and avoid)
	hasTopics := len(reportConfig.Structure.Goals.Topics) > 0
	hasAvoid := len(reportConfig.Structure.Goals.Avoid) > 0
	if hasTopics || hasAvoid {
		sb.WriteString("## Goals\n\n")

		// Focus topics from DSL goals.topics
		if hasTopics {
			sb.WriteString("### Focus Topics\n")
			sb.WriteString("When generating the report structure, focus on these topics:\n\n")
			for _, topic := range reportConfig.Structure.Goals.Topics {
				sb.WriteString(fmt.Sprintf("- %s\n", topic))
			}
			sb.WriteString("\n")
		}

		// Avoid list from DSL goals.avoid
		if hasAvoid {
			sb.WriteString("### Things to Avoid\n")
			sb.WriteString("Do NOT include these in the report structure:\n\n")
			for _, avoid := range reportConfig.Structure.Goals.Avoid {
				sb.WriteString(fmt.Sprintf("- %s\n", avoid))
			}
			sb.WriteString("\n")
		}
	}

	// Constraints from DSL
	if len(reportConfig.Structure.Constraints) > 0 {
		sb.WriteString("## Constraints\n")
		for _, constraint := range reportConfig.Structure.Constraints {
			sb.WriteString(fmt.Sprintf("- %s\n", constraint))
		}
		sb.WriteString("\n")
	}

	// Output format (fixed JSON schema)
	sb.WriteString("## Output Format\n")
	sb.WriteString("Return a JSON object with the following structure:\n")
	sb.WriteString("```json\n")

	// Use different schema based on nested flag
	if reportConfig.Structure.Nested {
		// Two-level nested structure
		sb.WriteString(`{
  "title": "Report title based on project name and report type",
  "summary": "Brief one-paragraph summary of the project",
  "sections": [
    {
      "id": "section_id",
      "title": "Section Title (Parent)",
      "description": "Brief description of this parent section",
      "subsections": [
        {
          "id": "subsection_id_1",
          "title": "Subsection Title 1",
          "description": "Detailed description of what this subsection should cover"
        },
        {
          "id": "subsection_id_2",
          "title": "Subsection Title 2",
          "description": "Detailed description of what this subsection should cover"
        }
      ]
    }
  ]
}
`)
	} else {
		// Flat structure (default)
		sb.WriteString(`{
  "title": "Report title based on project name and report type",
  "summary": "Brief one-paragraph summary of the project",
  "sections": [
    {
      "id": "section_id",
      "title": "Section Title",
      "description": "Detailed description of what this section should cover"
    }
  ]
}
`)
	}
	sb.WriteString("```\n\n")

	// Critical constraint: output as plain text only, no file creation
	// This is essential to prevent agents from writing files instead of returning content
	sb.WriteString("**CRITICAL: Output as plain text only. Do NOT create any files.**\n\n")

	// Language instruction - use PromptInstruction() to get human-readable language name
	// Priority: report yaml output.style.language > config.Report.OutputLanguage > auto-detect
	outputLanguage := reportConfig.Output.Style.Language
	if outputLanguage == "" {
		// Get from database config (real-time read)
		if reportCfg, err := g.configProvider.GetReportConfig(); err == nil && reportCfg != nil {
			outputLanguage = reportCfg.OutputLanguage
		}
	}
	if outputLanguage != "" {
		if langCfg, err := config.ParseLanguage(outputLanguage); err == nil {
			sb.WriteString(fmt.Sprintf("**Output Language**: All content must be in %s.\n\n", langCfg.PromptInstruction()))
		} else {
			// Fallback to original value if parsing fails
			logger.Warn("Failed to parse output language, using raw value",
				zap.String("language", outputLanguage),
				zap.Error(err),
			)
			sb.WriteString(fmt.Sprintf("**Output Language**: All content must be in %s.\n\n", outputLanguage))
		}
	}

	sb.WriteString("**Important:**\n")
	sb.WriteString("- Do NOT write any files to disk. Return ONLY the JSON content directly.\n")
	sb.WriteString("- Do NOT create, save, or write any files (e.g., .md, .json, .txt).\n")
	sb.WriteString("- Analyze the actual codebase to generate appropriate sections\n")
	sb.WriteString("- Generate sections based on actual project content and the focus areas above\n")
	sb.WriteString("- Each section should have a clear scope and purpose\n")

	// Add nested structure-specific instructions
	if reportConfig.Structure.Nested {
		sb.WriteString("- Use two-level structure: parent sections contain subsections\n")
		sb.WriteString("- Each parent section MUST have at least one subsection\n")
		sb.WriteString("- Only subsections (leaf nodes) will have content generated\n")
		sb.WriteString("- Parent sections should be broad categories, subsections should be specific topics\n")
	}

	sb.WriteString("- Return ONLY the JSON, no additional text or file operations\n")

	return sb.String()
}

// parseStructureResponse parses the LLM response into a ReportStructure
func (g *StructureGenerator) parseStructureResponse(response string) (*ReportStructure, error) {
	// Use the shared llm.ExtractJSON function which correctly handles nested JSON
	// by finding first '{' and last '}' in the content
	jsonStr, err := llm.ExtractJSON(response)
	if err != nil {
		// Include raw response snippet in error for debugging
		responseSnippet := response
		if len(responseSnippet) > 500 {
			responseSnippet = responseSnippet[:500] + "..."
		}
		return nil, fmt.Errorf("no JSON found in response (raw response preview: %q): %w", responseSnippet, err)
	}

	var structure ReportStructure
	if err := json.Unmarshal([]byte(jsonStr), &structure); err != nil {
		// Include JSON snippet in error for debugging
		jsonSnippet := jsonStr
		if len(jsonSnippet) > 500 {
			jsonSnippet = jsonSnippet[:500] + "..."
		}
		return nil, fmt.Errorf("failed to parse JSON (extracted JSON preview: %q): %w", jsonSnippet, err)
	}

	// Validate basic structure
	if structure.Title == "" {
		return nil, fmt.Errorf("structure missing title")
	}
	if len(structure.Sections) == 0 {
		return nil, fmt.Errorf("structure has no sections")
	}

	return &structure, nil
}

// StructureToModel converts ReportStructure to model.ReportSection slice.
// Supports hierarchical structure with parent sections and subsections.
// When a section has subsections:
//   - The parent section has IsLeaf=false
//   - Subsections have ParentSectionID set and IsLeaf=true
//
// Only leaf sections will have content generated in Phase 2.
func StructureToModel(structure *ReportStructure, reportID string) []model.ReportSection {
	var sections []model.ReportSection
	globalIndex := 0

	for _, s := range structure.Sections {
		// Check if this section has subsections
		hasSubsections := len(s.Subsections) > 0

		// Create the parent/main section
		parentSection := model.ReportSection{
			ReportID:        reportID,
			SectionIndex:    globalIndex,
			SectionID:       s.ID,
			ParentSectionID: nil, // Top-level section has no parent
			IsLeaf:          !hasSubsections,
			Title:           s.Title,
			Description:     s.Description,
			Status:          model.SectionStatusPending,
		}
		sections = append(sections, parentSection)
		globalIndex++

		// If has subsections, add them as child sections
		if hasSubsections {
			parentID := s.ID
			for _, sub := range s.Subsections {
				subSection := model.ReportSection{
					ReportID:        reportID,
					SectionIndex:    globalIndex,
					SectionID:       sub.ID,
					ParentSectionID: &parentID, // Set parent reference
					IsLeaf:          true,      // Subsections are always leaf nodes
					Title:           sub.Title,
					Description:     sub.Description,
					Status:          model.SectionStatusPending,
				}
				sections = append(sections, subSection)
				globalIndex++
			}
		}
	}

	return sections
}

// CountLeafSections counts the number of leaf sections (sections that will have content generated)
func CountLeafSections(structure *ReportStructure) int {
	count := 0
	for _, s := range structure.Sections {
		if len(s.Subsections) > 0 {
			// Parent section, count subsections
			count += len(s.Subsections)
		} else {
			// Leaf section
			count++
		}
	}
	return count
}

// StructureToJSONMap converts ReportStructure to JSONMap for storage
func StructureToJSONMap(structure *ReportStructure) model.JSONMap {
	data, err := json.Marshal(structure)
	if err != nil {
		return nil
	}
	var result model.JSONMap
	if err := json.Unmarshal(data, &result); err != nil {
		return nil
	}
	return result
}
