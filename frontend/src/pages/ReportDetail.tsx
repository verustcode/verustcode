import { useState, useEffect, useRef, useCallback } from 'react'
import { useTranslation } from 'react-i18next'
import { useParams, useNavigate } from 'react-router-dom'
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query'
import { ArrowLeft, Download, RefreshCw, Repeat, FileText, CheckCircle2, XCircle, Loader2, Copy, Check, FileCode, FileType, FileOutput } from 'lucide-react'
import { Button } from '@/components/ui/button'
import { Card, CardContent, CardDescription, CardHeader } from '@/components/ui/card'
import { Skeleton } from '@/components/ui/skeleton'
import { StatusBadge } from '@/components/common/StatusBadge'
import { LogViewer } from '@/components/common/LogViewer'
import { MarkdownRenderer } from '@/components/common/MarkdownRenderer'
import { CodeScanAnimation } from '@/components/common/CodeScanAnimation'
import { api, type ReportSection } from '@/lib/api'
import { formatDuration, cn, formatRepoUrl } from '@/lib/utils'
import { useExport } from '@/hooks/useExport'

/**
 * SVG Icons for sidebar navigation (inline SVG for consistency with exported HTML)
 * These are Lucide icons as inline SVG strings
 */
const SidebarIcons = {
  chevronRight: `<svg xmlns="http://www.w3.org/2000/svg" width="12" height="12" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="m9 18 6-6-6-6"/></svg>`,
  chevronDown: `<svg xmlns="http://www.w3.org/2000/svg" width="12" height="12" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="m6 9 6 6 6-6"/></svg>`,
  chevronsUpDown: `<svg xmlns="http://www.w3.org/2000/svg" width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="m7 15 5 5 5-5"/><path d="m7 9 5-5 5 5"/></svg>`
}

/**
 * SectionNode extends ReportSection with children for tree structure
 * Used to render hierarchical outline in the left sidebar
 */
interface SectionNode extends ReportSection {
  children: SectionNode[]
}

/**
 * Build a tree structure from flat sections array
 * Groups child sections under their parent based on parent_section_id
 * @param sections - Flat array of ReportSection
 * @returns Array of root SectionNodes with children attached
 */
function buildSectionTree(sections: ReportSection[]): SectionNode[] {
  // Sort by section_index to maintain order
  const sorted = [...sections].sort((a, b) => a.section_index - b.section_index)
  
  // Separate parent nodes (no parent_section_id) and child nodes
  const parentNodes: SectionNode[] = []
  const childMap = new Map<string, SectionNode[]>()
  
  for (const s of sorted) {
    const node: SectionNode = { ...s, children: [] }
    if (s.parent_section_id) {
      // Child section - group by parent_section_id
      const children = childMap.get(s.parent_section_id) || []
      children.push(node)
      childMap.set(s.parent_section_id, children)
    } else {
      // Parent section (no parent_section_id)
      parentNodes.push(node)
    }
  }
  
  // Attach children to their parent nodes
  for (const parent of parentNodes) {
    parent.children = childMap.get(parent.section_id) || []
  }
  
  return parentNodes
}

/**
 * Get the first leaf section from the tree (for auto-selection)
 * Returns the first clickable section: either a leaf parent or first child
 */
function getFirstLeafSection(tree: SectionNode[]): SectionNode | null {
  if (tree.length === 0) return null
  
  const first = tree[0]
  // If parent is a leaf (no children), it's clickable
  if (first.is_leaf || first.children.length === 0) {
    return first
  }
  // Otherwise return first child
  return first.children[0] || null
}

/**
 * Report detail page component
 * Uses cached rendering strategy: visited sections are kept in DOM and toggled via CSS
 */
