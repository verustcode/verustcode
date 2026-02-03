import { useState } from 'react'
import { useTranslation } from 'react-i18next'
import { useQuery, useMutation, useQueryClient, keepPreviousData } from '@tanstack/react-query'
import { useNavigate, Link } from 'react-router-dom'
import { Search, X, RefreshCw, Plus, Info, ChevronFirst, ChevronLast, ChevronLeft, ChevronRight, Trash2, ArrowUpDown, ArrowUp, ArrowDown } from 'lucide-react'
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
import { Label } from '@/components/ui/label'
import { api } from '@/lib/api'
import { formatRelativeTime, formatRepoUrl } from '@/lib/utils'
import { toast } from '@/hooks/useToast'
import type { RepositoryConfigItem } from '@/types/repository'

// Page size options and storage key
const PAGE_SIZE_OPTIONS = [10, 20, 50, 100] as const
const PAGE_SIZE_STORAGE_KEY = 'repositories_page_size'



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
 * Repositories page - Manage repository review configurations
 */
export default function Repositories() {
  const { t } = useTranslation()
  const queryClient = useQueryClient()
  const navigate = useNavigate()

  // Pagination and filter state
  const [page, setPage] = useState(1)
  const [pageSize, setPageSize] = useState(getStoredPageSize)
  const [search, setSearch] = useState('')
  
  // Sorting state
  type SortField = 'review_count' | 'last_review_at' | null
  const [sortBy, setSortBy] = useState<SortField>(null)
  const [sortOrder, setSortOrder] = useState<'asc' | 'desc'>('desc')

  // Dialog state
  const [showAddDialog, setShowAddDialog] = useState(false)
  const [showEditDialog, setShowEditDialog] = useState(false)
  const [showDeleteDialog, setShowDeleteDialog] = useState(false)
  const [selectedRepo, setSelectedRepo] = useState<RepositoryConfigItem | null>(null)

  // Form state for add/edit
  const [formUrl, setFormUrl] = useState('')
  const [formReviewFile, setFormReviewFile] = useState('')
  const [formDescription, setFormDescription] = useState('')
  const [urlParseError, setUrlParseError] = useState('')

  // Fetch repositories
  const { data, isLoading, refetch } = useQuery({
    queryKey: ['repositories', page, pageSize, search, sortBy, sortOrder],
    queryFn: () =>
      api.admin.repositories.list({
        page,
        page_size: pageSize,
        search: search || undefined,
        sort_by: sortBy || undefined,
        sort_order: sortBy ? sortOrder : undefined,
      }),
    placeholderData: keepPreviousData, // Keep old data during refetch to prevent flicker
  })
  
  // Handle column sort click
  const handleSortClick = (field: SortField) => {
    if (sortBy === field) {
      // Already sorting by this field: toggle order or clear
      if (sortOrder === 'desc') {
        setSortOrder('asc')
      } else {
        // Clear sorting
        setSortBy(null)
        setSortOrder('desc')
      }
    } else {
      // Set new sort field with desc order
      setSortBy(field)
      setSortOrder('desc')
    }
    setPage(1)
  }
  
  // Get sort icon for a column
  const getSortIcon = (field: SortField) => {
    if (sortBy !== field) {
      return <ArrowUpDown className="h-3.5 w-3.5 opacity-50" />
    }
    return sortOrder === 'desc' 
      ? <ArrowDown className="h-3.5 w-3.5" />
      : <ArrowUp className="h-3.5 w-3.5" />
  }

  // Fetch available review files
  const { data: reviewFilesData } = useQuery({
    queryKey: ['review-files'],
    queryFn: () => api.admin.reviewFiles.list(),
  })

  const reviewFiles = reviewFilesData?.files || []

  // Mutations
  const createMutation = useMutation({
    mutationFn: api.admin.repositories.create,
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['repositories'] })
      setShowAddDialog(false)
      resetForm()
      toast({ title: t('repositories.createSuccess') })
    },
  })

  const updateMutation = useMutation({
    mutationFn: ({ id, data }: { id: number; data: { review_file?: string; description?: string } }) =>
      api.admin.repositories.update(id, data),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['repositories'] })
      setShowEditDialog(false)
      setSelectedRepo(null)
      toast({ title: t('repositories.updateSuccess') })
    },
  })

  const deleteMutation = useMutation({
    mutationFn: api.admin.repositories.delete,
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['repositories'] })
      setShowDeleteDialog(false)
      setSelectedRepo(null)
      toast({ title: t('repositories.deleteSuccess') })
    },
  })

  const repositories = data?.data || []
  const total = data?.total || 0
  const totalPages = Math.ceil(total / pageSize)

  // Reset form
  const resetForm = () => {
    setFormUrl('')
    setFormReviewFile('')
    setFormDescription('')
    setUrlParseError('')
  }

  // Handle add repository
  const handleAdd = () => {
    resetForm()
    setShowAddDialog(true)
  }

  // Handle edit repository
  const handleEdit = (repo: RepositoryConfigItem) => {
    setSelectedRepo(repo)
    setFormReviewFile(repo.review_file || '')
    setFormDescription(repo.description || '')
    setShowEditDialog(true)
  }

  // Handle delete repository
  const handleDelete = (repo: RepositoryConfigItem) => {
    setSelectedRepo(repo)
    setShowDeleteDialog(true)
  }

  // Handle audit - navigate to reports page with repo URL pre-filled
  const handleAudit = (repo: RepositoryConfigItem) => {
    // Navigate to reports page with repo_url as query parameter
    navigate(`/admin/reports?repo_url=${encodeURIComponent(repo.repo_url)}`)
  }

  // Handle create submit - create with repo URL directly
  const handleCreateSubmit = async () => {
    if (!formUrl.trim()) return
    
    createMutation.mutate({
      repo_url: formUrl.trim(),
      review_file: formReviewFile || undefined,
      description: formDescription || undefined,
    })
  }

  // Handle update submit
  const handleUpdateSubmit = () => {
    if (!selectedRepo) return
    updateMutation.mutate({
      id: selectedRepo.id,
      data: {
        review_file: formReviewFile || undefined,
        description: formDescription || undefined,
      },
    })
  }

  // Handle delete confirm
  const handleDeleteConfirm = () => {
    if (!selectedRepo) return
    deleteMutation.mutate(selectedRepo.id)
  }

  // Handle inline review file change (reserved for future use)
  const _handleReviewFileChange = (repo: RepositoryConfigItem, newFile: string) => {
    // If repo has no ID, we need to create a config first
    if (!repo.id) {
      createMutation.mutate({
        repo_url: repo.repo_url,
        review_file: newFile === 'default.yaml' ? '' : newFile,
      })
    } else {
      updateMutation.mutate({
        id: repo.id,
        data: {
          review_file: newFile === 'default.yaml' ? '' : newFile,
        },
      })
    }
  }
  void _handleReviewFileChange // Suppress unused warning

  return (
    <div className="space-y-8">
      {/* Header with hint - default.yaml is a clickable link to Rules page */}
      <div className="flex items-center gap-3 rounded-md bg-[hsl(var(--primary)/0.08)] px-4 py-3 text-sm text-[hsl(var(--primary))]">
        <Info className="h-4 w-4 flex-shrink-0" />
        <span>
          {t('repositories.defaultConfigHintPrefix')}
          <Link 
            to="/admin/rules?file=default.yaml" 
            className="underline hover:opacity-80 transition-opacity font-medium"
          >
            default.yaml
          </Link>
          {t('repositories.defaultConfigHintSuffix')}
        </span>
      </div>

      {/* Filters */}
      <div className="flex flex-col gap-6 sm:flex-row sm:items-center">
        {/* Search */}
        <div className="relative flex-1">
          <Search className="absolute left-3 top-1/2 h-4 w-4 -translate-y-1/2 text-[hsl(var(--muted-foreground))]" />
          <Input
            placeholder={t('repositories.searchPlaceholder')}
            value={search}
            onChange={(e) => {
              setSearch(e.target.value)
              setPage(1)
            }}
            className="pl-9"
          />
          {search && (
            <Button
              variant="ghost"
              size="icon"
              className="absolute right-1 top-1/2 h-7 w-7 -translate-y-1/2"
              onClick={() => {
                setSearch('')
                setPage(1)
              }}
            >
              <X className="h-4 w-4" />
            </Button>
          )}
        </div>

        {/* Refresh button */}
        <Button variant="outline" size="sm" onClick={() => refetch()}>
          <RefreshCw className="mr-2 h-4 w-4" />
          {t('common.refresh')}
        </Button>

        {/* Add button */}
        <Button size="sm" onClick={handleAdd}>
          <Plus className="mr-2 h-4 w-4" />
          {t('repositories.addRepository')}
        </Button>
      </div>

      {/* Table */}
      <div className="rounded-md border border-[hsl(var(--border))]">
        <Table>
          <TableHeader>
            <TableRow>
              <TableHead className="w-[320px]">{t('repositories.repository')}</TableHead>
              <TableHead className="w-[180px]">{t('repositories.reviewConfig')}</TableHead>
              <TableHead className="w-[100px]">
                <button
                  className="flex items-center gap-1.5 hover:text-[hsl(var(--foreground))] transition-colors"
                  onClick={() => handleSortClick('review_count')}
                >
                  {t('repositories.reviewCount')}
                  {getSortIcon('review_count')}
                </button>
              </TableHead>
              <TableHead className="w-[140px]">
                <button
                  className="flex items-center gap-1.5 hover:text-[hsl(var(--foreground))] transition-colors"
                  onClick={() => handleSortClick('last_review_at')}
                >
                  {t('repositories.lastReview')}
                  {getSortIcon('last_review_at')}
                </button>
              </TableHead>
              <TableHead className="w-[100px]">{t('repositories.actions')}</TableHead>
            </TableRow>
          </TableHeader>
          <TableBody>
            {isLoading ? (
              [...Array(5)].map((_, i) => (
                <TableRow key={i}>
                  {[...Array(5)].map((_, j) => (
                    <TableCell key={j}>
                      <Skeleton className="h-5 w-full" />
                    </TableCell>
                  ))}
                </TableRow>
              ))
            ) : repositories.length === 0 ? (
              <TableRow>
                <TableCell colSpan={5} className="h-64">
                  <div className="flex flex-col items-center justify-center gap-4 py-8">
                    <img
                      src="assets/empty-repositories.svg"
                      alt="No repositories"
                      className="h-32 w-auto opacity-80"
                    />
                    <p className="text-sm text-[hsl(var(--muted-foreground))]">
                      {t('repositories.noRepositories')}
                    </p>
                  </div>
                </TableCell>
              </TableRow>
            ) : (
              repositories.map((repo, index) => (
                <TableRow key={repo.id || `${repo.repo_url}-${index}`}>
                  <TableCell>
                    <span className="truncate block max-w-[300px]" title={repo.repo_url}>
                      {formatRepoUrl(repo.repo_url)}
                    </span>
                  </TableCell>
                  <TableCell>
                    <span className="text-sm">
                      {repo.review_file || 'default.yaml'}
                    </span>
                  </TableCell>
                  <TableCell className="text-[hsl(var(--muted-foreground))]">
                    {repo.review_count}
                  </TableCell>
                  <TableCell className="text-sm text-[hsl(var(--muted-foreground))]">
                    {repo.last_review_at ? (
                      (() => {
                        const date = new Date(repo.last_review_at)
                        return !isNaN(date.getTime()) && date.getFullYear() > 2000
                          ? formatRelativeTime(repo.last_review_at, t)
                          : '-'
                      })()
                    ) : '-'}
                  </TableCell>
                  <TableCell>
                    <div className="flex items-center gap-1">
                      <Button
                        variant="ghost"
                        size="sm"
                        className="h-8"
                        onClick={() => handleEdit(repo)}
                      >
                        {t('common.edit')}
                      </Button>
                      <Button
                        variant="ghost"
                        size="sm"
                        className="h-8"
                        onClick={() => handleAudit(repo)}
                      >
                        {t('repositories.generateReport')}
                      </Button>
                      {repo.id > 0 && (
                        <Button
                          variant="ghost"
                          size="icon"
                          className="h-8 w-8 text-[hsl(var(--destructive))]"
                          onClick={() => handleDelete(repo)}
                          title={t('common.delete')}
                        >
                          <Trash2 className="h-4 w-4" />
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

      {/* Add Repository Dialog */}
      <Dialog open={showAddDialog} onOpenChange={setShowAddDialog}>
        <DialogContent>
          <DialogHeader>
            <DialogTitle>{t('repositories.addRepository')}</DialogTitle>
            <DialogDescription>
              {t('repositories.addRepositoryDescription')}
            </DialogDescription>
          </DialogHeader>
          <div className="grid gap-4 py-4">
            <div className="grid gap-2">
              <Label htmlFor="repoUrl">{t('repositories.repoUrl')}</Label>
              <Input
                id="repoUrl"
                value={formUrl}
                onChange={(e) => {
                  setFormUrl(e.target.value)
                  setUrlParseError('')
                }}
                placeholder="github.com/owner/repo"
              />
              {urlParseError && (
                <p className="text-sm text-[hsl(var(--destructive))]">{urlParseError}</p>
              )}
            </div>
            <div className="grid gap-2">
              <Label htmlFor="reviewFile">{t('repositories.reviewConfig')}</Label>
              <Select value={formReviewFile || 'default.yaml'} onValueChange={setFormReviewFile}>
                <SelectTrigger>
                  <SelectValue />
                </SelectTrigger>
                <SelectContent>
                  {reviewFiles.map((file) => (
                    <SelectItem key={file} value={file}>
                      {file}
                    </SelectItem>
                  ))}
                </SelectContent>
              </Select>
            </div>
            <div className="grid gap-2">
              <Label htmlFor="description">{t('repositories.description')}</Label>
              <Input
                id="description"
                value={formDescription}
                onChange={(e) => setFormDescription(e.target.value)}
                placeholder={t('repositories.descriptionPlaceholder')}
              />
            </div>
          </div>
          <DialogFooter>
            <Button variant="outline" onClick={() => setShowAddDialog(false)}>
              {t('common.cancel')}
            </Button>
            <Button
              onClick={handleCreateSubmit}
              disabled={!formUrl.trim() || createMutation.isPending}
            >
              {t('common.create')}
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>

      {/* Edit Repository Dialog */}
      <Dialog open={showEditDialog} onOpenChange={setShowEditDialog}>
        <DialogContent>
          <DialogHeader>
            <DialogTitle>{t('repositories.editRepository')}</DialogTitle>
            <DialogDescription>
              {selectedRepo?.repo_url}
            </DialogDescription>
          </DialogHeader>
          <div className="grid gap-4 py-4">
            <div className="grid gap-2">
              <Label htmlFor="editReviewFile">{t('repositories.reviewConfig')}</Label>
              <Select value={formReviewFile || 'default.yaml'} onValueChange={setFormReviewFile}>
                <SelectTrigger>
                  <SelectValue />
                </SelectTrigger>
                <SelectContent>
                  {reviewFiles.map((file) => (
                    <SelectItem key={file} value={file}>
                      {file}
                    </SelectItem>
                  ))}
                </SelectContent>
              </Select>
            </div>
            <div className="grid gap-2">
              <Label htmlFor="editDescription">{t('repositories.description')}</Label>
              <Input
                id="editDescription"
                value={formDescription}
                onChange={(e) => setFormDescription(e.target.value)}
                placeholder={t('repositories.descriptionPlaceholder')}
              />
            </div>
          </div>
          <DialogFooter>
            <Button variant="outline" onClick={() => setShowEditDialog(false)}>
              {t('common.cancel')}
            </Button>
            <Button onClick={handleUpdateSubmit} disabled={updateMutation.isPending}>
              {t('common.save')}
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>

      {/* Delete Confirmation Dialog */}
      <Dialog open={showDeleteDialog} onOpenChange={setShowDeleteDialog}>
        <DialogContent>
          <DialogHeader>
            <DialogTitle>{t('repositories.deleteRepository')}</DialogTitle>
            <DialogDescription>
              {t('repositories.deleteConfirm', { repo: selectedRepo?.repo_url || '' })}
            </DialogDescription>
          </DialogHeader>
          <DialogFooter>
            <Button variant="outline" onClick={() => setShowDeleteDialog(false)}>
              {t('common.cancel')}
            </Button>
            <Button
              variant="destructive"
              onClick={handleDeleteConfirm}
              disabled={deleteMutation.isPending}
            >
              {t('common.delete')}
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>
    </div>
  )
}

