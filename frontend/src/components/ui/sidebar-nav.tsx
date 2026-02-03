import * as React from 'react'
import { NavLink } from 'react-router-dom'
import { cva, type VariantProps } from 'class-variance-authority'
import { PanelLeftClose } from 'lucide-react'
import { cn } from '@/lib/utils'
import { Button } from '@/components/ui/button'
import { Tooltip, TooltipContent, TooltipTrigger } from '@/components/ui/tooltip'
import { Separator } from '@/components/ui/separator'
import { VerustCodeLogo } from '@/components/common/VerustCodeLogo'

/**
 * Sidebar navigation component
 * Encapsulated based on shadcn/ui style, supports expand/collapse, active highlighting, and Tooltip hints
 */

// Navigation item style variants
const navItemVariants = cva(
  'flex items-center gap-3 whitespace-nowrap rounded-md text-sm font-medium transition-colors',
  {
      variants: {
        active: {
          // Medium priority: light background + primary text + left border, supports dark/light mode
          true: 'bg-[hsl(var(--accent))] text-[hsl(var(--primary))] border-l-2 border-l-[hsl(var(--primary))]',
          false: 'text-[hsl(var(--foreground))] hover:bg-[hsl(var(--accent))] hover:text-[hsl(var(--accent-foreground))]',
        },
        collapsed: {
          // Collapsed state: center display, use square size to avoid crowding
          true: 'h-10 w-10 justify-center px-0',
          // Expanded state: regular layout
          false: 'h-10 px-3',
        },
    },
    defaultVariants: {
      active: false,
      collapsed: false,
    },
  }
)

export interface NavItem {
  path: string
  icon: React.ComponentType<{ className?: string }>
  label: string
  end?: boolean
  separator?: boolean  // Whether to show a separator after this item
}

interface SidebarNavProps {
  items: NavItem[]
  collapsed?: boolean
  className?: string
}

/**
 * SidebarNav - Sidebar navigation list
 */
export function SidebarNav({ items, collapsed = false, className }: SidebarNavProps) {
  return (
    <nav
      className={cn(
        'flex flex-col',
        // Collapsed state: increase spacing and center align, expanded state uses smaller spacing
        collapsed ? 'items-center gap-5' : 'gap-1',
        className
      )}
    >
      {items.map((item) => (
        <React.Fragment key={item.path}>
          <SidebarNavItem item={item} collapsed={collapsed} />
          {/* Render separator after this item if specified */}
          {item.separator && (
            <div className={cn(
              'my-2',
              // Collapsed state: centered with auto margins
              // Expanded state: add padding on both sides to create gaps
              collapsed ? 'flex justify-center' : 'px-6'
            )}>
              <Separator className={collapsed ? 'w-8' : ''} />
            </div>
          )}
        </React.Fragment>
      ))}
    </nav>
  )
}

interface SidebarNavItemProps {
  item: NavItem
  collapsed?: boolean
}

/**
 * SidebarNavItem - Single navigation item
 */
function SidebarNavItem({ item, collapsed = false }: SidebarNavItemProps) {
  const Icon = item.icon

  const linkContent = (
    <NavLink
      to={item.path}
      end={item.end}
      className={({ isActive }) => navItemVariants({ active: isActive, collapsed })}
    >
      <Icon className="h-5 w-5 shrink-0" />
      {!collapsed && <span>{item.label}</span>}
    </NavLink>
  )

  // Show Tooltip in collapsed state
  if (collapsed) {
    return (
      <Tooltip delayDuration={0}>
        <TooltipTrigger asChild>{linkContent}</TooltipTrigger>
        <TooltipContent side="right" sideOffset={10}>
          {item.label}
        </TooltipContent>
      </Tooltip>
    )
  }

  return linkContent
}

// Sidebar container styles
const sidebarVariants = cva(
  'flex h-full flex-col border-r border-[hsl(var(--border))] bg-[hsl(var(--card))] transition-all duration-300',
  {
    variants: {
      collapsed: {
        true: 'w-16',
        false: 'w-64',
      },
    },
    defaultVariants: {
      collapsed: false,
    },
  }
)

interface SidebarContainerProps extends VariantProps<typeof sidebarVariants> {
  children: React.ReactNode
  className?: string
}

/**
 * SidebarContainer - Sidebar container
 */
export function SidebarContainer({ collapsed, children, className }: SidebarContainerProps) {
  return <aside className={cn(sidebarVariants({ collapsed }), className)}>{children}</aside>
}

interface SidebarHeaderProps {
  collapsed?: boolean
  title: string
  subtitle?: string  // Optional subtitle (smaller, gray text below title)
  onToggle?: () => void  // Collapse/expand callback
}

/**
 * SidebarHeader - Sidebar header (Logo area + collapse button)
 * 
 * Interaction notes:
 * - Expanded state: shows Logo + title + subtitle + collapse button
 * - Collapsed state: only shows Logo (clickable to expand), hides title, subtitle and collapse button
 * - Logo size and vertical position remain consistent in both states
 */
export function SidebarHeader({ collapsed, title, subtitle, onToggle }: SidebarHeaderProps) {
  // Logo component - clickable to expand in collapsed state (no Tooltip)
  const logoElement = collapsed && onToggle ? (
    <button
      onClick={onToggle}
      className="flex items-center justify-center rounded-md hover:bg-[hsl(var(--accent))] p-1 transition-colors"
      aria-label="Expand sidebar"
    >
      <VerustCodeLogo size="sm" />
    </button>
  ) : (
    <VerustCodeLogo size="sm" />
  )

  return (
    <div className={cn(
      'flex h-16 bg-[hsl(var(--muted)/0.5)]',
      // Center Logo when collapsed, left align when expanded
      collapsed ? 'items-center justify-center px-2' : 'items-start px-3 pt-3'
    )}>
      {/* Logo - always displayed, size and position unchanged */}
      <div className="flex items-center shrink-0">
        {logoElement}
      </div>

      {/* Title and subtitle - only shown when expanded */}
      {!collapsed && (
        <div className="ml-2 flex flex-col min-w-0">
          <span className="text-lg font-bold text-[hsl(var(--primary))] leading-tight">{title}</span>
          {subtitle && (
            <span className="text-sm text-[hsl(var(--muted-foreground))] truncate leading-tight">{subtitle}</span>
          )}
        </div>
      )}

      {/* Flexible space - only when expanded */}
      {!collapsed && <div className="flex-1" />}

      {/* Collapse button - only shown when expanded (no Tooltip) */}
      {!collapsed && onToggle && (
        <Button
          variant="ghost"
          size="icon"
          onClick={onToggle}
          className="h-8 w-8 shrink-0"
          aria-label="Collapse sidebar"
        >
          <PanelLeftClose className="h-4 w-4" />
        </Button>
      )}
    </div>
  )
}









