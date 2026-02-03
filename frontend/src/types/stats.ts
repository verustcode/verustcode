/**
 * Statistics related types
 */

// Weekly delivery time statistics
export interface WeeklyDeliveryTime {
  week: string // ISO week format: "2024-W01"
  p50: number  // 50th percentile (median) in hours
  p90: number  // 90th percentile in hours
  p95: number  // 95th percentile in hours
}

// Weekly code change statistics
export interface WeeklyCodeChange {
  week: string          // ISO week format: "2024-W01"
  lines_added: number   // total lines added
  lines_deleted: number // total lines deleted
  net_change: number    // net change (added - deleted)
}

// Weekly file change statistics
export interface WeeklyFileChange {
  week: string          // ISO week format: "2024-W01"
  files_changed: number // total files changed
}

// Weekly MR count statistics
export interface WeeklyMRCount {
  week: string  // ISO week format: "2024-W01"
  count: number // number of MRs
}

// Weekly revision statistics
export interface WeeklyRevision {
  week: string            // ISO week format: "2024-W01"
  avg_revisions: number   // average number of revisions per MR
  total_revisions: number // total revisions
  mr_count: number        // number of MRs in this week

  // Layer statistics (generic design for revision count distribution)
  layer_1_count: number   // count of MRs in layer 1 (e.g., 1-2 revisions)
  layer_2_count: number   // count of MRs in layer 2 (e.g., 3-4 revisions)
  layer_3_count: number   // count of MRs in layer 3 (e.g., 5+ revisions)

  // Layer labels for rendering (provided by backend)
  layer_1_label: string   // label for layer 1 (e.g., "1-2次")
  layer_2_label: string   // label for layer 2 (e.g., "3-4次")
  layer_3_label: string   // label for layer 3 (e.g., "5+次")
}

// Issue severity statistics
export interface IssueSeverityStats {
  severity: string // critical, high, medium, low, info
  count: number    // number of issues with this severity
}

// Issue category statistics
export interface IssueCategoryStats {
  category: string // e.g., security, performance, code-quality
  count: number    // number of issues in this category
}

// Repository statistics aggregation
export interface RepoStats {
  delivery_time_stats: WeeklyDeliveryTime[]
  code_change_stats: WeeklyCodeChange[]
  file_change_stats: WeeklyFileChange[]
  mr_count_stats: WeeklyMRCount[]
  revision_stats: WeeklyRevision[]
  issue_severity_stats: IssueSeverityStats[]   // issues grouped by severity
  issue_category_stats: IssueCategoryStats[]   // issues grouped by category
}

// Time range options
export type TimeRange = '3m' | '6m' | '1y'

// Repository option for selector
export interface RepoOption {
  label: string
  value: string
}

