/**
 * Finding related types
 */

// FindingStatus represents the status of a finding compared to previous review
export type FindingStatus = 'fixed' | 'new' | 'persists'

// FindingItem represents a single finding item from the API
export interface FindingItem {
  review_id: string
  repo_url: string
  severity: string
  category: string
  description: string
  created_at: string
  status?: FindingStatus
}

// FindingsListParams represents the query parameters for the findings list API
export interface FindingsListParams {
  page?: number
  page_size?: number
  repo_url?: string
  severity?: string
  category?: string
  sort_by?: 'severity' | 'category'
  sort_order?: 'asc' | 'desc'
}

// FindingsListResponse represents the response from the findings list API
export interface FindingsListResponse {
  data: FindingItem[]
  total: number
  page: number
  page_size: number
}
