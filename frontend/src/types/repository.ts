/**
 * Repository configuration types
 */

// Repository with its review configuration
export interface RepositoryConfigItem {
  id: number
  repo_url: string
  review_file: string
  description?: string
  review_count: number
  last_review_at?: string
  created_at?: string
  updated_at?: string
}

// List repositories response
export interface ListRepositoriesResponse {
  data: RepositoryConfigItem[]
  total: number
  page: number
  page_size: number
}

// Create repository config request
export interface CreateRepositoryConfigRequest {
  repo_url: string
  review_file?: string
  description?: string
}

// Update repository config request
export interface UpdateRepositoryConfigRequest {
  review_file?: string
  description?: string
}


