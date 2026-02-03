import { useState } from 'react'
import { Outlet, Navigate } from 'react-router-dom'
import { TooltipProvider } from '@/components/ui/tooltip'
import { Sidebar } from './Sidebar'
import { Header } from './Header'
import { LoadingPage } from '@/components/common/LoadingSpinner'
import { useAuth } from '@/hooks/useAuth'

const SIDEBAR_COLLAPSED_KEY = 'scopeview_sidebar_collapsed'

/**
 * Main application layout with sidebar and header
 */
export function AppLayout() {
  const { user, loading, logout, isAuthenticated } = useAuth()
  const [sidebarCollapsed, setSidebarCollapsed] = useState(() => {
    const saved = localStorage.getItem(SIDEBAR_COLLAPSED_KEY)
    return saved === 'true'
  })

  const handleSidebarToggle = () => {
    const newState = !sidebarCollapsed
    setSidebarCollapsed(newState)
    localStorage.setItem(SIDEBAR_COLLAPSED_KEY, String(newState))
  }

  // Show loading while checking auth
  if (loading) {
    return <LoadingPage />
  }

  // Redirect to login if not authenticated
  if (!isAuthenticated) {
    return <Navigate to="/admin/login" replace />
  }

  return (
    <TooltipProvider>
      <div className="flex h-screen overflow-hidden bg-[hsl(var(--background))]">
        {/* Sidebar */}
        <Sidebar collapsed={sidebarCollapsed} onToggle={handleSidebarToggle} />

        {/* Main content area */}
        <div className="flex flex-1 flex-col overflow-hidden">
          {/* Header */}
          <Header username={user?.username} onLogout={logout} />

          {/* Page content */}
          <main className="flex-1 overflow-auto p-5">
            <Outlet />
          </main>
        </div>
      </div>
    </TooltipProvider>
  )
}
