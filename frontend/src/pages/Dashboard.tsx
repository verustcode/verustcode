import { useTranslation } from 'react-i18next'
import { useQuery } from '@tanstack/react-query'
import { Link } from 'react-router-dom'
import {
  FileSearch,
  CheckCircle,
  Clock,
  ArrowRight,
  AlertCircle,
  Server,
  FolderGit2,
  FileText,
} from 'lucide-react'
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'
import { Button } from '@/components/ui/button'
import { Skeleton } from '@/components/ui/skeleton'
import { StatusBadge } from '@/components/common/StatusBadge'
import {
  Tooltip,
  TooltipContent,
  TooltipProvider,
  TooltipTrigger,
} from '@/components/ui/tooltip'
import { api, type Report } from '@/lib/api'
import { formatDuration, formatRelativeTime, formatRepoUrl } from '@/lib/utils'
import type { Review } from '@/types/review'
import type { DashboardStats, QueueStats, ServerStatus } from '@/types/api'

/**
 * Stats card component
 */
function StatsCard({
  title,
  value,
  icon: Icon,
  loading,
  className,
}: {
  title: string
  value: string | number
  icon: React.ElementType
  loading?: boolean
  className?: string
}) {
  return (
    <Card className={className}>
      {/* Compact padding for stats cards */}
      <CardHeader className="flex flex-row items-center justify-between space-y-0 p-3 pb-1">
        <CardTitle className="text-[11px] font-medium text-[hsl(var(--muted-foreground))] whitespace-nowrap">
          {title}
        </CardTitle>
        <Icon className="h-3.5 w-3.5 text-[hsl(var(--muted-foreground))] shrink-0 ml-1" />
      </CardHeader>
      <CardContent className="p-3 pt-0">
        {loading ? (
          <Skeleton className="h-5 w-16" />
        ) : (
          <div className="text-base font-bold">{value}</div>
        )}
      </CardContent>
    </Card>
  )
}

/**
 * Dashboard page component
 */
