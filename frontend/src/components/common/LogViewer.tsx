import { useState, useEffect, useCallback, useMemo, useRef } from 'react'
import { useTranslation } from 'react-i18next'
import { useQuery } from '@tanstack/react-query'
import { Terminal, RefreshCw, ChevronDown, ChevronRight, ChevronsDownUp, ChevronsUpDown, Filter, X, AlertCircle, Copy, Check } from 'lucide-react'

import { api } from '@/lib/api'
import type { TaskLog, LogLevel, TaskType, TaskLogsResponse } from '@/types/api'
import { toast } from '@/hooks/useToast'

import { Button } from '@/components/ui/button'
import { Badge } from '@/components/ui/badge'
import { ScrollArea } from '@/components/ui/scroll-area'
import { Switch } from '@/components/ui/switch'
import { Label } from '@/components/ui/label'
import {
  Dialog,
  DialogContent,
  DialogHeader,
  DialogTitle,
  DialogTrigger,
} from '@/components/ui/dialog'
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select'

interface LogViewerProps {
  taskType: TaskType
  taskId: string
  trigger?: React.ReactNode
}

const LOG_LEVELS: LogLevel[] = ['debug', 'info', 'warn', 'error', 'fatal']

/**
 * LogViewer component displays task logs in a dialog.
 * Supports filtering by log level and auto-refresh.
 */
