import { useState } from 'react'
import { useTranslation } from 'react-i18next'
import { useQuery } from '@tanstack/react-query'
import { Link } from 'react-router-dom'
import {
  RefreshCw,
  ChevronFirst,
  ChevronLast,
  ChevronLeft,
  ChevronRight,
  ExternalLink,
  AlertCircle,
} from 'lucide-react'
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
import { Skeleton } from '@/components/ui/skeleton'
import { Badge } from '@/components/ui/badge'
import { api } from '@/lib/api'
import { formatRelativeTime, truncate, formatRepoUrl } from '@/lib/utils'
import type { FindingItem } from '@/types/finding'

// Severity options for filter
const severityOptions = ['all', 'critical', 'high', 'medium', 'low', 'info'] as const
type SeverityOption = typeof severityOptions[number]

// Severity colors for badges
const severityColors: Record<string, string> = {
  critical: 'bg-red-500/15 text-red-700 dark:text-red-400 border-red-500/30',
  high: 'bg-orange-500/15 text-orange-700 dark:text-orange-400 border-orange-500/30',
  medium: 'bg-yellow-500/15 text-yellow-700 dark:text-yellow-400 border-yellow-500/30',
  low: 'bg-blue-500/15 text-blue-700 dark:text-blue-400 border-blue-500/30',
  info: 'bg-gray-500/15 text-gray-700 dark:text-gray-400 border-gray-500/30',
}

// Page size options and storage key
const PAGE_SIZE_OPTIONS = [10, 20, 50, 100] as const
const PAGE_SIZE_STORAGE_KEY = 'findings_page_size'

// Read page size from localStorage, default to 20
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
 * Findings page component
 * Displays all findings across repositories with filtering and sorting
 */
