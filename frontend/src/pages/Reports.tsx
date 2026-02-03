import { useState, useEffect, useRef } from 'react'
import { useTranslation } from 'react-i18next'
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query'
import { Link, useLocation, useSearchParams } from 'react-router-dom'
import { Search, X, Plus, Ban, RefreshCw, ChevronFirst, ChevronLast, ChevronLeft, ChevronRight, Copy, Check, AlertCircle } from 'lucide-react'
import { Input } from '@/components/ui/input'
import { Button } from '@/components/ui/button'
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from '@/components/ui/table'
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
import { Label } from '@/components/ui/label'
import { Skeleton } from '@/components/ui/skeleton'
import { StatusBadge } from '@/components/common/StatusBadge'
import { Combobox } from '@/components/ui/combobox'
import { api, type Report, type ReportType } from '@/lib/api'
import { formatDate, truncate, formatRepoUrl } from '@/lib/utils'

// Report status options
type ReportStatus = 'pending' | 'analyzing' | 'generating' | 'completed' | 'failed' | 'cancelled'
const statusOptions: (ReportStatus | 'all')[] = ['all', 'pending', 'analyzing', 'generating', 'completed', 'failed', 'cancelled']

// Page size options
const PAGE_SIZE_OPTIONS = [10, 20, 50, 100] as const
const PAGE_SIZE_STORAGE_KEY = 'reports_page_size'

const getStoredPageSize = (): number => {
  const stored = localStorage.getItem(PAGE_SIZE_STORAGE_KEY)
  if (stored) {
    const size = parseInt(stored, 10)
    if (PAGE_SIZE_OPTIONS.includes(size as typeof PAGE_SIZE_OPTIONS[number])) {
      return size
    }
  }
  return 20
}

/**
 * Reports list page component
 */
