import { useState, useEffect, useRef, useMemo } from 'react'
import { useTranslation } from 'react-i18next'
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query'
import { useSearchParams } from 'react-router-dom'
import { Save, RotateCcw, CheckCircle, RefreshCw, Eye, Plus, ChevronLeft, ChevronRight } from 'lucide-react'
import yaml from 'js-yaml'
import { AxiosError } from 'axios'
import { Button } from '@/components/ui/button'
import { ModeSwitch } from '@/components/common/ModeSwitch'
import { LoadingSpinner } from '@/components/common/LoadingSpinner'
import { cn } from '@/lib/utils'
import { api } from '@/lib/api'
import { YamlEditor } from '@/components/editor/YamlEditor'
import { ReportTypeOverviewCards } from '@/components/reports/ReportTypeOverviewCards'
import { toast } from '@/hooks/useToast'
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select'
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from '@/components/ui/dialog'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'

type EditorMode = 'form' | 'yaml'

type SaveResult = {
  success: boolean
  isConflict?: boolean
}

// Default report type file
const DEFAULT_REPORT_TYPE_FILE = 'wiki.yaml'
// Default example template file
const DEFAULT_EXAMPLE_FILE = 'wiki_simple.yaml'

/**
 * Report Types management page component
 * Supports multiple report type files in config/reports/ directory
 * Uses YAML editor for editing and read-only cards for overview
 * Supports optimistic locking via hash
 */
