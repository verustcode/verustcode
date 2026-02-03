import { useState, useEffect, useRef } from 'react'
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
import { YamlEditor, type CursorPosition } from '@/components/editor/YamlEditor'
import { RuleOverviewCards } from '@/components/rules/RuleOverviewCards'
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

// Default rule file
const DEFAULT_RULE_FILE = 'default.yaml'
// Default example template file
const DEFAULT_EXAMPLE_FILE = 'default.example.yaml'

/**
 * Rules management page component
 * Supports multiple rule files in config/reviews/ directory
 * Uses YAML editor for editing and read-only cards for overview
 * Supports optimistic locking via hash
 */
export default function Rules() {
  const { t } = useTranslation()
  const queryClient = useQueryClient()
  const [searchParams, setSearchParams] = useSearchParams()

  // Selected file state - support URL query parameter for initial selection
  const [selectedFile, setSelectedFile] = useState<string>(() => {
    const fileParam = searchParams.get('file')
    return fileParam || DEFAULT_RULE_FILE
  })
  
  // Default to YAML mode
  const [mode, setMode] = useState<EditorMode>('yaml')
  const [content, setContent] = useState('')
  const [originalContent, setOriginalContent] = useState('')
  const [hash, setHash] = useState('')
  const [saveResult, setSaveResult] = useState<SaveResult | null>(null)
  // Track current rule from YAML editor cursor position
  const [activeRuleId, setActiveRuleId] = useState<string | null>(null)
  
  // New rule dialog state
  const [showNewRuleDialog, setShowNewRuleDialog] = useState(false)
  const [newRuleName, setNewRuleName] = useState('')
  const [copyFromFile, setCopyFromFile] = useState(DEFAULT_EXAMPLE_FILE)
  
  // Tab scroll ref
  const tabsContainerRef = useRef<HTMLDivElement>(null)
  const [canScrollLeft, setCanScrollLeft] = useState(false)
  const [canScrollRight, setCanScrollRight] = useState(false)

  // Fetch list of rule files
  const { data: rulesListData, isLoading: listLoading } = useQuery({
    queryKey: ['admin', 'rules', 'list'],
    queryFn: () => api.admin.rules.list(),
  })

  const ruleFiles = rulesListData?.files || []

  // Fetch selected rule file content
  const { data: fileContent, isLoading: contentLoading, refetch } = useQuery({
    queryKey: ['admin', 'rules', selectedFile],
    queryFn: () => api.admin.rules.get(selectedFile),
    enabled: !!selectedFile,
  })

  // Handle URL query parameter for file selection (e.g., from Repositories page link)
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
      api.admin.rules.save(selectedFile, content, hash),
    onSuccess: (data) => {
      queryClient.invalidateQueries({ queryKey: ['admin', 'rules'] })
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
      api.admin.rules.create(name, copyFrom),
    onSuccess: (data) => {
      queryClient.invalidateQueries({ queryKey: ['admin', 'rules', 'list'] })
      setShowNewRuleDialog(false)
      setNewRuleName('')
      setCopyFromFile(DEFAULT_EXAMPLE_FILE)
      // Switch to the new file
      if (data.name) {
        setSelectedFile(data.name)
      }
      toast({ title: t('rules.createSuccess') })
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
  }, [ruleFiles])

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

  // Handle cursor position change in YAML editor
  const handleCursorPositionChange = (position: CursorPosition) => {
    setActiveRuleId(position.ruleId || null)
  }

  // Handle mode switch
  const handleModeChange = (newMode: EditorMode) => {
    setMode(newMode)
  }

  // Handle file selection change
  const handleFileChange = (file: string) => {
    if (hasChanges) {
      // Warn user about unsaved changes
      if (!window.confirm(t('rules.unsavedChangesWarning'))) {
        return
      }
    }
    setSelectedFile(file)
    setSaveResult(null)
  }

  // Handle create new rule
  const handleCreateNewRule = () => {
    let fileName = newRuleName.trim()
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
    for (const file of ruleFiles) {
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
                ruleFiles.map((file) => (
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
                    {file.name === DEFAULT_RULE_FILE && (
                      <span className="ml-1 text-xs opacity-60">({t('rules.default')})</span>
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
          
          {/* New rule button */}
          <Button
            size="sm"
            onClick={() => setShowNewRuleDialog(true)}
            className="flex-shrink-0 mb-1"
          >
            <Plus className="mr-1 h-4 w-4" />
            {t('rules.newRule')}
          </Button>
        </div>

        {/* Editor area - use hidden + absolute to avoid overlap animation when switching modes */}
        {contentLoading ? (
          <div className="flex flex-1 items-center justify-center">
            <LoadingSpinner size="lg" />
          </div>
        ) : (
          <div className="relative flex-1">
            {/* YAML Editor - always rendered, hidden when in form mode */}
            <div className={cn(
              'absolute inset-0 overflow-hidden rounded-md border border-[hsl(var(--border))]',
              mode !== 'yaml' && 'hidden'
            )}>
              <YamlEditor
                value={content}
                onChange={setContent}
                onCursorPositionChange={handleCursorPositionChange}
              />
            </div>
            {/* Overview Cards - always rendered, hidden when in yaml mode */}
            <div className={cn('absolute inset-0 overflow-auto', mode !== 'form' && 'hidden')}>
              <RuleOverviewCards content={content} activeRuleId={activeRuleId} />
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
                {t('rules.saveSuccess')}
              </>
            ) : saveResult.isConflict ? (
              <>
                <span className="flex-1">{t('rules.conflictError')}</span>
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
                <span>{t('rules.readOnlyHint')}</span>
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

      {/* New Rule Dialog */}
      <Dialog open={showNewRuleDialog} onOpenChange={setShowNewRuleDialog}>
        <DialogContent>
          <DialogHeader>
            <DialogTitle>{t('rules.newRule')}</DialogTitle>
            <DialogDescription>
              {t('rules.newRuleDescription')}
            </DialogDescription>
          </DialogHeader>
          <div className="grid gap-4 py-4">
            <div className="grid gap-2">
              <Label htmlFor="fileName">{t('rules.fileName')}</Label>
              <Input
                id="fileName"
                value={newRuleName}
                onChange={(e) => setNewRuleName(e.target.value)}
                placeholder="frontend.yaml"
              />
            </div>
            <div className="grid gap-2">
              <Label htmlFor="copyFrom">{t('rules.copyFrom')}</Label>
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
                        <span className="ml-1 text-xs opacity-60">({t('rules.exampleTemplate')})</span>
                      )}
                    </SelectItem>
                  ))}
                </SelectContent>
              </Select>
            </div>
          </div>
          <DialogFooter>
            <Button variant="outline" onClick={() => setShowNewRuleDialog(false)}>
              {t('common.cancel')}
            </Button>
            <Button
              onClick={handleCreateNewRule}
              disabled={!newRuleName.trim() || createFileMutation.isPending}
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
