import { useRouteError, isRouteErrorResponse, useNavigate } from 'react-router-dom'
import { AlertTriangle, Home, RefreshCw, FileQuestion } from 'lucide-react'
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'
import { Button } from '@/components/ui/button'

/**
 * Error page component for React Router error boundary
 * Displays user-friendly error messages with actions to recover
 */
export function ErrorPage() {
  const error = useRouteError()
  const navigate = useNavigate()

  // Determine if this is a 404 error
  const is404 = isRouteErrorResponse(error) && error.status === 404

  // Get error details
  const getErrorDetails = () => {
    if (isRouteErrorResponse(error)) {
      return {
        title: `${error.status} ${error.statusText}`,
        message: error.data?.message || getDefaultMessage(error.status),
      }
    }
    if (error instanceof Error) {
      return {
        title: 'Unexpected Error',
        message: error.message,
      }
    }
    return {
      title: 'Unknown Error',
      message: 'An unexpected error occurred.',
    }
  }

  const getDefaultMessage = (status: number) => {
    switch (status) {
      case 404:
        return 'The page you are looking for does not exist.'
      case 403:
        return 'You do not have permission to access this resource.'
      case 500:
        return 'Internal server error. Please try again later.'
      default:
        return 'Something went wrong.'
    }
  }

  const { title, message } = getErrorDetails()

  const handleGoHome = () => {
    navigate('/admin')
  }

  const handleRefresh = () => {
    window.location.reload()
  }

  return (
    <div className="flex min-h-screen items-center justify-center bg-[hsl(var(--background))] p-4">
      <Card className="w-full max-w-md">
        <CardHeader className="text-center">
          <div className="mx-auto mb-4 flex h-16 w-16 items-center justify-center rounded-full bg-[hsl(var(--destructive))]/10">
            {is404 ? (
              <FileQuestion className="h-8 w-8 text-[hsl(var(--muted-foreground))]" />
            ) : (
              <AlertTriangle className="h-8 w-8 text-[hsl(var(--destructive))]" />
            )}
          </div>
          <CardTitle className="text-xl">{title}</CardTitle>
        </CardHeader>
        <CardContent className="space-y-6">
          <p className="text-center text-[hsl(var(--muted-foreground))]">
            {message}
          </p>

          {/* Show stack trace in development */}
          {import.meta.env.DEV && error instanceof Error && error.stack && (
            <div className="rounded-md bg-[hsl(var(--muted))]/50 p-3">
              <p className="mb-2 text-xs font-medium text-[hsl(var(--muted-foreground))]">
                Stack Trace (dev only)
              </p>
              <pre className="overflow-auto text-xs text-[hsl(var(--muted-foreground))]">
                {error.stack}
              </pre>
            </div>
          )}

          <div className="flex flex-col gap-3 sm:flex-row">
            <Button
              variant="outline"
              className="flex-1"
              onClick={handleRefresh}
            >
              <RefreshCw className="mr-2 h-4 w-4" />
              Refresh
            </Button>
            <Button className="flex-1" onClick={handleGoHome}>
              <Home className="mr-2 h-4 w-4" />
              Go to Dashboard
            </Button>
          </div>
        </CardContent>
      </Card>
    </div>
  )
}

/**
 * 404 Not Found page component
 * Used for unmatched routes
 */
export function NotFoundPage() {
  const navigate = useNavigate()

  const handleGoHome = () => {
    navigate('/admin')
  }

  return (
    <div className="flex min-h-screen items-center justify-center bg-[hsl(var(--background))] p-4">
      <Card className="w-full max-w-md">
        <CardHeader className="text-center">
          <div className="mx-auto mb-4 flex h-16 w-16 items-center justify-center rounded-full bg-[hsl(var(--muted))]/50">
            <FileQuestion className="h-8 w-8 text-[hsl(var(--muted-foreground))]" />
          </div>
          <CardTitle className="text-xl">404 Not Found</CardTitle>
        </CardHeader>
        <CardContent className="space-y-6">
          <p className="text-center text-[hsl(var(--muted-foreground))]">
            The page you are looking for does not exist.
          </p>

          <Button className="w-full" onClick={handleGoHome}>
            <Home className="mr-2 h-4 w-4" />
            Go to Dashboard
          </Button>
        </CardContent>
      </Card>
    </div>
  )
}
