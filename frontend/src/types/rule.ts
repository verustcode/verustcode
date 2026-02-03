/**
 * Rule configuration types (matching DSL schema)
 */

// Agent configuration (shared by Review DSL and Report DSL)
export interface AgentConfig {
  type?: string  // e.g., "cursor", "gemini"
  model?: string // e.g., "sonnet-4.5"
}

// Root configuration
export interface ReviewRulesConfig {
  version?: string
  rule_base?: RuleBaseConfig
  rules: ReviewRuleConfig[]
}

// Rule base configuration
export interface RuleBaseConfig {
  agent?: AgentConfig
  constraints?: ConstraintsConfig
  output?: OutputConfig
}

// Individual rule configuration
export interface ReviewRuleConfig {
  id: string
  role: string
  description?: string
  agent?: AgentConfig
  reference_docs?: string[]
  goals: GoalsConfig
  constraints?: ConstraintsConfig
  output?: OutputConfig
  multi_run?: MultiRunConfig
}

// Multi-run configuration
// Multi-run is automatically enabled when runs >= 2 (max 3 runs)
export interface MultiRunConfig {
  runs?: number
  models?: string[]
  merge_model?: string
}

// Goals configuration
export interface GoalsConfig {
  areas?: string[]
  avoid?: string[]
}

// Constraints configuration
export interface ConstraintsConfig {
  scope_control?: string[]
  focus_on_issues_only?: boolean
  severity?: SeverityConfig
  duplicates?: DuplicatesConfig
}

// Severity configuration
// Note: Severity levels are system constants: info, low, medium, high, critical
export interface SeverityConfig {
  min_report?: string // Minimum severity level to report
}

// Duplicates configuration
export interface DuplicatesConfig {
  suppress_similar?: boolean
  similarity?: number
}

// Output configuration
export interface OutputConfig {
  format?: 'markdown' | 'json'
  style?: OutputStyleConfig
  schema?: OutputSchemaConfig
  channels?: OutputItemConfig[]
}

// Output style configuration
export interface OutputStyleConfig {
  tone?: string
  concise?: boolean
  no_emoji?: boolean
  no_date?: boolean
  language?: string
}

// Output schema configuration
// Note: extra_fields can only be defined at rule level, not in rule_base (no inheritance)
export interface OutputSchemaConfig {
  extra_fields?: ExtraFieldConfig[]
}

// Extra field configuration for extending findings schema
export interface ExtraFieldConfig {
  name: string
  type: 'string' | 'integer' | 'boolean' | 'array'
  description: string
  required?: boolean
  enum?: string[]
}

// Output item configuration
export interface OutputItemConfig {
  type: 'console' | 'file' | 'comment' | 'webhook'
  dir?: string
  overwrite?: boolean
  mode?: 'append' | 'overwrite'
  marker_prefix?: string
  url?: string
  header_secret?: string
  timeout?: number
  max_retries?: number
}

// Rule file info
export interface RuleFile {
  name: string
  path: string
  modified_at: string
}