export default function ReportDetail() {
  const { t } = useTranslation()
  const { id } = useParams<{ id: string }>()
  const navigate = useNavigate()
  const queryClient = useQueryClient()
  const [copiedId, setCopiedId] = useState(false)
  const [selectedSectionId, setSelectedSectionId] = useState<number | null>(null)
  // Track visited sections to cache their rendered content
  const [visitedSectionIds, setVisitedSectionIds] = useState<Set<number>>(new Set())
  // Track expanded parent sections (for collapsible sidebar)
  const [expandedParents, setExpandedParents] = useState<Set<string>>(new Set())
  const contentRef = useRef<HTMLDivElement>(null)
  // Export dropdown state (custom popover to prevent accidental clicks)
  const [exportMenuOpen, setExportMenuOpen] = useState(false)
  const exportMenuRef = useRef<HTMLDivElement>(null)
  const exportButtonRef = useRef<HTMLButtonElement>(null)
  // Global export state - allows background downloads when navigating away
  const { exportingReportId, exportingType, startExport } = useExport()
  // Check if this specific report is being exported
  const isExportingThisReport = exportingReportId === id
  const currentExportType = exportingType

  // Handle section click - add to visited set and select
  const handleSectionClick = useCallback((sectionId: number) => {
    setSelectedSectionId(sectionId)
    // Add to visited sections for caching
    setVisitedSectionIds(prev => {
      if (prev.has(sectionId)) return prev
      const next = new Set(prev)
      next.add(sectionId)
      return next
    })
  }, [])

  // Toggle expand/collapse for a parent section
  const toggleParent = useCallback((sectionId: string) => {
    setExpandedParents(prev => {
      const next = new Set(prev)
      if (next.has(sectionId)) {
        next.delete(sectionId)
      } else {
        next.add(sectionId)
      }
      return next
    })
  }, [])

  // Fetch report
  const { data: report, isLoading, refetch } = useQuery({
    queryKey: ['report', id],
    queryFn: () => api.reports.get(id!),
    enabled: !!id,
    refetchInterval: (query) => {
      // Keep polling if report is still in progress
      const status = query.state.data?.status
      if (status === 'pending' || status === 'analyzing' || status === 'generating') {
        return 3000 // Poll every 3 seconds
      }
      return false
    },
  })

  // Build section tree for hierarchical rendering
  const sectionTree = report?.sections ? buildSectionTree(report.sections) : []

  // Toggle all parent sections expand/collapse
  const toggleAllParents = useCallback(() => {
    setExpandedParents(prev => {
      // If any parent is expanded, collapse all; otherwise expand all
      if (prev.size > 0) {
        return new Set()
      } else {
        // Get all parent section IDs that have children
        const parentIds = sectionTree
          .filter(p => p.children.length > 0)
          .map(p => p.section_id)
        return new Set(parentIds)
      }
    })
  }, [sectionTree])

  // Auto-select first leaf section when sections are loaded
  useEffect(() => {
    if (report?.sections?.length && selectedSectionId === null) {
      const tree = buildSectionTree(report.sections)
      const firstLeaf = getFirstLeafSection(tree)
      if (firstLeaf) {
        setSelectedSectionId(firstLeaf.id)
        // Also add first section to visited set
        setVisitedSectionIds(new Set([firstLeaf.id]))
      }
    }
  }, [report?.sections, selectedSectionId])

  // Scroll content area to top when section changes
  useEffect(() => {
    if (contentRef.current) {
      contentRef.current.scrollTop = 0
    }
  }, [selectedSectionId])

  // Close export menu when clicking outside
  useEffect(() => {
    if (!exportMenuOpen) return

    const handleClickOutside = (event: MouseEvent) => {
      if (
        exportMenuRef.current &&
        exportButtonRef.current &&
        !exportMenuRef.current.contains(event.target as Node) &&
        !exportButtonRef.current.contains(event.target as Node)
      ) {
        setExportMenuOpen(false)
      }
    }

    document.addEventListener('mousedown', handleClickOutside)
    return () => document.removeEventListener('mousedown', handleClickOutside)
  }, [exportMenuOpen])

  // Retry mutation
  const retryMutation = useMutation({
    mutationFn: () => api.reports.retry(id!),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['report', id] })
    },
  })

  const handleCopyId = async () => {
    if (!id) return
    try {
      await navigator.clipboard.writeText(id)
      setCopiedId(true)
      setTimeout(() => setCopiedId(false), 2000)
    } catch (err) {
      console.error('Failed to copy ID:', err)
    }
  }

  /**
   * Handle export to specified format
   * Uses global export context for background downloads
   * @param format - Export format: 'markdown', 'html', or 'pdf'
   */
  const handleExport = (format: 'markdown' | 'html' | 'pdf') => {
    setExportMenuOpen(false)
    if (!id || !report) return
    
    startExport({
      reportId: id,
      reportTitle: report.title || id,
      exportType: format,
    })
  }


  if (isLoading) {
    return (
      <div className="space-y-6">
        <Skeleton className="h-8 w-64" />
        <Skeleton className="h-[600px] w-full" />
      </div>
    )
  }

  if (!report) {
    return (
      <div className="flex items-center justify-center h-64">
        <p className="text-[hsl(var(--muted-foreground))]">
          {t('reports.notFound')}
        </p>
      </div>
    )
  }

  const isInProgress = ['pending', 'analyzing', 'generating'].includes(report.status)

  return (
    <div className="space-y-6">
      {/* Header */}
      <div className="flex items-center justify-between">
        <div className="flex items-center gap-4">
          <Button variant="ghost" size="icon" onClick={() => navigate('/admin/reports')}>
            <ArrowLeft className="h-5 w-5" />
          </Button>
          <div>
            <div className="flex items-center gap-3">
              <h1 className="text-2xl font-bold">
                {report.title || t('reports.untitled')}
              </h1>
              <StatusBadge status={report.status} />
            </div>
            <div className="flex items-center gap-2 mt-1 text-sm text-[hsl(var(--muted-foreground))]">
              <span className="font-mono">{report.id}</span>
              <button
                onClick={handleCopyId}
                className="p-1 rounded hover:bg-[hsl(var(--muted))] transition-colors"
              >
                {copiedId ? (
                  <Check className="h-3 w-3 text-green-500" />
                ) : (
                  <Copy className="h-3 w-3 opacity-50 hover:opacity-100" />
                )}
              </button>
            </div>
          </div>
        </div>
        <div className="flex items-center gap-2">
          <LogViewer taskType="report" taskId={report.id} />
          <Button variant="outline" size="sm" onClick={() => refetch()}>
            <RefreshCw className="mr-2 h-4 w-4" />
            {t('common.refresh')}
          </Button>
          {report.status === 'failed' && (
            <Button
              variant="ghost"
              size="sm"
              onClick={() => retryMutation.mutate()}
              disabled={retryMutation.isPending}
              className="hover:bg-transparent hover:text-[hsl(var(--primary))]"
            >
              <Repeat className="mr-2 h-4 w-4 text-amber-500" />
              {retryMutation.isPending ? t('reports.retry.retrying') : t('reports.retry.button')}
            </Button>
          )}
          {report.status === 'completed' && report.content && (
            <div className="relative">
                <Button
                  ref={exportButtonRef}
                  size="sm"
                  type="button"
                  disabled={isExportingThisReport}
                  onClick={(e) => {
                    e.stopPropagation()
                    setExportMenuOpen(!exportMenuOpen)
                  }}
                >
                  {isExportingThisReport ? (
                    <>
                      <Loader2 className="mr-2 h-4 w-4 animate-spin" />
                      {currentExportType === 'pdf' ? t('reports.exportingPdf') : t('reports.exporting')}
                    </>
                  ) : (
                    <>
                      <Download className="mr-2 h-4 w-4" />
                      {t('reports.export')}
                    </>
                  )}
                </Button>

              {exportMenuOpen && !isExportingThisReport && (
                <div
                  ref={exportMenuRef}
                  className={cn(
                    'absolute right-0 top-full z-50 mt-2 w-44 overflow-hidden rounded-md border border-[hsl(var(--border))] bg-[hsl(var(--popover))] p-1 text-[hsl(var(--popover-foreground))] shadow-md'
                  )}
                >
                  <button
                    type="button"
                    onClick={(e) => {
                      e.preventDefault()
                      e.stopPropagation()
                      handleExport('markdown')
                    }}
                    className="relative flex w-full cursor-default select-none items-center rounded-sm px-2 py-1.5 text-sm outline-none transition-colors hover:bg-[hsl(var(--accent))] focus:bg-[hsl(var(--accent))]"
                  >
                    <FileType className="mr-2 h-4 w-4" />
                    Markdown (.md)
                  </button>
                  <button
                    type="button"
                    onClick={(e) => {
                      e.preventDefault()
                      e.stopPropagation()
                      handleExport('html')
                    }}
                    className="relative flex w-full cursor-default select-none items-center rounded-sm px-2 py-1.5 text-sm outline-none transition-colors hover:bg-[hsl(var(--accent))] focus:bg-[hsl(var(--accent))]"
                  >
                    <FileCode className="mr-2 h-4 w-4" />
                    HTML (.html)
                  </button>
                  <button
                    type="button"
                    onClick={(e) => {
                      e.preventDefault()
                      e.stopPropagation()
                      handleExport('pdf')
                    }}
                    className="relative flex w-full cursor-default select-none items-center rounded-sm px-2 py-1.5 text-sm outline-none transition-colors hover:bg-[hsl(var(--accent))] focus:bg-[hsl(var(--accent))]"
                  >
                    <FileOutput className="mr-2 h-4 w-4" />
                    PDF (.pdf)
                  </button>
                </div>
              )}
            </div>
          )}
        </div>
      </div>

      {/* Info Cards */}
      <div className="grid gap-4 md:grid-cols-4">
        <Card>
          <CardHeader className="pb-1 pt-3">
            <CardDescription>{t('reports.repository')}</CardDescription>
          </CardHeader>
          <CardContent className="pb-3">
            <p className="text-sm font-medium truncate" title={report.repo_url}>
              {formatRepoUrl(report.repo_url)}
            </p>
          </CardContent>
        </Card>
        <Card>
          <CardHeader className="pb-1 pt-3">
            <CardDescription>{t('reports.ref')}</CardDescription>
          </CardHeader>
          <CardContent className="pb-3">
            <p className="text-sm font-medium">{report.ref}</p>
          </CardContent>
        </Card>
        <Card>
          <CardHeader className="pb-1 pt-3">
            <CardDescription>{t('reports.type')}</CardDescription>
          </CardHeader>
          <CardContent className="pb-3">
            <div className="flex items-center gap-2">
              <FileText className="h-4 w-4" />
              <span className="text-sm font-medium">{report.report_type}</span>
            </div>
          </CardContent>
        </Card>
        <Card>
          <CardHeader className="pb-1 pt-3">
            <CardDescription>{t('reports.duration')}</CardDescription>
          </CardHeader>
          <CardContent className="pb-3">
            <p className="text-sm font-medium">
              {report.duration ? formatDuration(report.duration) : '-'}
            </p>
          </CardContent>
        </Card>
      </div>

      {/* Main Content - Split Layout */}
      {report.sections && report.sections.length > 0 ? (
        <Card className="overflow-hidden">
          <div className="flex h-[calc(100vh-400px)] min-h-[500px]">
            {/* Left sidebar - Section list */}
            <div className="w-64 flex-shrink-0 border-r border-[hsl(var(--border))] overflow-hidden flex flex-col">
              <div className="py-3 px-4 border-b border-[hsl(var(--border))] bg-[hsl(var(--muted))] flex items-center justify-between">
                <div className="flex items-center gap-2">
                  <h3 className="text-sm font-medium">
                    {t('reports.sections')}
                  </h3>
                  {/* Progress indicator when generating */}
                  {isInProgress && report.total_sections > 0 && (
                    <span className="text-xs text-[hsl(var(--muted-foreground))] flex items-center gap-1">
                      <Loader2 className="h-3 w-3 animate-spin" />
                      {report.current_section}/{report.total_sections}
                    </span>
                  )}
                </div>
                {/* Toggle all expand/collapse button */}
                {sectionTree.some(p => p.children.length > 0) && (
                  <button
                    onClick={toggleAllParents}
                    className="p-1 rounded hover:bg-[hsl(var(--background))] transition-colors text-[hsl(var(--muted-foreground))]"
                    title={expandedParents.size > 0 ? t('reports.collapseAll') : t('reports.expandAll')}
                    dangerouslySetInnerHTML={{ __html: SidebarIcons.chevronsUpDown }}
                  />
                )}
              </div>
              <div className="overflow-y-auto flex-1">
                {sectionTree.map((parentNode) => {
                  const hasChildren = parentNode.children.length > 0
                  const isExpanded = expandedParents.has(parentNode.section_id)
                  
                  return (
                    <div key={parentNode.id}>
                      {/* Parent section: if it has children, render as collapsible group header; otherwise as clickable item */}
                      {hasChildren ? (
                        // Parent with children - render as collapsible group header
                        <button
                          onClick={() => toggleParent(parentNode.section_id)}
                          className="w-full px-4 py-2 text-sm font-semibold text-[hsl(var(--foreground))] uppercase tracking-wide bg-[hsl(var(--muted)/0.5)] border-b border-[hsl(var(--border))] flex items-center gap-2 hover:bg-[hsl(var(--muted))] transition-colors"
                        >
                          <span 
                            className="flex-shrink-0"
                            dangerouslySetInnerHTML={{ __html: isExpanded ? SidebarIcons.chevronDown : SidebarIcons.chevronRight }}
                          />
                          <span className="truncate">{parentNode.title}</span>
                        </button>
                      ) : (
                        // Parent without children (leaf parent) - render as clickable item
                        <button
                          onClick={() => handleSectionClick(parentNode.id)}
                          className={cn(
                            'w-full flex items-center gap-2 px-4 py-3 text-left text-sm transition-colors border-b border-[hsl(var(--border))]',
                            selectedSectionId === parentNode.id
                              ? 'bg-[hsl(var(--accent))] text-[hsl(var(--accent-foreground))]'
                              : 'hover:bg-[hsl(var(--muted))]'
                          )}
                        >
                          <SectionStatusIcon status={parentNode.status} />
                          <span className="truncate flex-1">{parentNode.title}</span>
                        </button>
                      )}
                      {/* Child sections - render with indent, only visible when expanded */}
                      {hasChildren && isExpanded && parentNode.children.map((child) => (
                        <button
                          key={child.id}
                          onClick={() => handleSectionClick(child.id)}
                          className={cn(
                            'w-full flex items-center gap-2 pl-8 pr-4 py-2.5 text-left text-sm transition-colors border-b border-[hsl(var(--border))] last:border-b-0',
                            selectedSectionId === child.id
                              ? 'bg-[hsl(var(--accent))] text-[hsl(var(--accent-foreground))]'
                              : 'hover:bg-[hsl(var(--muted))]'
                          )}
                        >
                          <SectionStatusIcon status={child.status} />
                          <span className="truncate flex-1">{child.title}</span>
                        </button>
                      ))}
                    </div>
                  )
                })}
              </div>
            </div>

            {/* Right content area - renders all visited sections, toggles visibility via CSS */}
            <div ref={contentRef} className="flex-1 overflow-y-auto p-6 relative">
              {/* Render all visited sections, keep them in DOM for instant switching */}
              {report.sections?.filter(s => visitedSectionIds.has(s.id)).map(section => (
                <CachedSectionContent
                  key={section.id}
                  section={section}
                  isVisible={section.id === selectedSectionId}
                  t={t}
                />
              ))}
              {/* Show placeholder if no section is selected */}
              {selectedSectionId === null && (
                <div className="text-center py-12 text-[hsl(var(--muted-foreground))]">
                  {t('reports.selectSection')}
                </div>
              )}
            </div>
          </div>
        </Card>
      ) : (
        <Card>
          <CardContent className="pt-6">
            <div className="flex flex-col items-center justify-center py-12 gap-4">
              {isInProgress ? (
                <>
                  <CodeScanAnimation size="md" />
                  <span className="text-[hsl(var(--muted-foreground))]">
                    {t('reports.analyzingStructure')}
                  </span>
                </>
              ) : (
                <span className="text-[hsl(var(--muted-foreground))]">
                  {t('reports.noSections')}
                </span>
              )}
            </div>
          </CardContent>
        </Card>
      )}
    </div>
  )
}

