import { clsx, type ClassValue } from 'clsx'
import { twMerge } from 'tailwind-merge'

/**
 * Merge class names with Tailwind CSS classes
 */
export function cn(...inputs: ClassValue[]) {
  return twMerge(clsx(inputs))
}

/**
 * Format duration in milliseconds to human readable string
 */
export function formatDuration(ms: number): string {
  if (ms < 1000) {
    return `${ms}ms`
  }
  const seconds = Math.floor(ms / 1000)
  if (seconds < 60) {
    return `${seconds}s`
  }
  const minutes = Math.floor(seconds / 60)
  const remainingSeconds = seconds % 60
  if (minutes < 60) {
    return remainingSeconds > 0 ? `${minutes}m ${remainingSeconds}s` : `${minutes}m`
  }
  const hours = Math.floor(minutes / 60)
  const remainingMinutes = minutes % 60
  return remainingMinutes > 0 ? `${hours}h ${remainingMinutes}m` : `${hours}h`
}

/**
 * Format date to locale string
 */
export function formatDate(date: string | Date): string {
  const d = typeof date === 'string' ? new Date(date) : date
  return d.toLocaleString()
}

/**
 * Format relative time (e.g., "2 minutes ago")
 * @param date - The date to format
 * @param t - Optional translation function for i18n support
 */
export function formatRelativeTime(
  date: string | Date,
  t?: (key: string, options?: { count?: number }) => string
): string {
  const d = typeof date === 'string' ? new Date(date) : date
  const now = new Date()
  const diff = now.getTime() - d.getTime()

  const seconds = Math.floor(diff / 1000)
  if (seconds < 60) {
    return t ? t('common.time.justNow') : 'just now'
  }

  const minutes = Math.floor(seconds / 60)
  if (minutes < 60) {
    return t ? t('common.time.minutesAgo', { count: minutes }) : `${minutes}m ago`
  }

  const hours = Math.floor(minutes / 60)
  if (hours < 24) {
    return t ? t('common.time.hoursAgo', { count: hours }) : `${hours}h ago`
  }

  const days = Math.floor(hours / 24)
  if (days < 7) {
    return t ? t('common.time.daysAgo', { count: days }) : `${days}d ago`
  }

  return formatDate(d)
}

/**
 * Debounce function
 */
export function debounce<T extends (...args: unknown[]) => unknown>(
  func: T,
  wait: number
): (...args: Parameters<T>) => void {
  let timeout: ReturnType<typeof setTimeout> | null = null

  return (...args: Parameters<T>) => {
    if (timeout) clearTimeout(timeout)
    timeout = setTimeout(() => func(...args), wait)
  }
}

/**
 * Sleep for specified milliseconds
 */
export function sleep(ms: number): Promise<void> {
  return new Promise(resolve => setTimeout(resolve, ms))
}

/**
 * Truncate string with ellipsis
 */
export function truncate(str: string, maxLength: number): string {
  if (str.length <= maxLength) return str
  return str.slice(0, maxLength - 3) + '...'
}

/**
 * Format repository URL for display (remove protocol prefix)
 * @param url - The repository URL to format
 * @returns Formatted URL without protocol prefix
 */
export function formatRepoUrl(url: string): string {
  return url.replace(/^https?:\/\//, '').replace(/\.git$/, '')
}

/**
 * Parse repository URL to get owner and repo
 */
export function parseRepoUrl(url: string): { owner: string; repo: string } | null {
  const match = url.match(/(?:github\.com|gitlab\.com)[/:]([^/]+)\/([^/.]+)/)
  if (match) {
    return { owner: match[1], repo: match[2] }
  }
  return null
}
