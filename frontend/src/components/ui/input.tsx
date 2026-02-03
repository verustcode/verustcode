import * as React from 'react'
import { cn } from '@/lib/utils'

export interface InputProps extends React.InputHTMLAttributes<HTMLInputElement> {}

const Input = React.forwardRef<HTMLInputElement, InputProps>(
  ({ className, type, ...props }, ref) => {
    return (
      <input
        type={type}
        className={cn(
          // Base styles: height, width, border (thin 1px), no shadow, with rounded corners
          'flex h-11 w-full rounded-md border border-[hsl(var(--border))] bg-transparent px-4 py-2 text-sm transition-colors',
          // File input styles
          'file:border-0 file:bg-transparent file:text-sm file:font-medium file:text-[hsl(var(--foreground))]',
          // Placeholder styles
          'placeholder:text-[hsl(var(--placeholder))]',
          // Focus styles: only change border color, no ring/outline to avoid double border
          'focus:outline-none focus:border-[hsl(var(--primary))]',
          // Disabled styles
          'disabled:cursor-not-allowed disabled:opacity-50',
          className
        )}
        ref={ref}
        {...props}
      />
    )
  }
)
Input.displayName = 'Input'

export { Input }
