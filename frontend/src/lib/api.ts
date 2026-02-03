/**
 * API client with axios for HTTP requests
 */

import axios, { type AxiosError, type AxiosInstance, type AxiosRequestConfig } from 'axios'
import { getToken, clearToken } from './auth'
import { toast } from '@/hooks/useToast'
import i18n from '@/i18n'
import type { RepositoryConfigItem, CreateRepositoryConfigRequest, UpdateRepositoryConfigRequest } from '@/types/repository'
import type { TaskLogsResponse } from '@/types/api'
import type { FindingsListParams, FindingsListResponse } from '@/types/finding'

// API base URL - uses proxy in development
const API_BASE_URL = '/api/v1'

// Report types
export interface Report {
  id: string
  repo_url: string
  ref: string
  repo_path?: string
  report_type: string
  title?: string
  status: 'pending' | 'analyzing' | 'generating' | 'completed' | 'failed' | 'cancelled'
  structure?: {
    title?: string
    summary?: string
    sections?: Array<{ id: string; title: string; description: string }>
  }
  total_sections: number
  current_section: number
  content?: string
  agent?: string
  started_at?: string
  completed_at?: string
  duration?: number
  error_message?: string
  retry_count: number
  created_at: string
  updated_at: string
  sections?: ReportSection[]
}

export interface ReportSection {
  id: number
  report_id: string
  section_index: number
  section_id: string
  parent_section_id?: string  // parent section ID (for subsections)
  is_leaf: boolean            // true if this section has no subsections
  title: string
  description?: string
  content?: string
  status: 'pending' | 'running' | 'completed' | 'failed' | 'skipped'
  started_at?: string
  completed_at?: string
  duration?: number
  error_message?: string
  retry_count: number
  created_at: string
  updated_at: string
}

export interface ReportProgress {
  report_id: string
  status: string
  total_sections: number
  current_section: number
  sections: Array<{
    section_id: string
    title: string
    status: string
  }>
}

export interface ReportType {
  id: string
  name: string
  description: string
  sections: number
}


// Custom config to skip error toast
interface CustomAxiosRequestConfig extends AxiosRequestConfig {
  skipErrorToast?: boolean
}

// Create axios instance
const apiClient: AxiosInstance = axios.create({
  baseURL: API_BASE_URL,
  timeout: 30000,
  headers: {
    'Content-Type': 'application/json',
  },
})

// Request interceptor to add auth token
apiClient.interceptors.request.use(
  (config) => {
    const token = getToken()
    if (token) {
      config.headers.Authorization = `Bearer ${token}`
    }
    return config
  },
  (error) => {
    return Promise.reject(error)
  }
)

// Get error message from API response
function getErrorMessage(error: AxiosError): string {
  // Try to get error message from response body
  const data = error.response?.data as { error?: string; message?: string } | undefined
  if (data?.error) {
    return data.error
  }
  if (data?.message) {
    return data.message
  }
  
  // Fallback to status-based messages
  const status = error.response?.status
  switch (status) {
    case 400:
      return 'Request failed with status code 400'
    case 401:
      return 'Unauthorized'
    case 403:
      return 'Forbidden'
    case 404:
      return 'Not found'
    case 409:
      return 'Conflict'
    case 500:
      return 'Internal server error'
    default:
      return error.message || 'Unknown error'
  }
}

// Response interceptor to handle errors
apiClient.interceptors.response.use(
  (response) => response,
  (error: AxiosError) => {
    // Skip toast for cancelled requests (user action, not an error)
    if (axios.isCancel(error) || error.name === 'CanceledError') {
      return Promise.reject(error)
    }

    // Check if we should skip error toast for this request
    const config = error.config as CustomAxiosRequestConfig
    const skipErrorToast = config?.skipErrorToast || false

    // Handle 401 Unauthorized - redirect to login
    if (error.response?.status === 401) {
      clearToken()
      // Redirect to login page
      if (window.location.pathname !== '/admin/login') {
        window.location.href = '/admin/login'
      }
    } else if (!skipErrorToast) {
      // Show toast for other errors (unless skipErrorToast is true)
      const message = getErrorMessage(error)
      toast({
        variant: 'destructive',
        title: i18n.t('common.error'),
        description: message,
      })
    }
    return Promise.reject(error)
  }
)

/**
 * Generic API request function
 */
async function request<T>(config: AxiosRequestConfig): Promise<T> {
  const response = await apiClient.request<T>(config)
  return response.data
}

/**
 * GET request
 */
export async function get<T>(url: string, config?: AxiosRequestConfig): Promise<T> {
  return request<T>({ ...config, method: 'GET', url })
}

/**
 * POST request
 */
