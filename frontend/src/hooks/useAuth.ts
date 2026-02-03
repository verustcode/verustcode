import { useState, useCallback, useEffect } from 'react'
import { useNavigate } from 'react-router-dom'
import { api } from '@/lib/api'
import { saveToken, clearToken, isAuthenticated, getToken } from '@/lib/auth'

interface User {
  username: string
}

/**
 * Hook for authentication state and actions
 */
export function useAuth() {
  const [user, setUser] = useState<User | null>(null)
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState<string | null>(null)
  const [authState, setAuthState] = useState<boolean>(() => {
    // Check if token exists and is valid in localStorage on initialization
    return isAuthenticated()
  })
  const navigate = useNavigate()

  // Check authentication status on mount
  useEffect(() => {
    const checkAuth = async () => {
      // Check token in localStorage
      const token = getToken()
      if (!token) {
        console.log('[Auth] No token found in localStorage')
        setAuthState(false)
        setLoading(false)
        return
      }

      console.log('[Auth] Token found, validating with server...')
      try {
        // Validate token with server
        const userData = await api.auth.me()
        setUser(userData)
        setAuthState(true)
        console.log('[Auth] Token validated successfully, user:', userData.username)
      } catch (err) {
        // Token is invalid, clear it
        console.warn('[Auth] Token validation failed, clearing token:', err)
        clearToken()
        setAuthState(false)
        setUser(null)
      } finally {
        setLoading(false)
      }
    }

    checkAuth()
  }, [])

  // Login function
  const login = useCallback(async (username: string, password: string, rememberMe: boolean = false) => {
    setError(null)
    setLoading(true)

    try {
      console.log('[Auth] Attempting login, rememberMe:', rememberMe)
      const response = await api.auth.login(username, password, rememberMe)
      
      // Save token to localStorage (includes expiration time)
      saveToken(response.token, response.expires_at)
      console.log('[Auth] Token saved, expires at:', response.expires_at)
      
      // Update authentication state
      setAuthState(true)
      
      // Fetch user info
      const userData = await api.auth.me()
      setUser(userData)
      
      navigate('/admin')
    } catch (err) {
      const message = err instanceof Error ? err.message : 'Login failed'
      setError(message)
      console.error('[Auth] Login failed:', err)
      throw err
    } finally {
      setLoading(false)
    }
  }, [navigate])

  // Logout function
  const logout = useCallback(() => {
    console.log('[Auth] Logging out...')
    clearToken()
    setUser(null)
    setAuthState(false)
    navigate('/admin/login')
  }, [navigate])

  return {
    user,
    loading,
    error,
    login,
    logout,
    isAuthenticated: authState,
  }
}
