import { useState, useEffect } from 'react'
import { useTranslation } from 'react-i18next'
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query'
import { Link, useLocation } from 'react-router-dom'
import { Search, X, ExternalLink, Ban, RefreshCw, ChevronFirst, ChevronLast, ChevronLeft, ChevronRight, Copy, Check } from 'lucide-react'
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
import { Skeleton } from '@/components/ui/skeleton'
import { StatusBadge } from '@/components/common/StatusBadge'
import { api } from '@/lib/api'
import { formatDate, formatDuration, truncate, formatRepoUrl } from '@/lib/utils'
import type { Review, ReviewStatus } from '@/types/review'

const statusOptions: (ReviewStatus | 'all')[] = ['all', 'pending', 'running', 'completed', 'failed', 'cancelled']

// Page size options and storage key
const PAGE_SIZE_OPTIONS = [10, 20, 50, 100] as const
const PAGE_SIZE_STORAGE_KEY = 'reviews_page_size'

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
 * Reviews list page component
 */
export default function Reviews() {
  const { t } = useTranslation()
  const queryClient = useQueryClient()
  const location = useLocation()

  const [page, setPage] = useState(1)
  const [pageSize, setPageSize] = useState(getStoredPageSize)
  const [status, setStatus] = useState<ReviewStatus | 'all'>('all')
  const [search, setSearch] = useState('')
  const [showCancelDialog, setShowCancelDialog] = useState(false)
  const [cancelReviewId, setCancelReviewId] = useState<string | null>(null)
  const [copiedId, setCopiedId] = useState<string | null>(null)

  // Fetch reviews
  const { data, isLoading, refetch } = useQuery({
    queryKey: ['reviews', page, pageSize, status, search],
    queryFn: () =>
      api.reviews.list({
        page,
        page_size: pageSize,
        status: status === 'all' ? undefined : status,
      }),
    refetchInterval: 10000,
  })

  // 当切换到审查列表页面时自动刷新数据
  useEffect(() => {
    // 只在路径为 /admin/reviews 时触发刷新
    if (location.pathname === '/admin/reviews') {
      refetch()
    }
  }, [location.pathname, refetch])

  // Cancel review mutation
  const cancelMutation = useMutation({
    mutationFn: (id: string) => api.reviews.cancel(id),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['reviews'] })
    },
  })

  const reviews = (data?.data || []) as Review[]
  const total = data?.total || 0
  const totalPages = Math.ceil(total / pageSize)

  // Filter by search locally
  const filteredReviews = search
    ? reviews.filter(
        (r) =>
          r.repo_url.toLowerCase().includes(search.toLowerCase()) ||
          String(r.pr_number).includes(search)
      )
    : reviews

  const handleCancel = (id: string) => {
    setCancelReviewId(id)
    setShowCancelDialog(true)
  }

  const handleCancelConfirm = () => {
    if (cancelReviewId !== null) {
      cancelMutation.mutate(cancelReviewId)
    }
    setShowCancelDialog(false)
    setCancelReviewId(null)
  }

  const handleCopyReviewId = async (id: string) => {
    try {
      await navigator.clipboard.writeText(id)
      setCopiedId(id)
      setTimeout(() => setCopiedId(null), 2000)
    } catch (err) {
      console.error('Failed to copy review ID:', err)
    }
  }

  return (
    <div className="space-y-8">
      {/* Filters */}
      <div className="flex flex-col gap-6 sm:flex-row sm:items-center">
        {/* Search */}
        <div className="relative flex-1">
          <Search className="absolute left-3 top-1/2 h-4 w-4 -translate-y-1/2 text-[hsl(var(--muted-foreground))]" />
          <Input
            placeholder={t('reviews.searchPlaceholder')}
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
        <Select value={status} onValueChange={(v) => setStatus(v as ReviewStatus | 'all')}>
          <SelectTrigger className="w-[180px]">
            <SelectValue placeholder={t('reviews.filterByStatus')} />
          </SelectTrigger>
          <SelectContent>
            {statusOptions.map((s) => (
              <SelectItem key={s} value={s}>
                {s === 'all' ? t('common.all') : t(`reviews.status.${s}`)}
              </SelectItem>
            ))}
          </SelectContent>
        </Select>

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
              <TableHead className="w-[150px]">{t('reviews.id')}</TableHead>
              <TableHead>{t('reviews.repository')}</TableHead>
              <TableHead className="w-[80px]">{t('reviews.prNumber')}</TableHead>
              <TableHead className="w-[100px]">{t('common.status')}</TableHead>
              <TableHead className="w-[80px]">{t('reviews.source')}</TableHead>
              <TableHead className="w-[160px]">{t('reviews.createdAt')}</TableHead>
              <TableHead className="w-[100px]">{t('reviews.duration')}</TableHead>
              <TableHead className="w-[100px]">{t('reviews.actions')}</TableHead>
            </TableRow>
          </TableHeader>
          <TableBody>
            {isLoading ? (
              [...Array(5)].map((_, i) => (
                <TableRow key={i}>
                  {[...Array(8)].map((_, j) => (
                    <TableCell key={j}>
                      <Skeleton className="h-5 w-full" />
                    </TableCell>
                  ))}
                </TableRow>
              ))
            ) : filteredReviews.length === 0 ? (
              <TableRow>
                <TableCell colSpan={8} className="h-24 text-center">
                  {t('reviews.noReviews')}
                </TableCell>
              </TableRow>
            ) : (
              filteredReviews.map((review) => (
                <TableRow key={review.id}>
                  <TableCell className="font-mono text-xs">
                    <div className="flex items-center gap-2">
                      <span className="text-[hsl(var(--muted-foreground))]">{review.id}</span>
                      <button
                        onClick={(e) => {
                          e.stopPropagation()
                          handleCopyReviewId(review.id)
                        }}
                        className="p-1 rounded hover:bg-[hsl(var(--muted))] transition-colors"
                        title={t('common.copy')}
                      >
                        {copiedId === review.id ? (
                          <Check className="h-3 w-3 text-green-500" />
                        ) : (
                          <Copy className="h-3 w-3 text-[hsl(var(--muted-foreground))] opacity-50 hover:opacity-100" />
                        )}
                      </button>
                    </div>
                  </TableCell>
                  <TableCell>
                    <span title={review.repo_url}>
                      {truncate(formatRepoUrl(review.repo_url), 50)}
                    </span>
                  </TableCell>
                  <TableCell>
                    {review.pr_url ? (
                      <a
                        href={review.pr_url}
                        target="_blank"
                        rel="noopener noreferrer"
                        className="flex items-center gap-1 text-[hsl(var(--primary))] hover:underline"
                      >
                        #{review.pr_number}
                        <ExternalLink className="h-3 w-3" />
                      </a>
                    ) : (
                      review.pr_number || '-'
                    )}
                  </TableCell>
                  <TableCell>
                    <StatusBadge status={review.status} />
                  </TableCell>
                  <TableCell className="text-[hsl(var(--muted-foreground))]">
                    {review.source}
                  </TableCell>
                  <TableCell className="text-[hsl(var(--muted-foreground))]">
                    {formatDate(review.created_at)}
                  </TableCell>
                  <TableCell className="text-[hsl(var(--muted-foreground))]">
                    {review.duration ? formatDuration(review.duration) : '-'}
                  </TableCell>
                  <TableCell>
                    <div className="flex items-center gap-1">
                      <Button variant="ghost" size="sm" asChild>
                        <Link to={`/admin/reviews/${review.id}`}>
                          {t('reviews.viewDetails')}
                        </Link>
                      </Button>
                      {(review.status === 'pending' || review.status === 'running') && (
                        <Button
                          variant="ghost"
                          size="icon"
                          className="h-8 w-8 text-[hsl(var(--destructive))]"
                          onClick={() => handleCancel(review.id)}
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
          {/* Page size selector */}
          <Select
            value={String(pageSize)}
            onValueChange={(v) => {
              const size = parseInt(v, 10)
              setPageSize(size)
              localStorage.setItem(PAGE_SIZE_STORAGE_KEY, v)
              // Reset to first page when changing page size
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

      {/* Cancel confirmation dialog */}
      <Dialog open={showCancelDialog} onOpenChange={setShowCancelDialog}>
        <DialogContent>
          <DialogHeader>
            <DialogTitle>{t('reviews.cancelReview')}</DialogTitle>
            <DialogDescription>
              {t('reviews.cancelConfirm')}
            </DialogDescription>
          </DialogHeader>
          <DialogFooter>
            <Button
              variant="outline"
              onClick={() => setShowCancelDialog(false)}
            >
              {t('common.cancel')}
            </Button>
            <Button
              variant="destructive"
              onClick={handleCancelConfirm}
              disabled={cancelMutation.isPending}
            >
              {t('reviews.cancelReview')}
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>
    </div>
  )
}
