/**
 * API response types
 */

// Generic API response wrapper
export interface ApiResponse<T> {
  data: T
  message?: string
}

// Paginated response
export interface PaginatedResponse<T> {
  data: T[]
  total: number
  page: number
  page_size: number
}

// Error response
export interface ApiError {
  code: string
  message: string
}

// Auth types
export interface LoginRequest {
  username: string
  password: string
}

export interface LoginResponse {
  token: string
  expires_at: string
}

export interface UserInfo {
  username: string
}

// Stats for dashboard
export interface DashboardStats {
  today_reviews: number
  running_reviews: number
  total_reviews: number
  success_rate: number
  avg_duration: number
  pending_count: number
  completed_today: number
  failed_today: number
  repository_count: number  // Total number of repositories
  total_reports: number     // Total number of reports
}

// Queue status
export interface QueueStats {
  total_pending: number
  total_running: number
  repo_count: number
  repos: Record<string, RepoStats>
}

export interface RepoStats {
  pending: number
  running: number
}

// Server status
export interface ServerStatus {
  version: string       // Application version
  build_time: string    // Build timestamp
  git_commit: string    // Git commit hash
  uptime: number        // Uptime in seconds
  started_at: string    // Server start time in RFC3339 format
  go_version: string    // Go runtime version
  memory_usage: number  // Memory usage in bytes (heap alloc)
}

// Task Log types
export type LogLevel = 'debug' | 'info' | 'warn' | 'error' | 'fatal'
export type TaskType = 'review' | 'report'

export interface TaskLog {
  id: number
  created_at: string
  task_type: TaskType
  task_id: string
  level: LogLevel
  logger?: string       // Name of the logger (e.g., engine, api)
  message: string       // Log message (matches backend JSON field name)
  fields?: Record<string, unknown>  // Additional structured fields
  caller?: string       // File path and line number (e.g., report/summary.go:160)
}

// Response from /reviews/:id/logs and /reports/:id/logs endpoints
export interface TaskLogsResponse {
  data: TaskLog[]
  total: number
  page: number
  page_size: number
}