export default function Reports() {
  const { t } = useTranslation()
  const queryClient = useQueryClient()
  const location = useLocation()
  const [searchParams, setSearchParams] = useSearchParams()
  
  // Track if we've processed the URL params to avoid repeated triggers
  const hasProcessedUrlParams = useRef(false)
  // Track if dialog has been opened before (to avoid resetting on initial mount)
  const dialogHasOpened = useRef(false)

  const [page, setPage] = useState(1)
  const [pageSize, setPageSize] = useState(getStoredPageSize)
  const [status, setStatus] = useState<ReportStatus | 'all'>('all')
  const [search, setSearch] = useState('')
  const [showCancelDialog, setShowCancelDialog] = useState(false)
  const [cancelReportId, setCancelReportId] = useState<string | null>(null)
  const [showCreateDialog, setShowCreateDialog] = useState(false)
  const [copiedId, setCopiedId] = useState<string | null>(null)

  // Create report form state
  const [newReportRepoUrl, setNewReportRepoUrl] = useState('')
  const [newReportRef, setNewReportRef] = useState('main')
  const [newReportType, setNewReportType] = useState('')
  const [repoUrlError, setRepoUrlError] = useState('')
  
  // Repositories and branches state
  const [availableRepos, setAvailableRepos] = useState<string[]>([])
  const [availableBranches, setAvailableBranches] = useState<string[]>([])
  const [isLoadingBranches, setIsLoadingBranches] = useState(false)

  // Fetch reports
  const { data, isLoading, refetch } = useQuery({
    queryKey: ['reports', page, pageSize, status, search],
    queryFn: () =>
      api.reports.list({
        page,
        page_size: pageSize,
        status: status === 'all' ? undefined : status,
      }),
    refetchInterval: 10000,
  })

  // Fetch report types
  const { data: reportTypesData } = useQuery({
    queryKey: ['reportTypes'],
    queryFn: () => api.reportTypes.list(),
  })

  // Fetch repositories list
  const { data: reposData } = useQuery({
    queryKey: ['repositories'],
    queryFn: () => api.reports.repositories(),
    enabled: showCreateDialog, // Only fetch when dialog is open
  })

  // Fetch branches when repo URL changes
  const { data: branchesData, refetch: refetchBranches, isFetching: isFetchingBranches } = useQuery({
    queryKey: ['branches', newReportRepoUrl],
    queryFn: () => api.reports.branches(newReportRepoUrl),
    enabled: false, // Manual fetch
  })

  // Handle branches data update
  useEffect(() => {
    if (branchesData?.data) {
      setAvailableBranches(branchesData.data)
      setIsLoadingBranches(false)
      // Auto-select first branch if available
      if (branchesData.data.length > 0 && !newReportRef) {
        setNewReportRef(branchesData.data[0])
      }
    }
  }, [branchesData, newReportRef])

  // Handle branches loading state
  useEffect(() => {
    setIsLoadingBranches(isFetchingBranches)
  }, [isFetchingBranches])

  // Auto-refresh when navigating to this page
  useEffect(() => {
    if (location.pathname === '/admin/reports') {
      refetch()
    }
  }, [location.pathname, refetch])

  // Handle repo_url query parameter from repositories page (audit button)
  useEffect(() => {
    const repoUrlParam = searchParams.get('repo_url')
    if (repoUrlParam && !hasProcessedUrlParams.current) {
      hasProcessedUrlParams.current = true
      // Open create dialog and pre-fill the repo URL
      setNewReportRepoUrl(repoUrlParam)
      setShowCreateDialog(true)
      // Clear the query parameter from URL to avoid re-triggering
      setSearchParams({}, { replace: true })
    }
  }, [searchParams, setSearchParams])

  // Set default report type
  useEffect(() => {
    if (reportTypesData?.data && reportTypesData.data.length > 0 && !newReportType) {
      setNewReportType(reportTypesData.data[0].id)
    }
  }, [reportTypesData, newReportType])

  // Update available repos when data is fetched
  useEffect(() => {
    if (reposData?.data) {
      setAvailableRepos(reposData.data)
    }
  }, [reposData])

  // Fetch branches when repo URL changes
  useEffect(() => {
    if (newReportRepoUrl && showCreateDialog) {
      // Validate URL format before fetching branches
      const isValid = validateRepoUrl(newReportRepoUrl)
      if (isValid) {
        setIsLoadingBranches(true)
        refetchBranches()
      } else {
        setAvailableBranches([])
        setIsLoadingBranches(false)
      }
    } else {
      setAvailableBranches([])
      setIsLoadingBranches(false)
    }
  }, [newReportRepoUrl, showCreateDialog, refetchBranches])

  // Reset form when dialog closes (but not on initial mount)
  useEffect(() => {
    if (showCreateDialog) {
      // Mark that dialog has been opened
      dialogHasOpened.current = true
    } else if (dialogHasOpened.current) {
      // Only reset when dialog closes after being opened (not on initial mount)
      setNewReportRepoUrl('')
      setNewReportRef('main')
      setAvailableBranches([])
      setIsLoadingBranches(false)
      setRepoUrlError('')
      // Reset the URL params flag so next navigation with repo_url will work
      hasProcessedUrlParams.current = false
    }
  }, [showCreateDialog])

  // Validate repository URL
  const validateRepoUrl = (url: string): boolean => {
    if (!url) {
      setRepoUrlError('')
      return false
    }

    // Common Git repository URL patterns
    const patterns = [
      // HTTPS: https://github.com/owner/repo or https://gitlab.com/owner/repo
      /^https?:\/\/[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}\/[\w-]+\/[\w.-]+\/?$/,
      // SSH: git@github.com:owner/repo.git
      /^git@[a-zA-Z0-9.-]+:[a-zA-Z0-9._/-]+\.git$/,
      // HTTPS with .git: https://github.com/owner/repo.git
      /^https?:\/\/[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}\/[\w-]+\/[\w.-]+\.git$/,
    ]

    const isValid = patterns.some(pattern => pattern.test(url))
    
    if (!isValid) {
      setRepoUrlError(t('reports.invalidRepoUrl'))
    } else {
      setRepoUrlError('')
    }
    
    return isValid
  }

  // Handle repository URL change with validation
  const handleRepoUrlChange = (value: string) => {
    setNewReportRepoUrl(value)
    setNewReportRef('') // Reset branch when repo changes
    validateRepoUrl(value)
  }

  // Cancel report mutation
  const cancelMutation = useMutation({
    mutationFn: (id: string) => api.reports.cancel(id),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['reports'] })
    },
  })

  // Create report mutation
  const createMutation = useMutation({
    mutationFn: (data: { repo_url: string; ref: string; report_type: string }) =>
      api.reports.create(data),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['reports'] })
      setShowCreateDialog(false)
      setNewReportRepoUrl('')
      setNewReportRef('main')
    },
  })

  const reports = (data?.data || []) as Report[]
  const total = data?.total || 0
  const totalPages = Math.ceil(total / pageSize)
  const reportTypes = (reportTypesData?.data || []) as ReportType[]

  // Filter by search locally
  const filteredReports = search
    ? reports.filter((r) => r.repo_url.toLowerCase().includes(search.toLowerCase()))
    : reports

  const handleCancel = (id: string) => {
    setCancelReportId(id)
    setShowCancelDialog(true)
  }

  const handleCancelConfirm = () => {
    if (cancelReportId !== null) {
      cancelMutation.mutate(cancelReportId)
    }
    setShowCancelDialog(false)
    setCancelReportId(null)
  }

  const handleCopyReportId = async (id: string) => {
    try {
      await navigator.clipboard.writeText(id)
      setCopiedId(id)
      setTimeout(() => setCopiedId(null), 2000)
    } catch (err) {
      console.error('Failed to copy report ID:', err)
    }
  }

  const handleCreateReport = () => {
    if (!newReportRepoUrl || !newReportRef || !newReportType) return
    createMutation.mutate({
      repo_url: newReportRepoUrl,
      ref: newReportRef,
      report_type: newReportType,
    })
  }

  // Get report type name directly from API response (config/reports/*.yaml)
  const getReportTypeName = (typeId: string) => {
    const type = reportTypes.find((t) => t.id === typeId)
    return type?.name || typeId
  }

  // Get report type description directly from API response (config/reports/*.yaml)
  const getReportTypeDescription = (typeId: string) => {
    const type = reportTypes.find((t) => t.id === typeId)
    return type?.description || ''
  }

  return (
    <div className="space-y-8">
      {/* Filters and Actions */}
      <div className="flex flex-col gap-6 sm:flex-row sm:items-center">
        {/* Search */}
        <div className="relative flex-1">
          <Search className="absolute left-3 top-1/2 h-4 w-4 -translate-y-1/2 text-[hsl(var(--muted-foreground))]" />
          <Input
            placeholder={t('reports.searchPlaceholder')}
            value={search}
            onChange={(e) => setSearch(e.target.value)}
            className="pl-9"
          />
          {search && (
            <Button
              variant="ghost"
              size="icon"
              className="absolute right-1 top-1/2 h-7 w-7 -translate-y-1/2"
              onClick={() => setSearch('')}
            >
              <X className="h-4 w-4" />
            </Button>
          )}
        </div>

        {/* Status filter */}
        <Select value={status} onValueChange={(v) => setStatus(v as ReportStatus | 'all')}>
          <SelectTrigger className="w-[180px]">
            <SelectValue placeholder={t('reports.filterByStatus')} />
          </SelectTrigger>
          <SelectContent>
            {statusOptions.map((s) => (
              <SelectItem key={s} value={s}>
                {s === 'all' ? t('common.all') : t(`reports.status.${s}`, s)}
              </SelectItem>
            ))}
          </SelectContent>
        </Select>

        {/* Refresh button */}
        <Button variant="outline" size="sm" onClick={() => refetch()}>
          <RefreshCw className="mr-2 h-4 w-4" />
          {t('common.refresh')}
        </Button>

        {/* Create button */}
        <Button size="sm" onClick={() => setShowCreateDialog(true)}>
          <Plus className="mr-2 h-4 w-4" />
          {t('reports.create')}
        </Button>

      </div>

      {/* Table */}
      <div className="rounded-md border border-[hsl(var(--border))]">
        <Table>
          <TableHeader>
            <TableRow>
              <TableHead className="w-[150px]">{t('reports.id')}</TableHead>
              <TableHead className="w-[300px]">{t('reports.repository')}</TableHead>
              <TableHead className="w-[100px]">{t('reports.ref')}</TableHead>
              <TableHead className="w-[140px]">{t('reports.type')}</TableHead>
              <TableHead className="w-[100px]">{t('common.status')}</TableHead>
              <TableHead className="w-[160px]">{t('reports.createdAt')}</TableHead>
              <TableHead className="w-[100px]">{t('reports.actions')}</TableHead>
            </TableRow>
          </TableHeader>
          <TableBody>
            {isLoading ? (
              [...Array(5)].map((_, i) => (
                <TableRow key={i}>
                  {[...Array(7)].map((_, j) => (
                    <TableCell key={j}>
                      <Skeleton className="h-5 w-full" />
                    </TableCell>
                  ))}
                </TableRow>
              ))
            ) : filteredReports.length === 0 ? (
              <TableRow>
                <TableCell colSpan={7} className="h-24">
                  <div className="flex flex-col items-center justify-center py-8 text-[hsl(var(--muted-foreground))]">
                    <AlertCircle className="mb-2 h-8 w-8" />
                    <p>{t('reports.noReports')}</p>
                  </div>
                </TableCell>
              </TableRow>
            ) : (
              filteredReports.map((report) => (
                <TableRow key={report.id}>
                  <TableCell className="font-mono text-xs">
                    <div className="flex items-center gap-2">
                      <span className="text-[hsl(var(--muted-foreground))]">{report.id}</span>
                      <button
                        onClick={(e) => {
                          e.stopPropagation()
                          handleCopyReportId(report.id)
                        }}
                        className="p-1 rounded hover:bg-[hsl(var(--muted))] transition-colors"
                        title={t('common.copy')}
                      >
                        {copiedId === report.id ? (
                          <Check className="h-3 w-3 text-green-500" />
                        ) : (
                          <Copy className="h-3 w-3 text-[hsl(var(--muted-foreground))] opacity-50 hover:opacity-100" />
                        )}
                      </button>
                    </div>
                  </TableCell>
                  <TableCell>
                    <span title={report.repo_url}>
                      {truncate(formatRepoUrl(report.repo_url), 40)}
                    </span>
                  </TableCell>
                  <TableCell className="text-[hsl(var(--muted-foreground))]">
                    {report.ref}
                  </TableCell>
                  <TableCell>
                    <span className="inline-flex items-center px-2 py-0.5 rounded-md text-xs font-medium bg-[hsl(var(--muted))] text-[hsl(var(--foreground))]" title={getReportTypeName(report.report_type)}>
                      {truncate(getReportTypeName(report.report_type), 13)}
                    </span>
                  </TableCell>
                  <TableCell>
                    <StatusBadge status={report.status} />
                  </TableCell>
                  <TableCell className="text-[hsl(var(--muted-foreground))]">
                    {formatDate(report.created_at)}
                  </TableCell>
                  <TableCell>
                    <div className="flex items-center gap-1">
                      <Button variant="ghost" size="sm" asChild>
                        <Link to={`/admin/reports/${report.id}`}>
                          {t('reports.viewDetails')}
                        </Link>
                      </Button>
                      {(report.status === 'pending' || report.status === 'analyzing' || report.status === 'generating') && (
                        <Button
                          variant="ghost"
                          size="icon"
                          className="h-8 w-8 text-[hsl(var(--destructive))]"
                          onClick={() => handleCancel(report.id)}
                          disabled={cancelMutation.isPending}
                        >
                          <Ban className="h-4 w-4" />
                        </Button>
                      )}
                    </div>
                  </TableCell>
                </TableRow>
              ))
            )}
          </TableBody>
        </Table>
      </div>

      {/* Pagination */}
      <div className="flex items-center justify-between">
        <div className="flex items-center gap-4">
          <span className="text-sm font-medium text-[hsl(var(--muted-foreground))]">
            {t('common.total')}: {total}
          </span>
          <Select
            value={String(pageSize)}
            onValueChange={(v) => {
              const size = parseInt(v, 10)
              setPageSize(size)
              localStorage.setItem(PAGE_SIZE_STORAGE_KEY, v)
              setPage(1)
            }}
          >
            <SelectTrigger className="w-[100px] h-8">
              <SelectValue />
            </SelectTrigger>
            <SelectContent>
              {PAGE_SIZE_OPTIONS.map((size) => (
                <SelectItem key={size} value={String(size)}>
                  {size} / {t('common.perPage')}
                </SelectItem>
              ))}
            </SelectContent>
          </Select>
        </div>
        {totalPages > 1 && (
          <div className="flex items-center gap-1">
            <Button
              variant="outline"
              size="icon"
              className="h-8 w-8"
              onClick={() => setPage(1)}
              disabled={page === 1}
            >
              <ChevronFirst className="h-4 w-4" />
            </Button>
            <Button
              variant="outline"
              size="icon"
              className="h-8 w-8"
              onClick={() => setPage((p) => Math.max(1, p - 1))}
              disabled={page === 1}
            >
              <ChevronLeft className="h-4 w-4" />
            </Button>
            <span className="text-sm min-w-[60px] text-center">
              {page} / {totalPages}
            </span>
            <Button
              variant="outline"
              size="icon"
              className="h-8 w-8"
              onClick={() => setPage((p) => Math.min(totalPages, p + 1))}
              disabled={page === totalPages}
            >
              <ChevronRight className="h-4 w-4" />
            </Button>
            <Button
              variant="outline"
              size="icon"
              className="h-8 w-8"
              onClick={() => setPage(totalPages)}
              disabled={page === totalPages}
            >
              <ChevronLast className="h-4 w-4" />
            </Button>
          </div>
        )}
      </div>

      {/* Cancel confirmation dialog */}
      <Dialog open={showCancelDialog} onOpenChange={setShowCancelDialog}>
        <DialogContent>
          <DialogHeader>
            <DialogTitle>{t('reports.cancelReport')}</DialogTitle>
            <DialogDescription>
              {t('reports.cancelConfirm')}
            </DialogDescription>
          </DialogHeader>
          <DialogFooter>
            <Button variant="outline" onClick={() => setShowCancelDialog(false)}>
              {t('common.cancel')}
            </Button>
            <Button
              variant="destructive"
              onClick={handleCancelConfirm}
              disabled={cancelMutation.isPending}
            >
              {t('reports.cancelReport')}
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>

      {/* Create report dialog */}
      <Dialog open={showCreateDialog} onOpenChange={setShowCreateDialog}>
        <DialogContent className="sm:max-w-[800px]">
          <DialogHeader>
            <DialogTitle>{t('reports.createReport')}</DialogTitle>
            <DialogDescription>
              {t('reports.createDescription')}
            </DialogDescription>
          </DialogHeader>
          <div className="grid gap-4 py-6">
            {/* Repository URL */}
            <div className="grid gap-2">
              <Label htmlFor="repo_url">{t('reports.repoUrl')}</Label>
              <Combobox
                options={availableRepos.map((repo) => ({
                  value: repo,
                  label: repo,
                }))}
                value={newReportRepoUrl}
                onValueChange={handleRepoUrlChange}
                placeholder="https://github.com/owner/repo"
                allowCustomValue={true}
              />
              {repoUrlError && (
                <p className="text-xs text-[hsl(var(--destructive))]">
                  {repoUrlError}
                </p>
              )}
            </div>
            {/* Branch */}
            <div className="grid gap-2">
              <Label htmlFor="ref">{t('reports.branch')}</Label>
              <Select
                value={newReportRef}
                onValueChange={setNewReportRef}
                disabled={isLoadingBranches || !newReportRepoUrl || availableBranches.length === 0}
              >
                <SelectTrigger>
                  <SelectValue
                    placeholder={
                      isLoadingBranches
                        ? t('common.loading')
                        : !newReportRepoUrl
                        ? t('reports.selectRepoFirst')
                        : availableBranches.length === 0
                        ? t('reports.noBranches')
                        : t('reports.selectBranch')
                    }
                  />
                </SelectTrigger>
                <SelectContent className="w-[var(--radix-select-trigger-width)]">
                  {availableBranches.map((branch) => (
                    <SelectItem key={branch} value={branch}>
                      {branch}
                    </SelectItem>
                  ))}
                </SelectContent>
              </Select>
            </div>
            <div className="grid gap-2">
              <Label>{t('reports.reportType')}</Label>
              <Select value={newReportType} onValueChange={setNewReportType}>
                <SelectTrigger className="h-auto min-h-[60px] py-3 [&>span]:line-clamp-2 [&>span]:whitespace-normal [&>span]:text-left">
                  <SelectValue placeholder={t('reports.selectType')} />
                </SelectTrigger>
                <SelectContent side="bottom" className="w-[var(--radix-select-trigger-width)] max-h-[300px] overflow-y-auto">
                  {reportTypes.map((type) => (
                    <SelectItem key={type.id} value={type.id} className="py-1.5">
                      <div className="flex flex-col items-start gap-1 w-full pr-6">
                        <span className="font-medium break-words w-full">
                          {getReportTypeName(type.id)}
                        </span>
                        <span className="text-xs text-[hsl(var(--muted-foreground))] line-clamp-1 break-words">
                          {getReportTypeDescription(type.id)}
                        </span>
                      </div>
                    </SelectItem>
                  ))}
                </SelectContent>
              </Select>
            </div>
          </div>
          <DialogFooter>
            <Button variant="outline" onClick={() => setShowCreateDialog(false)}>
              {t('common.cancel')}
            </Button>
            <Button
              onClick={handleCreateReport}
              disabled={createMutation.isPending || !newReportRepoUrl || !newReportRef || !newReportType || !!repoUrlError}
            >
              {createMutation.isPending ? t('common.loading') : t('common.create')}
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>

    </div>
  )
}