export default function Dashboard() {
  const { t } = useTranslation()

  // Fetch stats
  const { data: stats, isLoading: statsLoading } = useQuery<DashboardStats>({
    queryKey: ['admin', 'stats'],
    queryFn: () => api.admin.stats() as Promise<DashboardStats>,
    refetchInterval: 30000, // Refresh every 30 seconds
  })

  // Fetch recent reviews (3 items for dashboard)
  const { data: reviewsData, isLoading: reviewsLoading } = useQuery({
    queryKey: ['reviews', 'recent'],
    queryFn: () => api.reviews.list({ page: 1, page_size: 3 }),
    refetchInterval: 10000, // Refresh every 10 seconds
  })

  // Fetch recent reports (3 items for dashboard)
  const { data: reportsData, isLoading: reportsLoading } = useQuery({
    queryKey: ['reports', 'recent'],
    queryFn: () => api.reports.list({ page: 1, page_size: 3 }),
    refetchInterval: 10000, // Refresh every 10 seconds
  })

  // Fetch queue status
  // Use initialData to provide default values, avoiding flicker on first load
  const { data: queueStats } = useQuery<QueueStats>({
    queryKey: ['queue', 'status'],
    queryFn: () => api.queue.status() as Promise<QueueStats>,
    refetchInterval: 5000, // Refresh every 5 seconds
    placeholderData: (previousData) => previousData, // Keep previous data to avoid flicker
    initialData: { total_pending: 0, total_running: 0, repo_count: 0, repos: {} }, // Provide initial data
  })

  // Fetch server status
  const { data: serverStatus, isLoading: statusLoading } = useQuery<ServerStatus>({
    queryKey: ['admin', 'status'],
    queryFn: () => api.admin.status() as Promise<ServerStatus>,
    refetchInterval: 30000, // Refresh every 30 seconds
  })

  const reviews = (reviewsData?.data || []) as Review[]
  const reports = (reportsData?.data || []) as Report[]

  /**
   * Format memory size to human-readable string
   */
  const formatMemory = (bytes: number): string => {
    if (bytes < 1024) {
      return `${bytes} B`
    } else if (bytes < 1024 * 1024) {
      return `${(bytes / 1024).toFixed(1)} KB`
    } else if (bytes < 1024 * 1024 * 1024) {
      return `${(bytes / (1024 * 1024)).toFixed(1)} MB`
    } else {
      return `${(bytes / (1024 * 1024 * 1024)).toFixed(2)} GB`
    }
  }

  /**
   * Format uptime duration to human-readable string
   */
  const formatUptime = (seconds: number): string => {
    if (seconds < 60) {
      return t('dashboard.serverStatus.seconds', { count: seconds })
    }
    
    const days = Math.floor(seconds / 86400)
    const hours = Math.floor((seconds % 86400) / 3600)
    const minutes = Math.floor((seconds % 3600) / 60)
    
    const parts: string[] = []
    if (days > 0) {
      parts.push(t('dashboard.serverStatus.days', { count: days }))
    }
    if (hours > 0) {
      parts.push(t('dashboard.serverStatus.hours', { count: hours }))
    }
    if (minutes > 0 || parts.length === 0) {
      parts.push(t('dashboard.serverStatus.minutes', { count: minutes }))
    }
    
    return parts.join(' ')
  }

  /**
   * Format build time from YYYY-MM-DD_HH:MM:SS to friendly display (YYYY-MM-DD HH:MM:SS)
   */
  const formatBuildTime = (buildTime: string): string => {
    if (!buildTime || buildTime === 'unknown') {
      return '-'
    }
    
    // Simply replace underscore with space for better readability
    // Format: YYYY-MM-DD_HH:MM:SS -> YYYY-MM-DD HH:MM:SS
    return buildTime.replace('_', ' ')
  }

  return (
    <div className="space-y-6">
      {/* Stats cards - 6 cards grid */}
      <div className="grid gap-5 grid-cols-2 md:grid-cols-3 lg:grid-cols-6">
        <StatsCard
          title={t('dashboard.todayReviews')}
          value={stats?.today_reviews ?? 0}
          icon={FileSearch}
          loading={statsLoading}
        />
        <StatsCard
          title={t('dashboard.totalReviews')}
          value={stats?.total_reviews ?? 0}
          icon={FileSearch}
          loading={statsLoading}
        />
        <StatsCard
          title={t('dashboard.repositoryCount')}
          value={stats?.repository_count ?? 0}
          icon={FolderGit2}
          loading={statsLoading}
        />
        <StatsCard
          title={t('dashboard.totalReports')}
          value={stats?.total_reports ?? 0}
          icon={FileText}
          loading={statsLoading}
        />
        <StatsCard
          title={t('dashboard.reviewSuccessRate')}
          value={stats?.success_rate ? `${(stats.success_rate * 100).toFixed(1)}%` : '0%'}
          icon={CheckCircle}
          loading={statsLoading}
        />
        <StatsCard
          title={t('dashboard.reviewDuration')}
          value={stats?.avg_duration ? formatDuration(stats.avg_duration) : '0s'}
          icon={Clock}
          loading={statsLoading}
        />
      </div>

      {/* Recent activity - dual column layout */}
      <div className="grid gap-6 lg:grid-cols-2">
        {/* Recent reviews */}
        <Card>
          <CardHeader className="flex flex-row items-center justify-between">
            <CardTitle className="text-base">{t('dashboard.recentReviews')}</CardTitle>
            <div className="flex items-center gap-3">
              {/* Queue status badges with tooltips */}
              <TooltipProvider>
                <div className="hidden sm:flex items-center gap-2">
                  {/* Pending badge - 排队中 */}
                  <Tooltip>
                    <TooltipTrigger asChild>
                      <span className="inline-flex items-center gap-1.5 rounded-md bg-[hsl(var(--muted))] px-2 py-0.5 text-xs font-medium text-[hsl(var(--muted-foreground))] cursor-default">
                        <Clock className="h-3 w-3" />
                        {t('dashboard.pending')} {queueStats?.total_pending ?? 0}
                      </span>
                    </TooltipTrigger>
                    <TooltipContent>
                      <p>{t('dashboard.pending')}</p>
                    </TooltipContent>
                  </Tooltip>
                  {/* Running badge - 进行中 */}
                  <Tooltip>
                    <TooltipTrigger asChild>
                      <span className="inline-flex items-center gap-1.5 rounded-md bg-[hsl(var(--primary))] px-2 py-0.5 text-xs font-medium text-[hsl(var(--primary-foreground))] cursor-default">
                        {t('dashboard.running')} {queueStats?.total_running ?? 0}
                      </span>
                    </TooltipTrigger>
                    <TooltipContent>
                      <p>{t('dashboard.running')}</p>
                    </TooltipContent>
                  </Tooltip>
                </div>
              </TooltipProvider>
              <Button variant="ghost" size="sm" asChild>
                <Link to="/admin/reviews">
                  {t('dashboard.viewAll')}
                  <ArrowRight className="ml-1 h-4 w-4" />
                </Link>
              </Button>
            </div>
          </CardHeader>
          <CardContent>
            {reviewsLoading ? (
              <div className="space-y-3">
                {[...Array(3)].map((_, i) => (
                  <Skeleton key={i} className="h-12 w-full" />
                ))}
              </div>
            ) : reviews.length === 0 ? (
              <div className="flex flex-col items-center justify-center py-8 text-[hsl(var(--muted-foreground))]">
                <AlertCircle className="mb-2 h-8 w-8" />
                <p>{t('reviews.noReviews')}</p>
              </div>
            ) : (
              <div className="space-y-2">
                {reviews.map((review) => (
                  <Link
                    key={review.id}
                    to={`/admin/reviews/${review.id}`}
                    className="flex flex-col sm:flex-row sm:items-center sm:justify-between gap-2 sm:gap-4 rounded-md border border-[hsl(var(--border))] p-3 transition-colors hover:bg-[hsl(var(--muted))]"
                  >
                    {/* Left side: URL + PR */}
                    <div className="flex items-center gap-2 min-w-0">
                      <span className="text-sm text-[hsl(var(--muted-foreground))] truncate">
                        {formatRepoUrl(review.repo_url)}
                      </span>
                      {review.pr_number && (
                        <span className="text-xs text-[hsl(var(--muted-foreground))] shrink-0">
                          PR #{review.pr_number}
                        </span>
                      )}
                    </div>
                    {/* Right side: Status + Time */}
                    <div className="flex items-center gap-3 shrink-0">
                      <StatusBadge status={review.status} />
                      <span className="text-xs text-[hsl(var(--muted-foreground))] whitespace-nowrap">
                        {formatRelativeTime(review.created_at, t)}
                      </span>
                    </div>
                  </Link>
                ))}
              </div>
            )}
          </CardContent>
        </Card>

        {/* Recent reports */}
        <Card>
          <CardHeader className="flex flex-row items-center justify-between">
            <CardTitle className="text-base">{t('dashboard.recentReports')}</CardTitle>
            <Button variant="ghost" size="sm" asChild>
              <Link to="/admin/reports">
                {t('dashboard.viewAll')}
                <ArrowRight className="ml-1 h-4 w-4" />
              </Link>
            </Button>
          </CardHeader>
          <CardContent>
            {reportsLoading ? (
              <div className="space-y-3">
                {[...Array(3)].map((_, i) => (
                  <Skeleton key={i} className="h-12 w-full" />
                ))}
              </div>
            ) : reports.length === 0 ? (
              <div className="flex flex-col items-center justify-center py-8 text-[hsl(var(--muted-foreground))]">
                <AlertCircle className="mb-2 h-8 w-8" />
                <p>{t('reports.noReports')}</p>
              </div>
            ) : (
              <div className="space-y-2">
                {reports.map((report) => (
                  <Link
                    key={report.id}
                    to={`/admin/reports/${report.id}`}
                    className="flex flex-col sm:flex-row sm:items-center sm:justify-between gap-2 sm:gap-4 rounded-md border border-[hsl(var(--border))] p-3 transition-colors hover:bg-[hsl(var(--muted))]"
                  >
                    {/* Left side: URL + Report type */}
                    <div className="flex items-center gap-2 min-w-0">
                      <span className="text-sm text-[hsl(var(--muted-foreground))] truncate">
                        {formatRepoUrl(report.repo_url)}
                      </span>
                      <span className="inline-flex items-center rounded-md bg-[hsl(var(--muted))] px-2 py-0.5 text-xs font-medium text-[hsl(var(--muted-foreground))] shrink-0">
                        {report.report_type}
                      </span>
                    </div>
                    {/* Right side: Status + Time */}
                    <div className="flex items-center gap-3 shrink-0">
                      <StatusBadge status={report.status} />
                      <span className="text-xs text-[hsl(var(--muted-foreground))] whitespace-nowrap">
                        {formatRelativeTime(report.created_at, t)}
                      </span>
                    </div>
                  </Link>
                ))}
              </div>
            )}
          </CardContent>
        </Card>
      </div>

      {/* Server status card - placed at bottom */}
      <Card>
        <CardHeader className="flex flex-row items-center justify-between space-y-0 pb-2">
          <div className="flex items-center gap-2">
            <CardTitle className="text-base font-medium">
              {t('dashboard.serverStatus.title')}
            </CardTitle>
            {/* Health status indicator - 绿色圆点放在标题后面 */}
            {!statusLoading && (
              serverStatus && serverStatus.uptime >= 0 ? (
                <span className="relative flex h-2.5 w-2.5">
                  <span className="absolute inline-flex h-full w-full animate-ping rounded-full bg-green-400 opacity-75" />
                  <span className="relative inline-flex h-2.5 w-2.5 rounded-full bg-green-500" />
                </span>
              ) : (
                <span className="relative flex h-2.5 w-2.5">
                  <span className="relative inline-flex h-2.5 w-2.5 rounded-full bg-gray-400" />
                </span>
              )
            )}
          </div>
          <Server className="h-4 w-4 text-[hsl(var(--muted-foreground))]" />
        </CardHeader>
        <CardContent>
          {statusLoading ? (
            <div className="grid grid-cols-2 md:grid-cols-3 lg:grid-cols-5 gap-4">
              <Skeleton className="h-5 w-full" />
              <Skeleton className="h-5 w-full" />
              <Skeleton className="h-5 w-full" />
              <Skeleton className="h-5 w-full" />
              <Skeleton className="h-5 w-full" />
            </div>
          ) : (
            <div className="grid grid-cols-2 md:grid-cols-3 lg:grid-cols-5 gap-x-6 gap-y-3 text-sm">
              {/* Version */}
              <div className="flex flex-col gap-1">
                <span className="text-[hsl(var(--muted-foreground))] text-xs">
                  {t('dashboard.serverStatus.version')}
                </span>
                <span className="font-medium">
                  {serverStatus?.version ?? '-'}
                  {serverStatus?.git_commit && serverStatus.git_commit !== 'unknown' && (
                    <span className="ml-1 text-xs text-[hsl(var(--muted-foreground))]">
                      ({serverStatus.git_commit.substring(0, 7)})
                    </span>
                  )}
                </span>
              </div>
              {/* Build time */}
              <div className="flex flex-col gap-1">
                <span className="text-[hsl(var(--muted-foreground))] text-xs">
                  {t('dashboard.serverStatus.buildTime')}
                </span>
                <span className="font-medium">
                  {serverStatus?.build_time ? formatBuildTime(serverStatus.build_time) : '-'}
                </span>
              </div>
              {/* Uptime */}
              <div className="flex flex-col gap-1">
                <span className="text-[hsl(var(--muted-foreground))] text-xs">
                  {t('dashboard.serverStatus.uptime')}
                </span>
                <span className="font-medium">
                  {serverStatus?.uptime !== undefined ? formatUptime(serverStatus.uptime) : '-'}
                </span>
              </div>
              {/* Go version */}
              <div className="flex flex-col gap-1">
                <span className="text-[hsl(var(--muted-foreground))] text-xs">
                  {t('dashboard.serverStatus.goVersion')}
                </span>
                <span className="font-medium">
                  {serverStatus?.go_version ?? '-'}
                </span>
              </div>
              {/* Memory usage */}
              <div className="flex flex-col gap-1">
                <span className="text-[hsl(var(--muted-foreground))] text-xs">
                  {t('dashboard.serverStatus.memory')}
                </span>
                <span className="font-medium">
                  {serverStatus?.memory_usage !== undefined ? formatMemory(serverStatus.memory_usage) : '-'}
                </span>
              </div>
            </div>
          )}
        </CardContent>
      </Card>
    </div>
  )
}
