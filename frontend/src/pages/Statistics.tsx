import { useState, useEffect } from 'react'
import { useTranslation } from 'react-i18next'
import { useQuery } from '@tanstack/react-query'
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select'
import { api } from '@/lib/api'
import { formatRepoUrl } from '@/lib/utils'
import { DeliveryTimeChart } from '@/components/stats/DeliveryTimeChart'
import { CodeChangeChart } from '@/components/stats/CodeChangeChart'
import { MRCountChart } from '@/components/stats/MRCountChart'
import { RevisionChart } from '@/components/stats/RevisionChart'
import { IssueSeverityChart } from '@/components/stats/IssueSeverityChart'
import { IssueCategoryChart } from '@/components/stats/IssueCategoryChart'
import type { RepoStats, TimeRange, RepoOption } from '@/types/stats'
import type { Review } from '@/types/review'

/**
 * Statistics page component
 * 审查统计页面 - 仓库级别统计
 */
export default function Statistics() {
  const { t } = useTranslation()

  // State for filters
  const [selectedRepo, setSelectedRepo] = useState<string>('all')
  const [timeRange, setTimeRange] = useState<TimeRange>('3m')

  // Fetch repository list for selector
  const { data: reviewsData } = useQuery({
    queryKey: ['reviews', 'all'],
    queryFn: () => api.reviews.list({ page: 1, page_size: 1000 }),
  })

  // Extract unique repositories
  const repoOptions: RepoOption[] = [
    { label: t('statistics.allRepos'), value: 'all' },
  ]

  if (reviewsData?.data) {
    const reviews = reviewsData.data as Review[]
    const uniqueRepos = new Set<string>()
    reviews.forEach((review) => {
      if (review.repo_url) {
        uniqueRepos.add(review.repo_url)
      }
    })
    uniqueRepos.forEach((repoUrl) => {
      // Extract repo name from URL for display
      const repoName = formatRepoUrl(repoUrl)
      repoOptions.push({ label: repoName, value: repoUrl })
    })
  }

  // Fetch statistics data
  const {
    data: statsData,
    isLoading,
    refetch,
  } = useQuery<RepoStats>({
    queryKey: ['statistics', 'repo', selectedRepo, timeRange],
    queryFn: () =>
      api.admin.statistics.repo({
        repo_url: selectedRepo === 'all' ? undefined : selectedRepo,
        time_range: timeRange,
      }) as Promise<RepoStats>,
    refetchInterval: 30000, // Refresh every 30 seconds
  })

  // Refetch when filters change
  useEffect(() => {
    refetch()
  }, [selectedRepo, timeRange, refetch])

  return (
    <div className="space-y-6">
      {/* Filters */}
      <div className="flex flex-col gap-4 sm:flex-row sm:items-center">
        {/* Repository selector */}
        <div className="flex items-center gap-2">
          <span className="text-sm font-medium text-[hsl(var(--muted-foreground))]">
            {t('statistics.repository')}:
          </span>
          <Select value={selectedRepo} onValueChange={setSelectedRepo}>
            <SelectTrigger className="w-[300px]">
              <SelectValue />
            </SelectTrigger>
            <SelectContent>
              {repoOptions.map((option) => (
                <SelectItem key={option.value} value={option.value}>
                  {option.label}
                </SelectItem>
              ))}
            </SelectContent>
          </Select>
        </div>

        {/* Time range selector */}
        <div className="flex items-center gap-2">
          <span className="text-sm font-medium text-[hsl(var(--muted-foreground))]">
            {t('statistics.timeRange')}:
          </span>
          <Select value={timeRange} onValueChange={(v) => setTimeRange(v as TimeRange)}>
            <SelectTrigger className="w-[150px]">
              <SelectValue />
            </SelectTrigger>
            <SelectContent>
              <SelectItem value="3m">{t('statistics.threeMonths')}</SelectItem>
              <SelectItem value="6m">{t('statistics.sixMonths')}</SelectItem>
              <SelectItem value="1y">{t('statistics.oneYear')}</SelectItem>
            </SelectContent>
          </Select>
        </div>

        {/* Weekly Dimension Notice */}
        <div className="text-sm text-[hsl(var(--muted-foreground))]">
          {t('statistics.weeklyDimension')}
        </div>
      </div>

      {/* Issue Statistics - Two pie charts side by side */}
      <div className="grid gap-6 md:grid-cols-2">
        {/* Issue Severity Distribution */}
        <IssueSeverityChart
          data={statsData?.issue_severity_stats || []}
          loading={isLoading}
        />

        {/* Issue Category Distribution */}
        <IssueCategoryChart
          data={statsData?.issue_category_stats || []}
          loading={isLoading}
        />
      </div>

      {/* Charts Grid */}
      <div className="space-y-6">
        {/* MR Delivery Time */}
        <DeliveryTimeChart
          data={statsData?.delivery_time_stats || []}
          loading={isLoading}
          timeRange={timeRange}
        />

        {/* Code Changes (with File Changes line) */}
        <CodeChangeChart
          data={statsData?.code_change_stats || []}
          fileChangeData={statsData?.file_change_stats || []}
          loading={isLoading}
          timeRange={timeRange}
        />

        {/* MR Count */}
        <MRCountChart
          data={statsData?.mr_count_stats || []}
          loading={isLoading}
          timeRange={timeRange}
        />

        {/* Revision Count */}
        <RevisionChart
          data={statsData?.revision_stats || []}
          loading={isLoading}
          timeRange={timeRange}
        />
      </div>
    </div>
  )
}

