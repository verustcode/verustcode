import * as React from 'react'
import * as ToastPrimitives from '@radix-ui/react-toast'
import { cva, type VariantProps } from 'class-variance-authority'
import { X } from 'lucide-react'

import { cn } from '@/lib/utils'

const ToastProvider = ToastPrimitives.Provider

const ToastViewport = React.forwardRef<
  React.ElementRef<typeof ToastPrimitives.Viewport>,
  React.ComponentPropsWithoutRef<typeof ToastPrimitives.Viewport>
>(({ className, ...props }, ref) => (
  <ToastPrimitives.Viewport
    ref={ref}
    className={cn(
      'fixed top-0 z-[100] flex max-h-screen w-full flex-col-reverse p-4 sm:bottom-0 sm:right-0 sm:top-auto sm:flex-col md:max-w-[360px]',
      className
    )}
    {...props}
  />
))
ToastViewport.displayName = ToastPrimitives.Viewport.displayName

const toastVariants = cva(
  'group pointer-events-auto relative flex w-full items-center justify-between space-x-3 overflow-hidden rounded-md border p-4 pr-6 shadow-lg backdrop-blur-sm',
  {
    variants: {
      variant: {
        default: 'border-[hsl(var(--border))] bg-[hsl(var(--background))]/95 text-[hsl(var(--foreground))]',
        // Light mode: light tinted background + darker border; Dark mode: subtle tinted background
        success: 'success border-green-200 bg-green-50/95 text-[hsl(var(--foreground))] dark:border-green-800/50 dark:bg-green-950/80',
        warning: 'warning border-amber-200 bg-amber-50/95 text-[hsl(var(--foreground))] dark:border-amber-800/50 dark:bg-amber-950/80',
        error: 'error border-red-200 bg-red-50/95 text-[hsl(var(--foreground))] dark:border-red-800/50 dark:bg-red-950/80',
        info: 'info border-blue-200 bg-blue-50/95 text-[hsl(var(--foreground))] dark:border-blue-800/50 dark:bg-blue-950/80',
        // Keep destructive as alias for error (backward compatibility)
        destructive: 'error border-red-200 bg-red-50/95 text-[hsl(var(--foreground))] dark:border-red-800/50 dark:bg-red-950/80',
      },
    },
    defaultVariants: {
      variant: 'default',
    },
  }
)

const Toast = React.forwardRef<
  React.ElementRef<typeof ToastPrimitives.Root>,
  React.ComponentPropsWithoutRef<typeof ToastPrimitives.Root> &
    VariantProps<typeof toastVariants>
>(({ className, variant, ...props }, ref) => {
  return (
    <ToastPrimitives.Root
      ref={ref}
      className={cn(toastVariants({ variant }), className)}
      {...props}
    />
  )
})
Toast.displayName = ToastPrimitives.Root.displayName

const ToastAction = React.forwardRef<
  React.ElementRef<typeof ToastPrimitives.Action>,
  React.ComponentPropsWithoutRef<typeof ToastPrimitives.Action>
>(({ className, ...props }, ref) => (
  <ToastPrimitives.Action
    ref={ref}
    className={cn(
      'inline-flex h-8 shrink-0 items-center justify-center rounded-md border bg-transparent px-3 text-sm font-medium ring-offset-[hsl(var(--background))] transition-colors hover:bg-[hsl(var(--secondary))] focus:outline-none focus:ring-2 focus:ring-[hsl(var(--ring))] focus:ring-offset-2 disabled:pointer-events-none disabled:opacity-50 group-[.destructive]:border-[hsl(var(--muted))]/40 group-[.destructive]:hover:border-[hsl(var(--destructive))]/30 group-[.destructive]:hover:bg-[hsl(var(--destructive))] group-[.destructive]:hover:text-[hsl(var(--destructive-foreground))] group-[.destructive]:focus:ring-[hsl(var(--destructive))]',
      className
    )}
    {...props}
  />
))
ToastAction.displayName = ToastPrimitives.Action.displayName

const ToastClose = React.forwardRef<
  React.ElementRef<typeof ToastPrimitives.Close>,
  React.ComponentPropsWithoutRef<typeof ToastPrimitives.Close>
>(({ className, ...props }, ref) => (
  <ToastPrimitives.Close
    ref={ref}
    className={cn(
      'absolute right-2 top-2 rounded-md p-1 text-[hsl(var(--foreground))]/50 opacity-0 transition-opacity hover:text-[hsl(var(--foreground))] focus:opacity-100 focus:outline-none focus:ring-2 group-hover:opacity-100',
      // Variant-specific close button styles - use muted colors that work with tinted backgrounds
      'group-[.error]:text-red-400 group-[.error]:hover:text-red-600 dark:group-[.error]:text-red-400 dark:group-[.error]:hover:text-red-300',
      'group-[.success]:text-green-400 group-[.success]:hover:text-green-600 dark:group-[.success]:text-green-400 dark:group-[.success]:hover:text-green-300',
      'group-[.warning]:text-amber-400 group-[.warning]:hover:text-amber-600 dark:group-[.warning]:text-amber-400 dark:group-[.warning]:hover:text-amber-300',
      'group-[.info]:text-blue-400 group-[.info]:hover:text-blue-600 dark:group-[.info]:text-blue-400 dark:group-[.info]:hover:text-blue-300',
      className
    )}
    toast-close=""
    {...props}
  >
    <X className="h-4 w-4" />
  </ToastPrimitives.Close>
))
ToastClose.displayName = ToastPrimitives.Close.displayName

const ToastTitle = React.forwardRef<
  React.ElementRef<typeof ToastPrimitives.Title>,
  React.ComponentPropsWithoutRef<typeof ToastPrimitives.Title>
>(({ className, ...props }, ref) => (
  <ToastPrimitives.Title
    ref={ref}
    className={cn('text-xs font-semibold', className)}
    {...props}
  />
))
ToastTitle.displayName = ToastPrimitives.Title.displayName

const ToastDescription = React.forwardRef<
  React.ElementRef<typeof ToastPrimitives.Description>,
  React.ComponentPropsWithoutRef<typeof ToastPrimitives.Description>
>(({ className, ...props }, ref) => (
  <ToastPrimitives.Description
    ref={ref}
    className={cn('text-xs opacity-90', className)}
    {...props}
  />
))
ToastDescription.displayName = ToastPrimitives.Description.displayName

type ToastProps = React.ComponentPropsWithoutRef<typeof Toast>

type ToastActionElement = React.ReactElement<typeof ToastAction>

export {
  type ToastProps,
  type ToastActionElement,
  ToastProvider,
  ToastViewport,
  Toast,
  ToastTitle,
  ToastDescription,
  ToastClose,
  ToastAction,
}

