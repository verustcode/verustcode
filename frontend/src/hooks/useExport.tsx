import { createContext, useContext, useState, useCallback, useMemo, useRef } from 'react'
import type { ReactNode } from 'react'
import { useTranslation } from 'react-i18next'
import { toast } from './useToast'
import { get } from '@/lib/api'

/**
 * Export task type
 */
export type ExportType = 'pdf' | 'markdown' | 'html'

/**
 * Export task status
 */
export type ExportStatus = 'idle' | 'exporting' | 'success' | 'error' | 'cancelled'

/**
 * Export task info
 */
export interface ExportTask {
  id: string
  reportId: string
  reportTitle: string
  exportType: ExportType
  status: ExportStatus
  startTime: number
}

/**
 * Export context value type
 */
interface ExportContextValue {
  /** Current export tasks */
  tasks: ExportTask[]
  /** Whether any export is in progress */
  isExporting: boolean
  /** Current exporting report ID (for button state) */
  exportingReportId: string | null
  /** Current export type (for button state) */
  exportingType: ExportType | null
  /** Current exporting report title (for header indicator) */
  exportingReportTitle: string | null
  /** Start a new export task */
  startExport: (params: StartExportParams) => void
  /** Cancel an export task */
  cancelExport: (taskId: string) => void
}

/**
 * Parameters for starting an export
 */
interface StartExportParams {
  reportId: string
  reportTitle: string
  exportType: ExportType
}

/**
 * Export context for sharing export state across components
 */
const ExportContext = createContext<ExportContextValue | null>(null)

/**
 * Export provider props
 */
interface ExportProviderProps {
  children: ReactNode
}

/**
 * Generate unique task ID
 */
let taskIdCounter = 0
function generateTaskId(): string {
  return `export-${Date.now()}-${++taskIdCounter}`
}

/**
 * Export provider component that manages global export state
 */
export function ExportProvider({ children }: ExportProviderProps) {
  const { t } = useTranslation()
  const [tasks, setTasks] = useState<ExportTask[]>([])
  
  // Store AbortControllers for each task
  const abortControllersRef = useRef<Map<string, AbortController>>(new Map())

  const isExporting = tasks.some(t => t.status === 'exporting')
  
  const exportingTask = tasks.find(t => t.status === 'exporting')
  const exportingReportId = exportingTask?.reportId ?? null
  const exportingType = exportingTask?.exportType ?? null
  const exportingReportTitle = exportingTask?.reportTitle ?? null

  // Cancel an export task
  const cancelExport = useCallback((taskId: string) => {
    const controller = abortControllersRef.current.get(taskId)
    if (controller) {
      controller.abort()
      abortControllersRef.current.delete(taskId)
    }
    
    // Remove the task from state
    setTasks(prev => prev.filter(t => t.id !== taskId))
    
    toast({
      title: t('reports.exportCancelledTitle'),
      description: t('reports.exportCancelledDesc'),
    })
  }, [])

  const startExport = useCallback(async (params: StartExportParams) => {
    const { reportId, reportTitle, exportType } = params
    
    // Check if already exporting this report
    const existingTask = tasks.find(
      t => t.reportId === reportId && t.status === 'exporting'
    )
    if (existingTask) {
      toast({
        title: t('reports.exportInProgressTitle'),
        description: t('reports.exportInProgressDesc', { title: reportTitle }),
      })
      return
    }

    const taskId = generateTaskId()
    const newTask: ExportTask = {
      id: taskId,
      reportId,
      reportTitle,
      exportType,
      status: 'exporting',
      startTime: Date.now(),
    }

    // Create AbortController for this task
    const abortController = new AbortController()
    abortControllersRef.current.set(taskId, abortController)

    // Add task to state
    setTasks(prev => [...prev, newTask])

    // Show starting toast
    const exportTypeLabel = getExportTypeLabel(exportType, t)
    toast({
      title: t('reports.exportStartingTitle', { type: exportTypeLabel }),
      description: t('reports.exportStartingDesc', { title: reportTitle }),
    })

    try {
      let blob: Blob
      const signal = abortController.signal

      if (exportType === 'pdf') {
        blob = await get<Blob>(`/reports/${reportId}/export`, {
          params: { format: 'pdf' },
          responseType: 'blob',
          timeout: 180000,
          signal,
        })
      } else if (exportType === 'html') {
        blob = await get<Blob>(`/reports/${reportId}/export`, {
          params: { format: 'html' },
          responseType: 'blob',
          timeout: 30000,
          signal,
        })
      } else {
        blob = await get<Blob>(`/reports/${reportId}/export`, {
          params: { format: 'markdown' },
          responseType: 'blob',
          timeout: 30000,
          signal,
        })
      }

      // Clean up abort controller
      abortControllersRef.current.delete(taskId)

      // Trigger download
      const url = window.URL.createObjectURL(blob)
      const a = document.createElement('a')
      a.href = url
      a.download = getFilename(reportTitle || reportId, exportType)
      document.body.appendChild(a)
      a.click()
      window.URL.revokeObjectURL(url)
      document.body.removeChild(a)

      // Update task status
      setTasks(prev => prev.map(t => 
        t.id === taskId ? { ...t, status: 'success' as const } : t
      ))

      // Show success toast
      toast({
        title: t('reports.exportCompletedTitle', { type: exportTypeLabel }),
        description: t('reports.exportCompletedDesc', { title: reportTitle }),
      })

      // Remove completed task after a delay
      setTimeout(() => {
        setTasks(prev => prev.filter(t => t.id !== taskId))
      }, 3000)

    } catch (error) {
      // Clean up abort controller
      abortControllersRef.current.delete(taskId)

      // Check if the request was cancelled
      if (error instanceof Error && error.name === 'CanceledError') {
        // Already handled by cancelExport, just update status
        setTasks(prev => prev.map(t => 
          t.id === taskId ? { ...t, status: 'cancelled' as const } : t
        ))
        return
      }

      console.error('Export failed:', error)
      
      // Update task status
      setTasks(prev => prev.map(t => 
        t.id === taskId ? { ...t, status: 'error' as const } : t
      ))

      // Show error toast
      toast({
        title: t('reports.exportFailedTitle', { type: getExportTypeLabel(exportType, t) }),
        description: error instanceof Error ? error.message : t('reports.exportRetryLater'),
        variant: 'destructive',
      })

      // Remove failed task after a delay
      setTimeout(() => {
        setTasks(prev => prev.filter(t => t.id !== taskId))
      }, 5000)
    }
  }, [tasks])

  const value = useMemo<ExportContextValue>(() => ({
    tasks,
    isExporting,
    exportingReportId,
    exportingType,
    exportingReportTitle,
    startExport,
    cancelExport,
  }), [tasks, isExporting, exportingReportId, exportingType, exportingReportTitle, startExport, cancelExport])

  return (
    <ExportContext.Provider value={value}>
      {children}
    </ExportContext.Provider>
  )
}

/**
 * Hook for consuming export context
 * Must be used within an ExportProvider
 */
export function useExport(): ExportContextValue {
  const context = useContext(ExportContext)
  if (!context) {
    throw new Error('useExport must be used within an ExportProvider')
  }
  return context
}

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

/**
 * Generate filename for download
 */
function getFilename(title: string, type: ExportType): string {
  const sanitized = title.replace(/[/\\?%*:|"<>]/g, '-')
  
  switch (type) {
    case 'pdf':
      return `${sanitized}.pdf`
    case 'html':
      return `${sanitized}.html`
    case 'markdown':
      return `${sanitized}.md`
    default:
      return sanitized
  }
}