export async function post<T>(url: string, data?: unknown, config?: AxiosRequestConfig): Promise<T> {
  return request<T>({ ...config, method: 'POST', url, data })
}

/**
 * PUT request
 */
export async function put<T>(url: string, data?: unknown, config?: AxiosRequestConfig): Promise<T> {
  return request<T>({ ...config, method: 'PUT', url, data })
}

/**
 * DELETE request
 */
export async function del<T>(url: string, config?: AxiosRequestConfig): Promise<T> {
  return request<T>({ ...config, method: 'DELETE', url })
}

/**
 * API endpoints
 */
export const api = {
  // Auth
  auth: {
    login: (username: string, password: string, rememberMe: boolean = false) =>
      post<{ token: string; expires_at: string }>('/auth/login', { 
        username, 
        password, 
        remember_me: rememberMe 
      }),
    me: () => get<{ username: string }>('/auth/me'),
    // Password setup (first-time setup, no auth required)
    // Note: This endpoint returns 404 when password is already set (by design for security)
    // We skip error toast for this endpoint since 404 is expected behavior
    checkSetupStatus: () => 
      apiClient.get<{ needs_setup: boolean }>('/auth/setup/status', { 
        skipErrorToast: true 
      } as CustomAxiosRequestConfig).then(res => res.data),
    setupPassword: (password: string, confirmPassword: string) =>
      post<{ message: string }>('/auth/setup', {
        password,
        confirm_password: confirmPassword,
      }),
  },

  // Reviews
  reviews: {
    list: (params?: { page?: number; page_size?: number; status?: string }) =>
      get<{ data: unknown[]; total: number; page: number; page_size: number }>('/reviews', { params }),
    get: (id: string) => get<unknown>(`/reviews/${id}`),
    cancel: (id: string) => post<{ message: string }>(`/reviews/${id}/cancel`),
    retry: (id: string) => post<{ message: string }>(`/reviews/${id}/retry`),
    retryRule: (reviewId: string, ruleId: string) => 
      post<{ message: string }>(`/reviews/${reviewId}/rules/${ruleId}/retry`),
    // Get logs for a specific review
    getLogs: (id: string, params?: { page?: number; page_size?: number; level?: string }) =>
      get<TaskLogsResponse>(`/reviews/${id}/logs`, { params }),
  },

  // Admin - Stats & Status
  admin: {
    stats: () => get<unknown>('/admin/stats'),
    status: () => get<{ version: string; build_time: string; git_commit: string; uptime: number; started_at: string }>('/admin/status'),

    // Statistics
    statistics: {
      repo: (params: { repo_url?: string; time_range: string }) => 
        get<unknown>('/admin/stats/repo', { params }),
    },

    // Rules
    rules: {
      list: () => get<{ files: { name: string; path: string; modified_at: string }[] }>('/admin/rules'),
      get: (name: string) => get<{ content: string; hash: string }>(`/admin/rules/${name}`),
      save: (name: string, content: string, hash: string) => 
        put<{ message: string; hash: string }>(`/admin/rules/${name}`, { content, hash }),
      validate: (content: string) => post<{ valid: boolean; errors?: string[] }>('/admin/rules/validate', { content }),
      // Create new rule file by copying from an existing file (default: default.example.yaml)
      create: (name: string, copyFrom?: string) =>
        post<{ message: string; name: string; hash: string }>('/admin/rules', { name, copy_from: copyFrom }),
    },

    // Review files (list available review config files)
    reviewFiles: {
      list: () => get<{ files: string[] }>('/admin/review-files'),
    },

    // Report types (report configuration files management)
    reportTypes: {
      list: () => get<{ files: { name: string; path: string; modified_at: string }[] }>('/admin/report-types'),
      get: (name: string) => get<{ content: string; hash: string }>(`/admin/report-types/${name}`),
      save: (name: string, content: string, hash: string) => 
        put<{ message: string; hash: string }>(`/admin/report-types/${name}`, { content, hash }),
      validate: (content: string) => post<{ valid: boolean; errors?: string[] }>('/admin/report-types/validate', { content }),
      // Create new report type file by copying from an existing file (default: wiki_simple.yaml)
      create: (name: string, copyFrom?: string) =>
        post<{ message: string; name: string; hash: string }>('/admin/report-types', { name, copy_from: copyFrom }),
    },

    // Repositories (repository review config management)
    repositories: {
      list: (params?: { page?: number; page_size?: number; search?: string; sort_by?: string; sort_order?: 'asc' | 'desc' }) =>
        get<{ data: RepositoryConfigItem[]; total: number; page: number; page_size: number }>('/admin/repositories', { params }),
      create: (data: CreateRepositoryConfigRequest) =>
        post<{ id: number; message: string }>('/admin/repositories', data),
      update: (id: number, data: UpdateRepositoryConfigRequest) =>
        put<{ message: string }>(`/admin/repositories/${id}`, data),
      delete: (id: number) =>
        del<{ message: string }>(`/admin/repositories/${id}`),
    },

    // Findings (aggregated findings list across all repositories)
    findings: {
      list: (params?: FindingsListParams) =>
        get<FindingsListResponse>('/admin/findings', { params }),
    },

    // Settings (database-backed runtime settings)
    settings: {
      // Get all settings grouped by category
      getAll: () => get<{ settings: Record<string, Record<string, unknown>> }>('/admin/settings'),
      // Get settings for a specific category
      getByCategory: (category: string) => 
        get<{ category: string; settings: Record<string, unknown> }>(`/admin/settings/${category}`),
      // Update settings for a specific category
      updateCategory: (category: string, settings: Record<string, unknown>) =>
        put<{ success: boolean; message: string }>(`/admin/settings/${category}`, { settings }),
      // Apply all settings at once
      apply: (settings: Record<string, Record<string, unknown>>) =>
        post<{ success: boolean; message: string }>('/admin/settings/apply', { settings }),
      // Test git provider connection
      testGitProvider: (data: { type: string; url?: string; token: string; insecure_skip_verify?: boolean }) =>
        post<{ success: boolean; message: string }>('/admin/settings/git/test', data),
      // Test agent connection
      testAgent: (data: { 
        name: string; 
        cli_path?: string; 
        api_key?: string; 
        default_model?: string; 
        fallback_models?: string[]; 
        timeout?: number 
      }) =>
        post<{ success: boolean; message: string; data?: string }>('/admin/settings/agents/test', data),
      // Test notification configuration
      testNotification: (data: {
        channel: string;
        webhook_url?: string;
        webhook_secret?: string;
        smtp_host?: string;
        smtp_port?: number;
        smtp_username?: string;
        smtp_password?: string;
        email_from?: string;
        email_to?: string[];
        slack_webhook_url?: string;
        slack_channel?: string;
        feishu_webhook_url?: string;
        feishu_secret?: string;
      }) =>
        post<{ success: boolean; message: string }>('/admin/settings/notifications/test', data),
    },
  },

  // Queue status (public)
  queue: {
    status: () => get<{ total_pending: number; total_running: number; repo_count: number }>('/queue/status'),
  },

  // App metadata (protected - requires auth)
  meta: {
    get: () => get<{ name: string; subtitle: string; version: string }>('/admin/meta'),
  },

  // Reports
  reports: {
    list: (params?: { page?: number; page_size?: number; status?: string; report_type?: string }) =>
      get<{ data: Report[]; total: number; page: number; page_size: number }>('/reports', { params }),
    get: (id: string) => get<Report>(`/reports/${id}`),
    create: (data: { repo_url: string; ref: string; report_type: string; title?: string }) =>
      post<Report>('/reports', data),
    cancel: (id: string) => post<{ message: string }>(`/reports/${id}/cancel`),
    retry: (id: string) => post<{ message: string }>(`/reports/${id}/retry`),
    getProgress: (id: string) => get<ReportProgress>(`/reports/${id}/progress`),
    export: (id: string, format: 'markdown' | 'html' | 'json' | 'pdf' = 'markdown') =>
      get<Blob>(`/reports/${id}/export`, { 
        params: { format },
        responseType: 'blob' as const,
        // PDF generation may take longer due to Chrome rendering
        timeout: format === 'pdf' ? 180000 : 30000
      }),
    // Export report as image (PNG)
    // mode: 'summary' for summary card, 'full' for full report long image
    // theme: 'light' or 'dark' based on current app theme
    exportImage: (id: string, mode: 'summary' | 'full', theme: 'light' | 'dark') =>
      get<Blob>(`/reports/${id}/export`, {
        params: { format: 'image', mode, theme },
        responseType: 'blob' as const,
        // Image generation may take longer due to Chrome rendering
        timeout: 180000
      }),
    repositories: () => get<{ data: string[] }>('/reports/repositories'),
    branches: (repoUrl: string) => get<{ data: string[] }>('/reports/branches', { params: { repo_url: repoUrl } }),
    // Get logs for a specific report
    getLogs: (id: string, params?: { page?: number; page_size?: number; level?: string }) =>
      get<TaskLogsResponse>(`/reports/${id}/logs`, { params }),
  },

  // Report types (public)
  reportTypes: {
    list: () => get<{ data: ReportType[] }>('/report-types'),
  },

  // Schemas (public - for frontend to render data based on schema)
  schemas: {
    get: (name: string) => get<Record<string, unknown>>(`/schemas/${name}`),
  },

}

export default apiClient
