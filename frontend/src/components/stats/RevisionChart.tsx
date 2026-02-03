import { useTranslation } from 'react-i18next'
import {
  BarChart,
  Bar,
  XAxis,
  YAxis,
  CartesianGrid,
  Tooltip,
  ResponsiveContainer,
  Legend,
} from 'recharts'
import { AlertCircle } from 'lucide-react'
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'
import { Skeleton } from '@/components/ui/skeleton'
import type { WeeklyRevision, TimeRange } from '@/types/stats'

// Custom tooltip props interface
interface CustomTooltipProps {
  active?: boolean
  payload?: Array<{ payload: WeeklyRevision }>
}

interface RevisionChartProps {
  data: WeeklyRevision[]
  loading?: boolean
  timeRange?: TimeRange
}

// Layer colors - semantic colors for quality indication
// Green for low revisions (good), amber for medium, red for high revisions (needs attention)
const LAYER_COLORS = {
  layer_1: '#10b981', // green - 1-2 revisions
  layer_2: '#f59e0b', // amber - 3-4 revisions
  layer_3: '#ef4444', // red - 5+ revisions
}

/**
 * Convert ISO week format "2024-W01" to date string "2024/01/01" (Monday of that week)
 */
function weekToDate(weekStr: string): string {
  // Parse "2024-W01" format
  const match = weekStr.match(/^(\d{4})-W(\d{2})$/)
  if (!match) return weekStr

  const year = parseInt(match[1], 10)
  const week = parseInt(match[2], 10)

  // Calculate the Monday of the given ISO week
  // ISO week 1 is the week containing January 4th
  const jan4 = new Date(year, 0, 4)
  const dayOfWeek = jan4.getDay() || 7 // Convert Sunday (0) to 7
  const monday = new Date(jan4)
  monday.setDate(jan4.getDate() - dayOfWeek + 1 + (week - 1) * 7)

  const y = monday.getFullYear()
  const m = String(monday.getMonth() + 1).padStart(2, '0')
  const d = String(monday.getDate()).padStart(2, '0')

  return `${y}/${m}/${d}`
}

/**
 * Custom tooltip component for revision chart
 */
function CustomTooltip({ active, payload }: CustomTooltipProps) {
  const { t } = useTranslation()

  if (!active || !payload || payload.length === 0) {
    return null
  }

  const data = payload[0]?.payload
  if (!data) return null

  // Calculate percentages
  const total = data.mr_count || 0
  const layer1Pct = total > 0 ? ((data.layer_1_count / total) * 100).toFixed(1) : '0'
  const layer2Pct = total > 0 ? ((data.layer_2_count / total) * 100).toFixed(1) : '0'
  const layer3Pct = total > 0 ? ((data.layer_3_count / total) * 100).toFixed(1) : '0'

  return (
    <div className="bg-[hsl(var(--popover))] border border-[hsl(var(--border))] rounded-lg shadow-lg p-3 text-sm">
      <div className="font-semibold mb-2 text-[hsl(var(--popover-foreground))]">
        {weekToDate(data.week)}
      </div>
      <div className="space-y-1.5">
        <div className="flex items-center justify-between gap-4">
          <span className="flex items-center gap-1.5">
            <span
              className="w-2.5 h-2.5 rounded-sm"
              style={{ backgroundColor: LAYER_COLORS.layer_1 }}
            />
            <span className="text-[hsl(var(--muted-foreground))]">{data.layer_1_label}:</span>
          </span>
          <span className="text-[hsl(var(--popover-foreground))]">
            {data.layer_1_count} ({layer1Pct}%)
          </span>
        </div>
        <div className="flex items-center justify-between gap-4">
          <span className="flex items-center gap-1.5">
            <span
              className="w-2.5 h-2.5 rounded-sm"
              style={{ backgroundColor: LAYER_COLORS.layer_2 }}
            />
            <span className="text-[hsl(var(--muted-foreground))]">{data.layer_2_label}:</span>
          </span>
          <span className="text-[hsl(var(--popover-foreground))]">
            {data.layer_2_count} ({layer2Pct}%)
          </span>
        </div>
        <div className="flex items-center justify-between gap-4">
          <span className="flex items-center gap-1.5">
            <span
              className="w-2.5 h-2.5 rounded-sm"
              style={{ backgroundColor: LAYER_COLORS.layer_3 }}
            />
            <span className="text-[hsl(var(--muted-foreground))]">{data.layer_3_label}:</span>
          </span>
          <span className="text-[hsl(var(--popover-foreground))]">
            {data.layer_3_count} ({layer3Pct}%)
          </span>
        </div>
        <div className="border-t border-[hsl(var(--border))] pt-1.5 mt-1.5 space-y-0.5">
          <div className="flex justify-between text-[hsl(var(--popover-foreground))]">
            <span>{t('statistics.totalMRs')}:</span>
            <span className="font-medium">{data.mr_count}</span>
          </div>
          <div className="flex justify-between text-[hsl(var(--popover-foreground))]">
            <span>{t('statistics.avgRevisions')}:</span>
            <span className="font-medium">{data.avg_revisions}</span>
          </div>
        </div>
      </div>
    </div>
  )
}

