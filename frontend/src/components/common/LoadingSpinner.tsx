import { Loader2 } from 'lucide-react'
import { cn } from '@/lib/utils'

interface LoadingSpinnerProps {
  size?: 'sm' | 'md' | 'lg'
  className?: string
}

/**
 * Loading spinner component
 */
export function LoadingSpinner({ size = 'md', className }: LoadingSpinnerProps) {
  const sizeClasses = {
    sm: 'h-4 w-4',
    md: 'h-6 w-6',
    lg: 'h-8 w-8',
  }

  return (
    <Loader2
      className={cn(
        'animate-spin text-[hsl(var(--primary))]',
        sizeClasses[size],
        className
      )}
    />
  )
}

interface LoadingPageProps {
  message?: string
}

/**
 * Full page loading component
 */
export function LoadingPage({ message }: LoadingPageProps) {
  return (
    <div className="flex h-screen w-full items-center justify-center">
      <div className="flex flex-col items-center gap-4">
        <LoadingSpinner size="lg" />
        {message && (
          <p className="text-sm text-[hsl(var(--muted-foreground))]">{message}</p>
        )}
      </div>
    </div>
  )
}