function SectionStatusIcon({ status }: { status: string }) {
  switch (status) {
    case 'completed':
      return <CheckCircle2 className="h-3 w-3 text-green-500" />
    case 'running':
      return <Loader2 className="h-3 w-3 text-blue-500 animate-spin" />
    case 'failed':
      return <XCircle className="h-3 w-3 text-red-500" />
    default:
      return <div className="h-3 w-3 rounded-full border border-[hsl(var(--muted-foreground))]" />
  }
}

/**
 * Cached section content component
 * Renders section content and remains in DOM when hidden for instant switching
 * Uses CSS visibility to hide/show instead of unmounting
 */
interface CachedSectionContentProps {
  section: ReportSection
  isVisible: boolean
  t: (key: string) => string
}

function CachedSectionContent({ section, isVisible, t }: CachedSectionContentProps) {
  // Use absolute positioning + visibility to keep all sections in DOM
  // This avoids re-rendering when switching between sections
  const visibilityClass = isVisible
    ? 'block'
    : 'hidden' // Use hidden to completely remove from layout when not visible

  // No content yet - show appropriate status message
  if (!section.content) {
    return (
      <div className={cn('text-center py-12 text-[hsl(var(--muted-foreground))]', visibilityClass)}>
        {section.status === 'running' ? (
          <div className="flex items-center justify-center gap-2">
            <Loader2 className="h-5 w-5 animate-spin" />
            <span>{t('reports.generatingSection')}</span>
          </div>
        ) : section.status === 'failed' ? (
          <div className="text-red-500">
            {section.error_message || t('reports.sectionFailed')}
          </div>
        ) : (
          t('reports.noSectionContent')
        )}
      </div>
    )
  }

  // Render markdown content - stays in DOM once rendered
  // Use consistent styling with ReviewDetail's ReviewContent component
  return (
    <div className={visibilityClass}>
      <MarkdownRenderer
        content={section.content}
        className="p-4 text-sm"
      />
    </div>
  )
}