export default function Findings() {
  const { t } = useTranslation()

  // Pagination state
  const [page, setPage] = useState(1)
  const [pageSize, setPageSize] = useState(getStoredPageSize)

  // Filter state
  const [severityFilter, setSeverityFilter] = useState<SeverityOption>('all')
  const [categoryFilter, setCategoryFilter] = useState<string>('all')
  const [repoFilter, setRepoFilter] = useState<string>('all')

  // Fetch findings
  const { data, isLoading, refetch } = useQuery({
    queryKey: ['findings', page, pageSize, severityFilter, categoryFilter, repoFilter],
    queryFn: () =>
      api.admin.findings.list({
        page,
        page_size: pageSize,
        severity: severityFilter === 'all' ? undefined : severityFilter,
        category: categoryFilter === 'all' ? undefined : categoryFilter,
        repo_url: repoFilter === 'all' ? undefined : repoFilter,
        sort_by: 'severity',
        sort_order: 'desc',
      }),
    refetchInterval: 30000, // Refresh every 30 seconds
  })

  // Fetch all findings to extract unique repos and categories for filters
  const { data: allFindingsData } = useQuery({
    queryKey: ['findings', 'all-for-filters'],
    queryFn: () =>
      api.admin.findings.list({
        page: 1,
        page_size: 1000,
      }),
    staleTime: 60000, // Cache for 1 minute
  })

  // Extract unique repos and categories from all findings
  const uniqueRepos = [...new Set(allFindingsData?.data?.map((f) => f.repo_url) || [])]
  const uniqueCategories = [...new Set(allFindingsData?.data?.map((f) => f.category).filter(Boolean) || [])]

  const findings = data?.data || []
  const total = data?.total || 0
  const totalPages = Math.ceil(total / pageSize)

  // Get severity badge class
  const getSeverityBadgeClass = (severity: string) => {
    return severityColors[severity.toLowerCase()] || severityColors.info
  }

  return (
    <div className="space-y-8">
      {/* Filters */}
      <div className="flex flex-col gap-6 sm:flex-row sm:items-center">
        {/* Repository filter */}
        <div className="flex items-center gap-2">
          <span className="text-sm font-medium text-[hsl(var(--muted-foreground))] whitespace-nowrap">
            {t('findings.repository')}:
          </span>
          <Select value={repoFilter} onValueChange={setRepoFilter}>
            <SelectTrigger className="w-[280px]">
              <SelectValue />
            </SelectTrigger>
            <SelectContent>
              <SelectItem value="all">{t('common.all')}</SelectItem>
              {uniqueRepos.map((repo) => (
                <SelectItem key={repo} value={repo}>
                  {truncate(formatRepoUrl(repo), 40)}
                </SelectItem>
              ))}
            </SelectContent>
          </Select>
        </div>

        {/* Severity filter */}
        <div className="flex items-center gap-2">
          <span className="text-sm font-medium text-[hsl(var(--muted-foreground))] whitespace-nowrap">
            {t('findings.severity')}:
          </span>
          <Select value={severityFilter} onValueChange={(v) => setSeverityFilter(v as SeverityOption)}>
            <SelectTrigger className="w-[140px]">
              <SelectValue />
            </SelectTrigger>
            <SelectContent>
              {severityOptions.map((s) => (
                <SelectItem key={s} value={s}>
                  {s === 'all' ? t('common.all') : t(`statistics.severity.${s}`, { defaultValue: s.toUpperCase() })}
                </SelectItem>
              ))}
            </SelectContent>
          </Select>
        </div>

        {/* Category filter */}
        <div className="flex items-center gap-2">
          <span className="text-sm font-medium text-[hsl(var(--muted-foreground))] whitespace-nowrap">
            {t('findings.category')}:
          </span>
          <Select value={categoryFilter} onValueChange={setCategoryFilter}>
            <SelectTrigger className="w-[180px]">
              <SelectValue />
            </SelectTrigger>
            <SelectContent>
              <SelectItem value="all">{t('common.all')}</SelectItem>
              {uniqueCategories.map((cat) => (
                <SelectItem key={cat} value={cat}>
                  {t(`statistics.category.${cat}`, { defaultValue: cat })}
                </SelectItem>
              ))}
            </SelectContent>
          </Select>
        </div>

        {/* Spacer */}
        <div className="flex-1" />

        {/* Refresh button */}
        <Button variant="outline" size="sm" onClick={() => refetch()}>
          <RefreshCw className="mr-2 h-4 w-4" />
          {t('common.refresh')}
        </Button>
      </div>

      {/* Table */}
      <div className="rounded-md border border-[hsl(var(--border))]">
        <Table>
          <TableHeader>
            <TableRow>
              <TableHead className="w-[250px]">{t('findings.repository')}</TableHead>
              <TableHead className="w-[100px]">{t('findings.severity')}</TableHead>
              <TableHead className="w-[120px]">{t('findings.category')}</TableHead>
              <TableHead>{t('findings.description')}</TableHead>
              <TableHead className="w-[120px]">{t('findings.time')}</TableHead>
              <TableHead className="w-[80px]">{t('findings.actions')}</TableHead>
            </TableRow>
          </TableHeader>
          <TableBody>
            {isLoading ? (
              [...Array(5)].map((_, i) => (
                <TableRow key={i}>
                  {[...Array(6)].map((_, j) => (
                    <TableCell key={j}>
                      <Skeleton className="h-5 w-full" />
                    </TableCell>
                  ))}
                </TableRow>
              ))
            ) : findings.length === 0 ? (
              <TableRow>
                <TableCell colSpan={6} className="h-64">
                  <div className="flex flex-col items-center justify-center gap-4 py-8">
                    <AlertCircle className="h-12 w-12 text-[hsl(var(--muted-foreground))] opacity-50" />
                    <p className="text-sm text-[hsl(var(--muted-foreground))]">
                      {t('findings.noFindings')}
                    </p>
                  </div>
                </TableCell>
              </TableRow>
            ) : (
              findings.map((finding: FindingItem, index: number) => (
                <TableRow key={`${finding.review_id}-${index}`}>
                  <TableCell>
                    <span title={finding.repo_url}>
                      {truncate(formatRepoUrl(finding.repo_url), 35)}
                    </span>
                  </TableCell>
                  <TableCell>
                    <span
                      className={`inline-flex items-center px-2 py-0.5 text-xs font-medium rounded border ${getSeverityBadgeClass(finding.severity)}`}
                    >
                      {finding.severity.toUpperCase()}
                    </span>
                  </TableCell>
                  <TableCell>
                    {finding.category ? (
                      <Badge 
                        variant="secondary" 
                        className="text-xs font-normal bg-[hsl(var(--primary))]/8 text-[hsl(var(--primary))] border border-[hsl(var(--primary))]/15"
                      >
                        {t(`statistics.category.${finding.category}`, { defaultValue: finding.category })}
                      </Badge>
                    ) : (
                      <span className="text-sm text-[hsl(var(--muted-foreground))]">-</span>
                    )}
                  </TableCell>
                  <TableCell>
                    <p
                      className="text-xs line-clamp-2 leading-relaxed"
                      title={finding.description}
                    >
                      {finding.description || '-'}
                    </p>
                  </TableCell>
                  <TableCell className="text-[hsl(var(--muted-foreground))]">
                    {formatRelativeTime(finding.created_at, t)}
                  </TableCell>
                  <TableCell>
                    <Button variant="ghost" size="sm" asChild>
                      <Link to={`/admin/reviews/${finding.review_id}`}>
                        {t('common.view')}
                        <ExternalLink className="h-4 w-4 ml-0.5 text-[hsl(var(--primary))]" />
                      </Link>
                    </Button>
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
          {/* Page size selector */}
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
            {/* First page */}
            <Button
              variant="outline"
              size="icon"
              className="h-8 w-8"
              onClick={() => setPage(1)}
              disabled={page === 1}
            >
              <ChevronFirst className="h-4 w-4" />
            </Button>
            {/* Previous page */}
            <Button
              variant="outline"
              size="icon"
              className="h-8 w-8"
              onClick={() => setPage((p) => Math.max(1, p - 1))}
              disabled={page === 1}
            >
              <ChevronLeft className="h-4 w-4" />
            </Button>
            {/* Page indicator */}
            <span className="text-sm min-w-[60px] text-center">
              {page} / {totalPages}
            </span>
            {/* Next page */}
            <Button
              variant="outline"
              size="icon"
              className="h-8 w-8"
              onClick={() => setPage((p) => Math.min(totalPages, p + 1))}
              disabled={page === totalPages}
            >
              <ChevronRight className="h-4 w-4" />
            </Button>
            {/* Last page */}
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
    </div>
  )
}
