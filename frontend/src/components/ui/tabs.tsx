import * as React from 'react'
import * as TabsPrimitive from '@radix-ui/react-tabs'
import { cn } from '@/lib/utils'

const Tabs = TabsPrimitive.Root

/**
 * TabsList - Container for tab triggers
 * Uses bottom border separator for a cleaner look
 */
const TabsList = React.forwardRef<
  React.ElementRef<typeof TabsPrimitive.List>,
  React.ComponentPropsWithoutRef<typeof TabsPrimitive.List>
>(({ className, ...props }, ref) => (
  <TabsPrimitive.List
    ref={ref}
    className={cn(
      'inline-flex h-9 items-center gap-1 border-b border-[hsl(var(--border))] text-[hsl(var(--muted-foreground))]',
      className
    )}
    {...props}
  />
))
TabsList.displayName = TabsPrimitive.List.displayName

/**
 * TabsTrigger - Individual tab button
 * Active state: bottom highlight border + foreground text color (no background)
 * Inactive state: transparent background + muted text color
 */
const TabsTrigger = React.forwardRef<
  React.ElementRef<typeof TabsPrimitive.Trigger>,
  React.ComponentPropsWithoutRef<typeof TabsPrimitive.Trigger>
>(({ className, ...props }, ref) => (
  <TabsPrimitive.Trigger
    ref={ref}
    className={cn(
      // Base styles: layout and typography
      'relative inline-flex items-center justify-center whitespace-nowrap px-4 py-2 text-sm font-medium',
      // Transition for smooth color changes
      'transition-colors duration-200',
      // Focus visible styles
      'ring-offset-[hsl(var(--background))] focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-[hsl(var(--ring))] focus-visible:ring-offset-2',
      // Disabled state
      'disabled:pointer-events-none disabled:opacity-50',
      // Hover state (when not active)
      'hover:text-[hsl(var(--foreground))]',
      // Active state: foreground text color + bottom highlight border
      'data-[state=active]:text-[hsl(var(--primary))]',
      // Bottom highlight border using pseudo-element
      'after:absolute after:bottom-0 after:left-0 after:right-0 after:h-0.5 after:transition-colors after:duration-200',
      'data-[state=active]:after:bg-[hsl(var(--primary))]',
      className
    )}
    {...props}
  />
))
TabsTrigger.displayName = TabsPrimitive.Trigger.displayName

const TabsContent = React.forwardRef<
  React.ElementRef<typeof TabsPrimitive.Content>,
  React.ComponentPropsWithoutRef<typeof TabsPrimitive.Content>
>(({ className, ...props }, ref) => (
  <TabsPrimitive.Content
    ref={ref}
    className={cn(
      'mt-2 ring-offset-[hsl(var(--background))] focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-[hsl(var(--ring))] focus-visible:ring-offset-2',
      className
    )}
    {...props}
  />
))
TabsContent.displayName = TabsPrimitive.Content.displayName

export { Tabs, TabsList, TabsTrigger, TabsContent }
