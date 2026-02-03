/**
 * Custom hook to fetch and cache application metadata
 */

import { useQuery } from '@tanstack/react-query'
import { useEffect } from 'react'
import { api } from '@/lib/api'
import { useAuth } from '@/hooks/useAuth'

export interface AppMeta {
  name: string
  subtitle: string
  version: string
}

/**
 * Hook to fetch application metadata (name, subtitle, version)
 * Data is cached and only fetched once after authentication
 * Requires authentication to access the meta endpoint
 */
export function useAppMeta() {
  const { isAuthenticated } = useAuth()

  const { data, isLoading, error } = useQuery<AppMeta>({
    queryKey: ['app', 'meta'],
    queryFn: () => api.meta.get(),
    staleTime: Infinity, // Never consider stale - only fetch once
    gcTime: Infinity, // Keep in cache forever
    refetchOnWindowFocus: false,
    refetchOnMount: false,
    enabled: isAuthenticated, // Only fetch when authenticated
  })

  // Update document title when metadata is loaded
  useEffect(() => {
    if (data) {
      const title = data.subtitle 
        ? `${data.name} - ${data.subtitle}`
        : data.name
      document.title = title
    }
  }, [data])

  return {
    appMeta: data,
    isLoading,
    error,
    subtitle: data?.subtitle || '',
    appName: data?.name || 'VerustCode',
    version: data?.version || '',
  }
}

