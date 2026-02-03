/**
 * Authentication utilities for JWT token management
 */

const TOKEN_KEY = 'scopeview_token'
const TOKEN_EXPIRY_KEY = 'scopeview_token_expiry'

/**
 * Save auth token to localStorage
 */
export function saveToken(token: string, expiresAt: string): void {
  localStorage.setItem(TOKEN_KEY, token)
  localStorage.setItem(TOKEN_EXPIRY_KEY, expiresAt)
  console.log('[Auth] Token saved to localStorage, expires at:', expiresAt)
}

/**
 * Get auth token from localStorage
 */
export function getToken(): string | null {
  const token = localStorage.getItem(TOKEN_KEY)
  const expiry = localStorage.getItem(TOKEN_EXPIRY_KEY)

  if (!token || !expiry) {
    console.log('[Auth] No token or expiry found in localStorage')
    return null
  }

  // Check if token is expired
  const expiryDate = new Date(expiry)
  const now = new Date()
  
  if (expiryDate <= now) {
    console.log('[Auth] Token expired, expiry:', expiry, 'now:', now.toISOString())
    clearToken()
    return null
  }

  console.log('[Auth] Token valid, expires at:', expiry)
  return token
}

/**
 * Clear auth token from localStorage
 */
export function clearToken(): void {
  localStorage.removeItem(TOKEN_KEY)
  localStorage.removeItem(TOKEN_EXPIRY_KEY)
  console.log('[Auth] Token cleared from localStorage')
}

/**
 * Check if user is authenticated
 */
export function isAuthenticated(): boolean {
  return getToken() !== null
}

/**
 * Get token expiry time
 */
export function getTokenExpiry(): Date | null {
  const expiry = localStorage.getItem(TOKEN_EXPIRY_KEY)
  if (!expiry) return null
  return new Date(expiry)
}

/**
 * Check if token is about to expire (within 5 minutes)
 */
export function isTokenExpiringSoon(): boolean {
  const expiry = getTokenExpiry()
  if (!expiry) return true

  const fiveMinutesFromNow = new Date(Date.now() + 5 * 60 * 1000)
  return expiry <= fiveMinutesFromNow
}