export default function ReportTypes() {
  const { t } = useTranslation()
  const queryClient = useQueryClient()
  const [searchParams, setSearchParams] = useSearchParams()

  // Selected file state - support URL query parameter for initial selection
  const [selectedFile, setSelectedFile] = useState<string>(() => {
    const fileParam = searchParams.get('file')
    return fileParam || DEFAULT_REPORT_TYPE_FILE
  })
  
  // Default to YAML mode
  const [mode, setMode] = useState<EditorMode>('yaml')
  const [content, setContent] = useState('')
  const [originalContent, setOriginalContent] = useState('')
  const [hash, setHash] = useState('')
  const [saveResult, setSaveResult] = useState<SaveResult | null>(null)
  
  // New report type dialog state
  const [showNewReportTypeDialog, setShowNewReportTypeDialog] = useState(false)
  const [newReportTypeName, setNewReportTypeName] = useState('')
  const [copyFromFile, setCopyFromFile] = useState(DEFAULT_EXAMPLE_FILE)
  
  // Tab scroll ref
  const tabsContainerRef = useRef<HTMLDivElement>(null)
  const [canScrollLeft, setCanScrollLeft] = useState(false)
  const [canScrollRight, setCanScrollRight] = useState(false)

  // Fetch list of report type files
  const { data: reportTypesListData, isLoading: listLoading } = useQuery({
    queryKey: ['admin', 'report-types', 'list'],
    queryFn: () => api.admin.reportTypes.list(),
  })

  // Sort report type files: default file first, then others
  const reportTypeFiles = useMemo(() => {
    const files = reportTypesListData?.files || []
    return [...files].sort((a, b) => {
      // Default file always comes first
      if (a.name === DEFAULT_REPORT_TYPE_FILE) return -1
      if (b.name === DEFAULT_REPORT_TYPE_FILE) return 1
      // Keep original order for other files
      return 0
    })
  }, [reportTypesListData?.files])

  // Fetch selected report type file content
  const { data: fileContent, isLoading: contentLoading, refetch } = useQuery({
    queryKey: ['admin', 'report-types', selectedFile],
    queryFn: () => api.admin.reportTypes.get(selectedFile),
    enabled: !!selectedFile,
  })

  // Handle URL query parameter for file selection
  useEffect(() => {
    const fileParam = searchParams.get('file')
    if (fileParam && fileParam !== selectedFile) {
      setSelectedFile(fileParam)
      // Clear the query parameter from URL after processing
      setSearchParams({}, { replace: true })
    }
  }, [searchParams, selectedFile, setSearchParams])

  // Update content when file is loaded
  useEffect(() => {
    if (fileContent?.content) {
      setContent(fileContent.content)
      setOriginalContent(fileContent.content)
      setHash(fileContent.hash)
      setSaveResult(null)
    }
  }, [fileContent])

  // Save mutation with optimistic locking
  const saveMutation = useMutation({
    mutationFn: ({ content, hash }: { content: string; hash: string }) =>
      api.admin.reportTypes.save(selectedFile, content, hash),
    onSuccess: (data) => {
      queryClient.invalidateQueries({ queryKey: ['admin', 'report-types'] })
      setOriginalContent(content)
      // Update hash with new hash from response
      if (data.hash) {
        setHash(data.hash)
      }
      setSaveResult({ success: true })
    },
    onError: (error) => {
      // Check for 409 Conflict - show inline with refresh button
      if (error instanceof AxiosError && error.response?.status === 409) {
        setSaveResult({
          success: false,
          isConflict: true,
        })
      }
      // Other errors are handled by global toast in api.ts
    },
  })

  // Create new file mutation
  const createFileMutation = useMutation({
    mutationFn: ({ name, copyFrom }: { name: string; copyFrom: string }) =>
      api.admin.reportTypes.create(name, copyFrom),
    onSuccess: (data) => {
      queryClient.invalidateQueries({ queryKey: ['admin', 'report-types', 'list'] })
      setShowNewReportTypeDialog(false)
      setNewReportTypeName('')
      setCopyFromFile(DEFAULT_EXAMPLE_FILE)
      // Switch to the new file
      if (data.name) {
        setSelectedFile(data.name)
      }
      toast({ title: t('reportTypes.createSuccess') })
    },
  })
  
  // Check scroll state for tab arrows
  const updateScrollState = () => {
    const container = tabsContainerRef.current
    if (container) {
      setCanScrollLeft(container.scrollLeft > 0)
      setCanScrollRight(
        container.scrollLeft < container.scrollWidth - container.clientWidth - 1
      )
    }
  }
  
  // Scroll tabs left/right
  const scrollTabs = (direction: 'left' | 'right') => {
    const container = tabsContainerRef.current
    if (container) {
      const scrollAmount = 150
      container.scrollBy({
        left: direction === 'left' ? -scrollAmount : scrollAmount,
        behavior: 'smooth',
      })
    }
  }
  
  // Update scroll state on resize and initial render, add wheel scroll support
  useEffect(() => {
    const container = tabsContainerRef.current
    if (container) {
      updateScrollState()
      container.addEventListener('scroll', updateScrollState)
      window.addEventListener('resize', updateScrollState)
      
      // Handle wheel event to enable horizontal scrolling on hover
      const handleWheel = (e: WheelEvent) => {
        if (e.deltaY !== 0) {
          e.preventDefault()
          container.scrollBy({
            left: e.deltaY,
            behavior: 'smooth',
          })
        }
      }
      container.addEventListener('wheel', handleWheel, { passive: false })
      
      return () => {
        container.removeEventListener('scroll', updateScrollState)
        window.removeEventListener('resize', updateScrollState)
        container.removeEventListener('wheel', handleWheel)
      }
    }
  }, [reportTypeFiles])

  // Determine if scroll buttons should be shown (only when content is scrollable)
  const showScrollButtons = canScrollLeft || canScrollRight

  const hasChanges = content !== originalContent

  const handleSave = () => {
    try {
      // Validate YAML syntax only - don't transform
      yaml.load(content)
      // Send original YAML string with hash for optimistic locking
      saveMutation.mutate({ content, hash })
    } catch (error) {
      // Show YAML syntax error via toast
      toast({
        variant: 'destructive',
        title: t('common.invalidYaml'),
        description: error instanceof Error ? error.message : 'YAML syntax error',
      })
    }
  }

  const handleReset = () => {
    setContent(originalContent)
    setSaveResult(null)
  }

  const handleRefresh = () => {
    refetch()
    setSaveResult(null)
  }

  // Handle mode switch
  const handleModeChange = (newMode: EditorMode) => {
    setMode(newMode)
  }

  // Handle file selection change
  const handleFileChange = (file: string) => {
    if (hasChanges) {
      // Warn user about unsaved changes
      if (!window.confirm(t('reportTypes.unsavedChangesWarning'))) {
        return
      }
    }
    setSelectedFile(file)
    setSaveResult(null)
  }

  // Handle create new report type
  const handleCreateNewReportType = () => {
    let fileName = newReportTypeName.trim()
    if (!fileName) return
    
    // Ensure .yaml extension
    if (!fileName.endsWith('.yaml') && !fileName.endsWith('.yml')) {
      fileName = fileName + '.yaml'
    }
    
    createFileMutation.mutate({
      name: fileName,
      copyFrom: copyFromFile,
    })
  }
  
  // Get available copy sources (example file + all existing files)
  const getCopySourceFiles = () => {
    const sources: string[] = [DEFAULT_EXAMPLE_FILE]
    for (const file of reportTypeFiles) {
      if (!sources.includes(file.name)) {
        sources.push(file.name)
      }
    }
    return sources
  }

  return (
    <div className="flex h-[calc(100vh-7rem)] flex-col">
      <div className="flex flex-1 flex-col overflow-hidden">
        {/* Tab navigation bar */}
        <div className="flex items-center gap-2 mb-4">
          {/* Scroll left button - only show when content is scrollable */}
          {showScrollButtons && (
            <Button
              variant="ghost"
              size="icon"
              className={cn(
                'h-8 w-8 flex-shrink-0',
                !canScrollLeft && 'opacity-30 cursor-default'
              )}
              onClick={() => scrollTabs('left')}
              disabled={!canScrollLeft}
            >
              <ChevronLeft className="h-4 w-4" />
            </Button>
          )}
          
          {/* Scrollable tabs container */}
          <div
            ref={tabsContainerRef}
            className="flex-1 overflow-x-auto scrollbar-hide"
            style={{ scrollbarWidth: 'none', msOverflowStyle: 'none' }}
          >
            <div className="flex items-center gap-1 min-w-max">
              {listLoading ? (
                <div className="px-4 py-2">
                  <LoadingSpinner size="sm" />
                </div>
              ) : (
                reportTypeFiles.map((file) => (
                  <button
                    key={file.name}
                    onClick={() => handleFileChange(file.name)}
                    className={cn(
                      // Base styles
                      'relative px-4 py-2 text-sm font-medium whitespace-nowrap transition-colors duration-200',
                      // Hover state (when not active)
                      'hover:text-[hsl(var(--foreground))]',
                      // Active/inactive text color
                      selectedFile === file.name
                        ? 'text-[hsl(var(--primary))]'
                        : 'text-[hsl(var(--muted-foreground))]',
                      // Bottom highlight border using after pseudo-element
                      // Match standard tabs component style from Settings page
                      'after:absolute after:bottom-0 after:left-0 after:right-0 after:h-0.5 after:transition-colors after:duration-200',
                      selectedFile === file.name && 'after:bg-[hsl(var(--primary))]'
                    )}
                  >
                    {file.name}
                    {file.name === DEFAULT_REPORT_TYPE_FILE && (
                      <span className="ml-1 text-xs opacity-60">({t('reportTypes.default')})</span>
                    )}
                  </button>
                ))
              )}
            </div>
          </div>
          
          {/* Scroll right button - only show when content is scrollable */}
          {showScrollButtons && (
            <Button
              variant="ghost"
              size="icon"
              className={cn(
                'h-8 w-8 flex-shrink-0',
                !canScrollRight && 'opacity-30 cursor-default'
              )}
              onClick={() => scrollTabs('right')}
              disabled={!canScrollRight}
            >
              <ChevronRight className="h-4 w-4" />
            </Button>
          )}
          
          {/* New report type button */}
          <Button
            size="sm"
            onClick={() => setShowNewReportTypeDialog(true)}
            className="flex-shrink-0 mb-1"
          >
            <Plus className="mr-1 h-4 w-4" />
            {t('reportTypes.newReportType')}
          </Button>
        </div>

        {/* Editor area - use invisible + absolute to preserve editor state when switching modes */}
        {contentLoading ? (
          <div className="flex flex-1 items-center justify-center">
            <LoadingSpinner size="lg" />
          </div>
        ) : (
          <div className="relative flex-1">
            {/* YAML Editor - always rendered, invisible when in form mode */}
            <div className={cn(
              'absolute inset-0 overflow-hidden rounded-md border border-[hsl(var(--border))]',
              mode !== 'yaml' && 'invisible'
            )}>
              <YamlEditor
                value={content}
                onChange={setContent}
              />
            </div>
            {/* Form Mode - read-only overview cards */}
            <div className={cn('absolute inset-0 overflow-auto', mode !== 'form' && 'invisible')}>
              <ReportTypeOverviewCards content={content} />
            </div>
          </div>
        )}

        {/* Save result - only show success or conflict messages */}
        {saveResult && (
          <div
            className={cn(
              'mt-6 flex items-center gap-3 rounded-md p-4',
              saveResult.success
                ? 'bg-emerald-500/10 text-emerald-600 dark:text-emerald-400'
                : 'bg-amber-500/10 text-amber-600 dark:text-amber-400'
            )}
          >
            {saveResult.success ? (
              <>
                <CheckCircle className="h-4 w-4" />
                {t('reportTypes.saveSuccess')}
              </>
            ) : saveResult.isConflict ? (
              <>
                <span className="flex-1">{t('reportTypes.conflictError')}</span>
                <Button variant="outline" size="sm" onClick={handleRefresh}>
                  <RefreshCw className="mr-2 h-4 w-4" />
                  {t('common.refresh')}
                </Button>
              </>
            ) : null}
          </div>
        )}

        {/* Actions */}
        <div className="mt-4 flex items-center justify-between gap-3 pb-2">
          {/* Left side: Mode switch + notice */}
          <div className="flex items-center gap-3">
            <ModeSwitch mode={mode} onModeChange={handleModeChange} disabled={contentLoading} />
            {/* Mode-specific notice */}
            {mode !== 'yaml' && (
              <div className="flex items-center gap-2 rounded-md bg-amber-500/10 px-3 py-1.5 text-sm text-amber-600 dark:text-amber-400">
                <Eye className="h-4 w-4 flex-shrink-0" />
                <span>{t('reportTypes.readOnlyHint')}</span>
              </div>
            )}
          </div>

          {/* Right side: Action buttons */}
          <div className="flex items-center gap-3">
            <Button
              size="sm"
              variant="outline"
              onClick={handleReset}
              disabled={!hasChanges}
            >
              <RotateCcw className="mr-2 h-4 w-4" />
              {t('common.reset')}
            </Button>
            <Button
              size="sm"
              onClick={handleSave}
              disabled={!hasChanges || saveMutation.isPending || mode === 'form'}
            >
              {saveMutation.isPending ? (
                <LoadingSpinner size="sm" className="text-[hsl(var(--primary-foreground))]" />
              ) : (
                <>
                  <Save className="mr-2 h-4 w-4" />
                  {t('common.save')}
                </>
              )}
            </Button>
          </div>
        </div>
      </div>

      {/* New Report Type Dialog */}
      <Dialog open={showNewReportTypeDialog} onOpenChange={setShowNewReportTypeDialog}>
        <DialogContent>
          <DialogHeader>
            <DialogTitle>{t('reportTypes.newReportType')}</DialogTitle>
            <DialogDescription>
              {t('reportTypes.newReportTypeDescription')}
            </DialogDescription>
          </DialogHeader>
          <div className="grid gap-4 py-4">
            <div className="grid gap-2">
              <Label htmlFor="fileName">{t('reportTypes.fileName')}</Label>
              <Input
                id="fileName"
                value={newReportTypeName}
                onChange={(e) => setNewReportTypeName(e.target.value)}
                placeholder="my-report.yaml"
              />
            </div>
            <div className="grid gap-2">
              <Label htmlFor="copyFrom">{t('reportTypes.copyFrom')}</Label>
              <Select
                value={copyFromFile}
                onValueChange={setCopyFromFile}
              >
                <SelectTrigger id="copyFrom">
                  <SelectValue />
                </SelectTrigger>
                <SelectContent>
                  {getCopySourceFiles().map((file) => (
                    <SelectItem key={file} value={file}>
                      {file}
                      {file === DEFAULT_EXAMPLE_FILE && (
                        <span className="ml-1 text-xs opacity-60">({t('reportTypes.exampleTemplate')})</span>
                      )}
                    </SelectItem>
                  ))}
                </SelectContent>
              </Select>
            </div>
          </div>
          <DialogFooter>
            <Button variant="outline" onClick={() => setShowNewReportTypeDialog(false)}>
              {t('common.cancel')}
            </Button>
            <Button
              onClick={handleCreateNewReportType}
              disabled={!newReportTypeName.trim() || createFileMutation.isPending}
            >
              {createFileMutation.isPending ? (
                <LoadingSpinner size="sm" className="mr-2" />
              ) : null}
              {t('common.create')}
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>
    </div>
  )
}
