/**
 * Review related types
 */

// Review status enum
export type ReviewStatus = 'pending' | 'running' | 'completed' | 'failed' | 'cancelled'

// Rule status enum
export type RuleStatus = 'pending' | 'running' | 'completed' | 'failed' | 'skipped'

// Run status enum
export type RunStatus = 'pending' | 'running' | 'completed' | 'failed'

// Review model
export interface Review {
  id: string // UUID
  created_at: string
  updated_at: string
  ref: string
  commit_sha: string
  pr_number?: number
  pr_url?: string
  repo_url: string
  repo_path?: string
  source: string
  triggered_by?: string
  status: ReviewStatus
  current_rule_index: number
  retry_count: number
  started_at?: string
  completed_at?: string
  duration?: number
  // PR/Branch metadata
  branch_created_at?: string
  author?: string
  // Diff statistics
  lines_added?: number
  lines_deleted?: number
  files_changed?: number
  error_message?: string
  rules?: ReviewRule[]
}

// Review rule model
export interface ReviewRule {
  id: number
  created_at: string
  updated_at: string
  review_id: string // UUID reference
  rule_index: number
  rule_id: string
  rule_config?: Record<string, unknown>
  status: RuleStatus
  multi_run_enabled: boolean
  multi_run_runs: number
  current_run_index: number
  findings_count: number
  prompt?: string // Rendered prompt text (markdown format)
  started_at?: string
  completed_at?: string
  duration?: number
  error_message?: string
  retry_count: number // number of retry attempts for this rule
  runs?: ReviewRuleRun[]
  results?: ReviewResult[]
}

// Review rule run model
export interface ReviewRuleRun {
  id: number
  created_at: string
  updated_at: string
  review_rule_id: number
  run_index: number
  model?: string
  agent: string
  status: RunStatus
  findings_count: number
  started_at?: string
  completed_at?: string
  duration?: number
  error_message?: string
}

// Review result model
export interface ReviewResult {
  id: number
  created_at: string
  updated_at: string
  review_rule_id: number
  data: Record<string, unknown>
}

// Create review request
export interface CreateReviewRequest {
  repository: string
  ref: string
  provider?: string
  agent?: string
  pr_number?: number
}