/**
 * RevisionChart - MR修订次数分布图表
 * 展示每周MR修订次数的分层分布（堆叠图）
 */
export function RevisionChart({ data, loading }: RevisionChartProps) {
  const { t } = useTranslation()
  
  // Adjust x-axis interval based on data length to reduce label density
  // Show at most ~12 labels for readability
  const xAxisInterval = data.length <= 12 ? 0 : Math.ceil(data.length / 12) - 1

  if (loading) {
    return (
      <Card>
        <CardHeader>
          <CardTitle>{t('statistics.revision')}</CardTitle>
        </CardHeader>
        <CardContent>
          <Skeleton className="h-[340px] w-full" />
        </CardContent>
      </Card>
    )
  }

  if (!data || data.length === 0) {
    return (
      <Card>
        <CardHeader>
          <CardTitle>{t('statistics.revision')}</CardTitle>
        </CardHeader>
        <CardContent>
          <div className="flex flex-col items-center justify-center h-[340px] text-[hsl(var(--muted-foreground))]">
            <AlertCircle className="mb-2 h-8 w-8" />
            <p>{t('common.noData')}</p>
          </div>
        </CardContent>
      </Card>
    )
  }

  // Get labels from first data item (provided by backend)
  const labels = {
    layer_1: data[0]?.layer_1_label || t('statistics.revisionLayer1'),
    layer_2: data[0]?.layer_2_label || t('statistics.revisionLayer2'),
    layer_3: data[0]?.layer_3_label || t('statistics.revisionLayer3'),
  }

  return (
    <Card>
      <CardHeader className="pb-2">
        <CardTitle>{t('statistics.revision')}</CardTitle>
      </CardHeader>
      <CardContent className="[&_*:focus]:outline-none [&_*:focus-visible]:outline-none">
        <ResponsiveContainer width="100%" height={340}>
          <BarChart data={data} margin={{ bottom: 5, right: 20 }}>
            <CartesianGrid strokeDasharray="3 3" stroke="hsl(var(--border))" opacity={0.5} />
            <XAxis
              dataKey="week"
              stroke="hsl(var(--muted-foreground))"
              fontSize={11}
              tickLine={false}
              axisLine={{ stroke: 'hsl(var(--border))' }}
              tickFormatter={weekToDate}
              interval={xAxisInterval}
            />
            <YAxis
              stroke="hsl(var(--muted-foreground))"
              fontSize={11}
              tickLine={false}
              axisLine={{ stroke: 'hsl(var(--border))' }}
              label={{
                value: t('statistics.mrCount'),
                angle: -90,
                position: 'insideLeft',
                style: { fill: 'hsl(var(--muted-foreground))', fontSize: 11 },
              }}
            />
            <Tooltip
              content={<CustomTooltip />}
              cursor={{ fill: 'hsl(var(--muted))', opacity: 0.3 }}
              animationDuration={0}
            />
            <Legend
              wrapperStyle={{ paddingTop: '12px' }}
              iconType="square"
              formatter={(value: string) => (
                <span style={{ color: 'hsl(var(--foreground))', fontSize: 12 }}>{value}</span>
              )}
            />
            <Bar
              dataKey="layer_1_count"
              name={labels.layer_1}
              fill={LAYER_COLORS.layer_1}
              stackId="revision"
              radius={[4, 4, 0, 0]}
              isAnimationActive={false}
            />
            <Bar
              dataKey="layer_2_count"
              name={labels.layer_2}
              fill={LAYER_COLORS.layer_2}
              stackId="revision"
              radius={[4, 4, 0, 0]}
              isAnimationActive={false}
            />
            <Bar
              dataKey="layer_3_count"
              name={labels.layer_3}
              fill={LAYER_COLORS.layer_3}
              stackId="revision"
              radius={[4, 4, 0, 0]}
              isAnimationActive={false}
            />
          </BarChart>
        </ResponsiveContainer>
      </CardContent>
    </Card>
  )
}
