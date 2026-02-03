import { useState, useRef, useEffect } from 'react'
import { useTranslation } from 'react-i18next'
import { LogOut, User, Loader2, Download, X } from 'lucide-react'
import { Button } from '@/components/ui/button'
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from '@/components/ui/dialog'
import { cn } from '@/lib/utils'
import { ThemeToggle } from './ThemeToggle'
import { LanguageSwitch } from '@/components/common/LanguageSwitch'
import { useExport, type ExportType } from '@/hooks/useExport'

interface HeaderProps {
  username?: string
  onLogout: () => void
}

/**
 * Header component with user menu
 * Uses custom popover menu to prevent accidental triggers
 * Includes confirmation dialog for logout to prevent accidental logouts
 */
/**
 * Get human-readable label for export type
 */
function getExportTypeLabel(type: ExportType, t: (key: string) => string): string {
  switch (type) {
    case 'pdf':
      return t('reports.exportTypePdf')
    case 'html':
      return t('reports.exportTypeHtml')
    case 'markdown':
      return t('reports.exportTypeMarkdown')
    default:
      return t('reports.exportTypeFile')
  }
}

export function Header({ username, onLogout }: HeaderProps) {
  const { t } = useTranslation()
  const [open, setOpen] = useState(false)
  const [showLogoutDialog, setShowLogoutDialog] = useState(false)
  const [showExportPopover, setShowExportPopover] = useState(false)
  const menuRef = useRef<HTMLDivElement>(null)
  const buttonRef = useRef<HTMLButtonElement>(null)
  const exportPopoverRef = useRef<HTMLDivElement>(null)
  const exportButtonRef = useRef<HTMLButtonElement>(null)
  
  // Global export state for background download indicator
  const { isExporting, tasks, cancelExport } = useExport()
  
  // Filter only exporting tasks
  const exportingTasks = tasks.filter(t => t.status === 'exporting')

  // Close menu when clicking outside
  useEffect(() => {
    if (!open) return

    const handleClickOutside = (event: MouseEvent) => {
      if (
        menuRef.current &&
        buttonRef.current &&
        !menuRef.current.contains(event.target as Node) &&
        !buttonRef.current.contains(event.target as Node)
      ) {
        setOpen(false)
      }
    }

    document.addEventListener('mousedown', handleClickOutside)
    return () => document.removeEventListener('mousedown', handleClickOutside)
  }, [open])

  // Close export popover when clicking outside
  useEffect(() => {
    if (!showExportPopover) return

    const handleClickOutside = (event: MouseEvent) => {
      if (
        exportPopoverRef.current &&
        exportButtonRef.current &&
        !exportPopoverRef.current.contains(event.target as Node) &&
        !exportButtonRef.current.contains(event.target as Node)
      ) {
        setShowExportPopover(false)
      }
    }

    document.addEventListener('mousedown', handleClickOutside)
    return () => document.removeEventListener('mousedown', handleClickOutside)
  }, [showExportPopover])

  const handleLogoutClick = () => {
    setOpen(false)
    setShowLogoutDialog(true)
  }

  const handleLogoutConfirm = () => {
    setShowLogoutDialog(false)
    onLogout()
  }

  return (
    <>
      <header className="flex h-16 items-center justify-end border-b border-[hsl(var(--border))] bg-[hsl(var(--card))] px-6">
        {/* Actions */}
        <div className="flex items-center gap-2">
          {/* Background export indicator - compact icon button with popover */}
          {isExporting && (
            <div className="relative">
              <button
                ref={exportButtonRef}
                type="button"
                onClick={() => setShowExportPopover(!showExportPopover)}
                className="flex items-center gap-1 rounded-md bg-[hsl(var(--muted))] p-2 transition-colors hover:bg-[hsl(var(--accent))]"
                title={t('reports.viewExportTasks')}
              >
                <Loader2 className="h-4 w-4 animate-spin text-[hsl(var(--primary))]" />
                <Download className="h-4 w-4 text-[hsl(var(--muted-foreground))]" />
                {exportingTasks.length > 1 && (
                  <span className="ml-0.5 text-xs font-medium text-[hsl(var(--muted-foreground))]">
                    {exportingTasks.length}
                  </span>
                )}
              </button>

              {/* Export tasks popover */}
              {showExportPopover && (
                <div
                  ref={exportPopoverRef}
                  className="absolute right-0 top-full z-50 mt-2 w-72 overflow-hidden rounded-md border border-[hsl(var(--border))] bg-[hsl(var(--popover))] p-3 text-[hsl(var(--popover-foreground))] shadow-md"
                >
                  <div className="mb-2 flex items-center gap-2 text-sm font-semibold">
                    <Loader2 className="h-4 w-4 animate-spin" />
                    {t('reports.exportingTasks')}
                  </div>
                  <p className="mb-3 text-xs text-[hsl(var(--muted-foreground))]">
                    {t('reports.exportingTasksDesc')}
                  </p>
                  <div className="space-y-2">
                    {exportingTasks.length === 0 ? (
                      <p className="text-sm text-[hsl(var(--muted-foreground))]">
                        {t('reports.noExportingTasks')}
                      </p>
                    ) : (
                      exportingTasks.map((task) => (
                        <div
                          key={task.id}
                          className="flex items-center justify-between rounded-md border border-[hsl(var(--border))] bg-[hsl(var(--muted))] px-2.5 py-1.5"
                        >
                          <div className="flex items-center gap-2 min-w-0 flex-1">
                            <Loader2 className="h-3.5 w-3.5 animate-spin text-[hsl(var(--primary))] flex-shrink-0" />
                            <span className="truncate text-sm font-medium" title={task.reportTitle}>
                              {task.reportTitle}
                            </span>
                            <span className="flex-shrink-0 rounded bg-[hsl(var(--background))] px-1.5 py-0.5 text-xs text-[hsl(var(--muted-foreground))]">
                              {getExportTypeLabel(task.exportType, t)}
                            </span>
                          </div>
                          <button
                            type="button"
                            onClick={() => cancelExport(task.id)}
                            className="ml-2 flex-shrink-0 rounded p-1 text-[hsl(var(--muted-foreground))] transition-colors hover:bg-[hsl(var(--destructive))] hover:text-[hsl(var(--destructive-foreground))]"
                            title={t('common.cancel')}
                          >
                            <X className="h-3.5 w-3.5" />
                          </button>
                        </div>
                      ))
                    )}
                  </div>
                </div>
              )}
            </div>
          )}
          
          <ThemeToggle />
          <LanguageSwitch />

          {/* User menu */}
          <div className="relative">
            <Button
              ref={buttonRef}
              variant="ghost"
              size="sm"
              className="gap-2"
              type="button"
              onClick={(e) => {
                e.stopPropagation()
                setOpen(!open)
              }}
            >
              <User className="h-4 w-4" />
              <span className="hidden sm:inline">{username || 'Admin'}</span>
            </Button>

            {open && (
              <div
                ref={menuRef}
                className="absolute right-0 top-full z-50 mt-2 w-48 overflow-hidden rounded-md border border-[hsl(var(--border))] bg-[hsl(var(--popover))] p-1 text-[hsl(var(--popover-foreground))] shadow-md"
              >
                <div className="px-2 py-1.5 text-sm font-semibold">
                  {username || 'Admin'}
                </div>
                <div className="-mx-1 my-1 h-px bg-[hsl(var(--muted))]" />
                <button
                  type="button"
                  onClick={(e) => {
                    e.preventDefault()
                    e.stopPropagation()
                    handleLogoutClick()
                  }}
                  className={cn(
                    'relative flex w-full cursor-default select-none items-center rounded-sm px-2 py-1.5 text-sm outline-none transition-colors hover:bg-[hsl(var(--accent))] focus:bg-[hsl(var(--accent))]',
                    'text-[hsl(var(--destructive))]'
                  )}
                >
                  <LogOut className="mr-2 h-4 w-4" />
                  {t('auth.logout')}
                </button>
              </div>
            )}
          </div>
        </div>
      </header>

      {/* Logout confirmation dialog */}
      <Dialog open={showLogoutDialog} onOpenChange={setShowLogoutDialog}>
        <DialogContent>
          <DialogHeader>
            <DialogTitle>{t('auth.logout')}</DialogTitle>
            <DialogDescription>
              {t('auth.logoutConfirm')}
            </DialogDescription>
          </DialogHeader>
          <DialogFooter>
            <Button
              variant="outline"
              onClick={() => setShowLogoutDialog(false)}
            >
              {t('common.cancel')}
            </Button>
            <Button
              variant="destructive"
              onClick={handleLogoutConfirm}
            >
              {t('auth.logout')}
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>
    </>
  )
}
