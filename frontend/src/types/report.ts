// Report type configuration types
// These types correspond to the backend ReportConfig structure in internal/dsl/report_schema.go

/**
 * Agent configuration for report generation
 * Shared with Review DSL
 */
export interface AgentConfig {
  type?: string  // AI agent type (cursor, gemini, etc.)
  model?: string // Optional model override
}

/**
 * Style configuration for report output
 * Common fields with Review DSL + report-specific fields
 */
export interface ReportStyleConfig {
  // Common fields (same as Review DSL)
  tone?: string           // Writing tone: professional, friendly, technical, strict, constructive
  concise?: boolean       // Keep content concise
  no_emoji?: boolean      // Disable emoji
  language?: string       // Output language (ISO 639-1 code: en, zh-cn, zh-tw, ja, ko, fr, de, es)
  
  // Report-specific fields
  use_mermaid?: boolean   // Use Mermaid diagrams
  heading_level?: number  // Starting heading level (1-4, default: 2)
  max_section_length?: number  // Optional: max section length in characters
  include_line_numbers?: boolean  // Include line numbers in code references
}

/**
 * Output configuration for reports
 */
export interface ReportOutputConfig {
  style?: ReportStyleConfig
}

/**
 * Goals configuration for report phases
 * Unlike Review DSL which uses predefined area IDs, Report DSL uses free-text topics
 */
export interface ReportGoalsConfig {
  topics?: string[]  // Focus topics for this phase
  avoid?: string[]   // Things to avoid during this phase
}

/**
 * Section summary configuration
 * Each section generates a short summary during Phase 2 content generation
 */
export interface SectionSummaryConfig {
  max_length?: number  // Maximum characters for each section summary (default: 200)
}

/**
 * Phase 1: Structure Generation configuration
 * Output is always JSON with a fixed schema
 */
export interface StructurePhase {
  description?: string          // What this phase should accomplish
  nested?: boolean              // Enable two-level section structure (parent sections + subsections)
  goals?: ReportGoalsConfig     // Focus areas and things to avoid
  constraints?: string[]        // Constraints for structure generation
  reference_docs?: string[]     // Document files to include in the prompt
}

/**
 * Phase 2: Section Content Generation configuration
 * Output is always Markdown
 */
export interface SectionPhase {
  description?: string          // What this phase should accomplish
  goals?: ReportGoalsConfig     // Focus topics and things to avoid
  constraints?: string[]        // Constraints for section generation
  reference_docs?: string[]     // Document files to include in the prompt
  summary?: SectionSummaryConfig  // Section-level summary configuration
}

/**
 * Phase 3: Overall Report Summary Generation configuration
 * Output is always Markdown
 */
export interface SummaryPhase {
  description?: string          // What this phase should accomplish
  goals?: ReportGoalsConfig     // Focus topics and things to avoid for summary generation
  constraints?: string[]        // Constraints for summary generation
}

/**
 * Complete report configuration
 * Each report type is defined in a separate YAML file with this structure
 */
export interface ReportConfig {
  version?: string              // Version of the DSL schema
  id: string                    // Unique identifier for this report type
  name: string                  // Display name for this report type
  description?: string          // Details about what this report type generates
  agent?: AgentConfig           // AI agent configuration
  output?: ReportOutputConfig   // Output style and format configuration
  structure: StructurePhase     // Phase 1: structure generation configuration
  section: SectionPhase         // Phase 2: section content generation configuration
  summary: SummaryPhase         // Phase 3: report summary generation configuration
}

/**
 * Default values for report configuration
 */
export const DEFAULT_REPORT_CONFIG: Partial<ReportConfig> = {
  version: '1.0',
  agent: {
    type: 'cursor',
  },
  output: {
    style: {
      tone: 'professional',
      concise: true,
      no_emoji: true,
      language: 'zh-cn',
      use_mermaid: true,
      heading_level: 2,
      include_line_numbers: true,
    },
  },
  structure: {
    description: '分析项目并生成文档结构框架。',
    nested: false,
  },
  section: {
    description: '为每个章节生成详细的文档内容。',
    summary: {
      max_length: 200,
    },
  },
  summary: {
    description: '基于各章节摘要，生成项目的整体概览。',
  },
}