export function LogViewer({ taskType, taskId, trigger }: LogViewerProps) {
  const { t } = useTranslation()
  const [open, setOpen] = useState(false)
  const [levelFilter, setLevelFilter] = useState<LogLevel | 'all'>('all')
  
  // localStorage key for this task
  const storageKey = `task-log-${taskType}-${taskId}`
  
  // Load state from localStorage on initialization
  const loadStateFromStorage = useCallback(() => {
    try {
      const saved = localStorage.getItem(storageKey)
      if (saved) {
        const parsed = JSON.parse(saved)
        return {
          expandedLogs: parsed.expandedLogs ? new Set(parsed.expandedLogs as number[]) : new Set<number>(),
          showCaller: parsed.showCaller ?? false,
          autoRefresh: parsed.autoRefresh ?? false,
        }
      }
    } catch (err) {
      console.warn('Failed to load log viewer state from localStorage:', err)
    }
    return {
      expandedLogs: new Set<number>(),
      showCaller: false,
      autoRefresh: false,
    }
  }, [storageKey])
  
  // Initialize state from localStorage
  const [expandedLogs, setExpandedLogs] = useState<Set<number>>(() => {
    return loadStateFromStorage().expandedLogs
  })
  const [autoRefresh, setAutoRefresh] = useState(() => {
    return loadStateFromStorage().autoRefresh
  })
  const [showCaller, setShowCaller] = useState(() => {
    return loadStateFromStorage().showCaller
  })
  const [copiedLogId, setCopiedLogId] = useState<number | null>(null)
  const scrollAreaRef = useRef<HTMLDivElement>(null)
  
  // Reload state when taskId or taskType changes
  useEffect(() => {
    const state = loadStateFromStorage()
    setExpandedLogs(state.expandedLogs)
    setShowCaller(state.showCaller)
    setAutoRefresh(state.autoRefresh)
  }, [taskId, taskType, loadStateFromStorage])
  
  // Save state to localStorage
  const saveStateToStorage = useCallback((expanded: Set<number>, caller: boolean, refresh: boolean) => {
    try {
      localStorage.setItem(storageKey, JSON.stringify({
        expandedLogs: Array.from(expanded),
        showCaller: caller,
        autoRefresh: refresh,
      }))
    } catch (err) {
      console.warn('Failed to save log viewer state to localStorage:', err)
    }
  }, [storageKey])

  // Fetch logs based on task type
  const fetchLogs = useCallback(async (): Promise<TaskLogsResponse> => {
    const params = {
      page: 1,
      page_size: 500,
      ...(levelFilter !== 'all' ? { level: levelFilter } : {}),
    }
    
    if (taskType === 'review') {
      return api.reviews.getLogs(taskId, params)
    } else {
      return api.reports.getLogs(taskId, params)
    }
  }, [taskType, taskId, levelFilter])

  const { data, isLoading, isError, error, refetch, isFetching } = useQuery({
    queryKey: ['task-logs', taskType, taskId, levelFilter],
    queryFn: fetchLogs,
    enabled: open,
    refetchInterval: autoRefresh ? 3000 : false,
    staleTime: 2000,
  })

  // Save state to localStorage when expandedLogs, showCaller, or autoRefresh changes
  useEffect(() => {
    saveStateToStorage(expandedLogs, showCaller, autoRefresh)
  }, [expandedLogs, showCaller, autoRefresh, saveStateToStorage])

  const logs = data?.data ?? []

  // Auto-scroll to bottom when auto-refresh is enabled and logs update
  useEffect(() => {
    if (!autoRefresh || !data?.data?.length || !scrollAreaRef.current || !open) {
      return
    }

    // Find the viewport element inside ScrollArea
    // Radix UI ScrollArea creates a viewport with data-radix-scroll-area-viewport attribute
    const viewport = scrollAreaRef.current.querySelector('[data-radix-scroll-area-viewport]') as HTMLElement
    if (viewport) {
      // Use requestAnimationFrame to ensure DOM has updated, then scroll
      requestAnimationFrame(() => {
        requestAnimationFrame(() => {
          viewport.scrollTo({
            top: viewport.scrollHeight,
            behavior: 'smooth',
          })
        })
      })
    }
  }, [data, autoRefresh, open])

  const toggleExpand = useCallback((logId: number) => {
    setExpandedLogs(prev => {
      const next = new Set(prev)
      if (next.has(logId)) {
        next.delete(logId)
      } else {
        next.add(logId)
      }
      return next
    })
  }, [])

  // Handle copy message to clipboard
  const handleCopyMessage = useCallback(async (logId: number, message: string) => {
    try {
      await navigator.clipboard.writeText(message)
      setCopiedLogId(logId)
      toast({
        title: t('common.copy'),
        description: t('logs.messageCopied'),
      })
      setTimeout(() => setCopiedLogId(null), 2000)
    } catch (err) {
      console.error('Failed to copy message:', err)
      toast({
        title: t('logs.copyError'),
        variant: 'destructive',
      })
    }
  }, [t])

  // Get logs with fields
  const logsWithFields = useMemo(() => {
    return logs.filter(log => log.fields && Object.keys(log.fields).length > 0)
  }, [logs])

  // Expand/collapse all logs with fields
  const expandAll = useCallback(() => {
    setExpandedLogs(new Set(logsWithFields.map(log => log.id)))
  }, [logsWithFields])

  const collapseAll = useCallback(() => {
    setExpandedLogs(new Set())
  }, [])

  // Check if all expandable logs are expanded
  const allExpanded = useMemo(() => {
    return logsWithFields.length > 0 && logsWithFields.every(log => expandedLogs.has(log.id))
  }, [logsWithFields, expandedLogs])

  const getLevelBadgeVariant = useCallback((level: LogLevel): 'default' | 'secondary' | 'destructive' | 'outline' | 'success' | 'warning' | 'info' => {
    switch (level) {
      case 'debug':
        return 'secondary'
      case 'info':
        return 'info'
      case 'warn':
        return 'warning'
      case 'error':
      case 'fatal':
        return 'destructive'
      default:
        return 'default'
    }
  }, [])

  const formatTimestamp = useCallback((timestamp: string): string => {
    const date = new Date(timestamp)
    return date.toLocaleString(undefined, {
      year: 'numeric',
      month: '2-digit',
      day: '2-digit',
      hour: '2-digit',
      minute: '2-digit',
      second: '2-digit',
      fractionalSecondDigits: 3,
    })
  }, [])

  // Format caller to show only last 2 parts (e.g., report/summary.go:160)
  const formatCaller = useCallback((caller: string): string => {
    if (!caller) return ''
    // Split by '/' and take last 2 parts
    const parts = caller.split('/')
    if (parts.length >= 2) {
      return parts.slice(-2).join('/')
    }
    return caller
  }, [])

  // Memoize rendered log entries
  const logEntries = useMemo(() => {
    return logs.map((log: TaskLog) => {
      const isExpanded = expandedLogs.has(log.id)
      const hasFields = log.fields && Object.keys(log.fields).length > 0

      return (
        <div
          key={log.id}
          className="border-b border-border/30 last:border-b-0 hover:bg-muted/30 transition-colors"
        >
          <div
            className="flex items-center gap-2 px-3 py-2 cursor-pointer"
            onClick={() => hasFields && toggleExpand(log.id)}
          >
            {/* Expand icon - only show if has fields */}
            <div className="w-4 h-4 flex-shrink-0">
              {hasFields ? (
                isExpanded ? (
                  <ChevronDown className="h-4 w-4 text-muted-foreground" />
                ) : (
                  <ChevronRight className="h-4 w-4 text-muted-foreground" />
                )
              ) : null}
            </div>

            {/* Timestamp */}
            <span className="text-xs text-muted-foreground font-mono whitespace-nowrap flex-shrink-0">
              {formatTimestamp(log.created_at)}
            </span>

            {/* Level badge */}
            <Badge
              variant={getLevelBadgeVariant(log.level)}
              className="text-xs px-1.5 py-0 flex-shrink-0 uppercase font-mono"
            >
              {log.level}
            </Badge>

            {/* Logger name */}
            {log.logger && (
              <span className="text-xs text-muted-foreground font-mono flex-shrink-0">
                [{log.logger}]
              </span>
            )}

            {/* Caller (file:line) */}
            {showCaller && log.caller && (
              <span className="text-xs text-muted-foreground font-mono flex-shrink-0">
                [{formatCaller(log.caller)}]
              </span>
            )}

            {/* Message */}
            <div className="flex items-start gap-1 flex-1 min-w-0 group">
              <span className="text-sm font-mono break-all">
                {log.message}
              </span>
              <button
                onClick={(e) => {
                  e.stopPropagation()
                  handleCopyMessage(log.id, log.message)
                }}
                className="flex-shrink-0 p-0.5 rounded hover:bg-muted transition-colors opacity-0 group-hover:opacity-100 mt-0.5"
                title={t('common.copy')}
              >
                {copiedLogId === log.id ? (
                  <Check className="h-3 w-3 text-green-500" />
                ) : (
                  <Copy className="h-3 w-3 text-muted-foreground opacity-50 hover:opacity-100" />
                )}
              </button>
            </div>
          </div>

          {/* Expanded fields */}
          {isExpanded && hasFields && (
            <div className="pl-[2.25rem] pr-3 pt-0 pb-0.5">
              <div className="bg-muted/50 rounded-md px-2 py-1 text-xs font-mono">
                {Object.entries(log.fields!).map(([key, value]) => (
                  <div key={key} className="flex gap-2">
                    <span className="text-blue-500 dark:text-blue-400">{key}:</span>
                    <span className="text-foreground break-all">
                      {typeof value === 'object' ? JSON.stringify(value) : String(value)}
                    </span>
                  </div>
                ))}
              </div>
            </div>
          )}
        </div>
      )
    })
  }, [logs, expandedLogs, showCaller, toggleExpand, formatTimestamp, formatCaller, getLevelBadgeVariant, copiedLogId, handleCopyMessage, t])

  const defaultTrigger = (
    <Button variant="outline" size="sm" className="gap-2">
      <Terminal className="h-4 w-4" />
      {t('logs.viewLogs')}
    </Button>
  )

  return (
    <Dialog open={open} onOpenChange={setOpen}>
      <DialogTrigger asChild>
        {trigger ?? defaultTrigger}
      </DialogTrigger>
      <DialogContent className="max-w-6xl h-[80vh] flex flex-col p-0 gap-0">
        <DialogHeader className="px-6 py-4 border-b">
          <div className="flex items-center justify-between">
            <div>
              <DialogTitle className="flex items-center gap-2">
                <Terminal className="h-5 w-5" />
                {t('logs.title')}
              </DialogTitle>
            </div>
          </div>

          {/* Toolbar */}
          <div className="flex items-center gap-3 mt-4">
            {/* Level filter */}
            <div className="flex items-center gap-2">
              <Filter className="h-4 w-4 text-muted-foreground" />
              <Select
                value={levelFilter}
                onValueChange={(value) => setLevelFilter(value as LogLevel | 'all')}
              >
                <SelectTrigger className="w-[130px] h-8">
                  <SelectValue placeholder={t('logs.filterLevel')} />
                </SelectTrigger>
                <SelectContent>
                  <SelectItem value="all">{t('common.all')}</SelectItem>
                  {LOG_LEVELS.map((level) => (
                    <SelectItem key={level} value={level}>
                      <span className="uppercase">{t(`logs.level.${level}`)}</span>
                    </SelectItem>
                  ))}
                </SelectContent>
              </Select>
              {levelFilter !== 'all' && (
                <Button
                  variant="ghost"
                  size="icon"
                  className="h-8 w-8"
                  onClick={() => setLevelFilter('all')}
                >
                  <X className="h-4 w-4" />
                </Button>
              )}
            </div>

            {/* Expand/Collapse All */}
            {logsWithFields.length > 0 && (
              <Button
                variant="outline"
                size="sm"
                className="gap-2"
                onClick={allExpanded ? collapseAll : expandAll}
              >
                {allExpanded ? (
                  <>
                    <ChevronsUpDown className="h-4 w-4" />
                    {t('logs.collapseAll')}
                  </>
                ) : (
                  <>
                    <ChevronsDownUp className="h-4 w-4" />
                    {t('logs.expandAll')}
                  </>
                )}
              </Button>
            )}

            {/* Show caller toggle */}
            <div className="flex items-center gap-2">
              <Label htmlFor="show-caller" className="text-sm cursor-pointer">
                {t('logs.showCaller')}
              </Label>
              <Switch
                id="show-caller"
                checked={showCaller}
                onCheckedChange={(checked) => {
                  setShowCaller(checked)
                }}
              />
            </div>

            <div className="flex-1" />

            {/* Auto-refresh toggle */}
            <div className="flex items-center gap-2">
              <Label htmlFor="auto-refresh" className="text-sm cursor-pointer">
                {t('logs.autoRefresh')}
              </Label>
              <Switch
                id="auto-refresh"
                checked={autoRefresh}
                onCheckedChange={setAutoRefresh}
              />
            </div>

            {/* Manual refresh */}
            <Button
              variant="outline"
              size="sm"
              className="gap-2"
              onClick={() => refetch()}
              disabled={isFetching}
            >
              <RefreshCw className={`h-4 w-4 ${isFetching ? 'animate-spin' : ''}`} />
              {t('common.refresh')}
            </Button>
          </div>
        </DialogHeader>

        {/* Log content */}
        <ScrollArea ref={scrollAreaRef} className="flex-1">
          <div className="min-h-full bg-background">
            {isLoading ? (
              <div className="flex items-center justify-center h-48 text-muted-foreground">
                <RefreshCw className="h-5 w-5 animate-spin mr-2" />
                {t('common.loading')}
              </div>
            ) : isError ? (
              <div className="flex flex-col items-center justify-center h-48 text-destructive gap-2">
                <AlertCircle className="h-6 w-6" />
                <span>{t('logs.loadError')}</span>
                <span className="text-xs text-muted-foreground">
                  {error instanceof Error ? error.message : String(error)}
                </span>
                <Button variant="outline" size="sm" onClick={() => refetch()}>
                  {t('common.refresh')}
                </Button>
              </div>
            ) : logs.length === 0 ? (
              <div className="flex flex-col items-center justify-center h-48 text-muted-foreground gap-2">
                <Terminal className="h-8 w-8 opacity-50" />
                <span>{t('logs.noLogs')}</span>
              </div>
            ) : (
              <div className="divide-y divide-border/30">
                {logEntries}
              </div>
            )}
          </div>
        </ScrollArea>

        {/* Footer with stats */}
        {!isLoading && !isError && logs.length > 0 && (
          <div className="px-6 py-3 border-t bg-muted/30 text-xs text-muted-foreground flex items-center justify-between">
            <div className="flex items-center gap-4">
              <span>
                {t('logs.totalLogs', { count: data?.total ?? logs.length })}
              </span>
              <span>
                {t('logs.description', {
                  type: t(`logs.taskType.${taskType}`),
                  id: taskId,
                })}
              </span>
            </div>
            {data?.total && data.total > logs.length && (
              <span className="text-amber-600 dark:text-amber-400">
                {t('logs.showingLatest', { count: logs.length })}
              </span>
            )}
          </div>
        )}
      </DialogContent>
    </Dialog>
  )
}

