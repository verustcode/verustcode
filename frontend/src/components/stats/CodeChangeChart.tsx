import { useMemo } from 'react'
import { useTranslation } from 'react-i18next'
import {
  ComposedChart,
  Bar,
  Line,
  XAxis,
  YAxis,
  CartesianGrid,
  Tooltip,
  Legend,
  ResponsiveContainer,
} from 'recharts'
import { AlertCircle } from 'lucide-react'
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'
import { Skeleton } from '@/components/ui/skeleton'
import type { WeeklyCodeChange, WeeklyFileChange, TimeRange } from '@/types/stats'

interface CodeChangeChartProps {
  data: WeeklyCodeChange[]
  fileChangeData?: WeeklyFileChange[]
  loading?: boolean
  timeRange?: TimeRange
}

// Modern color palette - semantic colors for add/delete/files
const COLORS = {
  added: '#10b981',   // Emerald - positive/add
  deleted: '#f43f5e', // Rose - negative/delete
  files: '#f59e0b',   // Amber - file changes
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
 * Format number with thousand separators
 * e.g., 1234567 -> "1,234,567"
 */
function formatNumber(value: number): string {
  return value.toLocaleString('en-US')
}

// Merged data type for chart
interface MergedWeeklyData {
  week: string
  displayDate: string
  lines_added: number
  lines_deleted: number
  files_changed: number
}

/**
 * CodeChangeChart - 代码变动图表
 * 展示代码增加和删除的堆叠柱状图，以及文件变更数的折线图
 */
export function CodeChangeChart({ data, fileChangeData, loading }: CodeChangeChartProps) {
  const { t } = useTranslation()
  
  // Adjust x-axis interval based on data length to reduce label density
  // Show at most ~12 labels for readability
  const xAxisInterval = data.length <= 12 ? 0 : Math.ceil(data.length / 12) - 1

  // Merge code change and file change data
  const mergedData = useMemo<MergedWeeklyData[]>(() => {
    if (!data || data.length === 0) return []

    // Create a map of file changes by week
    const fileChangeMap = new Map<string, number>()
    if (fileChangeData) {
      fileChangeData.forEach((item) => {
        fileChangeMap.set(item.week, item.files_changed)
      })
    }

    return data.map((item) => ({
      week: item.week,
      displayDate: weekToDate(item.week),
      lines_added: item.lines_added,
      lines_deleted: item.lines_deleted,
      files_changed: fileChangeMap.get(item.week) || 0,
    }))
  }, [data, fileChangeData])

  if (loading) {
    return (
      <Card>
        <CardHeader>
          <CardTitle>{t('statistics.codeChange')}</CardTitle>
        </CardHeader>
        <CardContent>
          <Skeleton className="h-[300px] w-full" />
        </CardContent>
      </Card>
    )
  }

  if (!data || data.length === 0) {
    return (
      <Card>
        <CardHeader>
          <CardTitle>{t('statistics.codeChange')}</CardTitle>
        </CardHeader>
        <CardContent>
          <div className="flex flex-col items-center justify-center h-[300px] text-[hsl(var(--muted-foreground))]">
            <AlertCircle className="mb-2 h-8 w-8" />
            <p>{t('common.noData')}</p>
          </div>
        </CardContent>
      </Card>
    )
  }

  return (
    <Card>
      <CardHeader>
        <CardTitle>{t('statistics.codeChange')}</CardTitle>
      </CardHeader>
      <CardContent className="[&_*:focus]:outline-none [&_*:focus-visible]:outline-none">
        <ResponsiveContainer width="100%" height={340}>
          <ComposedChart data={mergedData} margin={{ bottom: 5 }}>
            <CartesianGrid strokeDasharray="3 3" stroke="hsl(var(--border))" opacity={0.5} />
            <XAxis
              dataKey="displayDate"
              stroke="hsl(var(--muted-foreground))"
              fontSize={11}
              tickLine={false}
              axisLine={{ stroke: 'hsl(var(--border))' }}
              interval={xAxisInterval}
            />
            <YAxis
              yAxisId="left"
              stroke="hsl(var(--muted-foreground))"
              fontSize={11}
              tickLine={false}
              axisLine={{ stroke: 'hsl(var(--border))' }}
              tickFormatter={formatNumber}
              label={{
                value: t('statistics.lines'),
                angle: -90,
                position: 'insideLeft',
                style: { fill: 'hsl(var(--muted-foreground))', fontSize: 11 },
              }}
            />
            <YAxis
              yAxisId="right"
              orientation="right"
              stroke="hsl(var(--muted-foreground))"
              fontSize={11}
              tickLine={false}
              axisLine={{ stroke: 'hsl(var(--border))' }}
              tickFormatter={formatNumber}
              label={{
                value: t('statistics.filesCount'),
                angle: 90,
                position: 'insideRight',
                style: { fill: 'hsl(var(--muted-foreground))', fontSize: 11 },
              }}
            />
            <Tooltip
              animationDuration={0}
              cursor={false}
              contentStyle={{
                backgroundColor: 'hsl(var(--popover))',
                border: '1px solid hsl(var(--border))',
                borderRadius: '8px',
                boxShadow: '0 4px 6px -1px rgb(0 0 0 / 0.1)',
                fontSize: 12,
              }}
              labelStyle={{ color: 'hsl(var(--popover-foreground))', fontWeight: 500 }}
              formatter={(value) => formatNumber(value as number)}
              labelFormatter={(label) => weekToDate(String(label))}
            />
            <Legend 
              wrapperStyle={{ fontSize: 12, paddingTop: 10 }}
            />
            <Bar
              yAxisId="left"
              dataKey="lines_deleted"
              name={t('statistics.linesDeleted')}
              fill={COLORS.deleted}
              stackId="code"
              stroke="none"
              strokeWidth={0}
              isAnimationActive={false}
            />
            <Bar
              yAxisId="left"
              dataKey="lines_added"
              name={t('statistics.linesAdded')}
              fill={COLORS.added}
              stackId="code"
              radius={[4, 4, 0, 0]}
              stroke="none"
              strokeWidth={0}
              isAnimationActive={false}
            />
            <Line
              yAxisId="right"
              type="monotone"
              dataKey="files_changed"
              name={t('statistics.filesChanged')}
              stroke={COLORS.files}
              strokeWidth={2}
              dot={{ fill: COLORS.files, strokeWidth: 0, r: 3 }}
              activeDot={{ r: 5, strokeWidth: 0 }}
              isAnimationActive={false}
            />
          </ComposedChart>
        </ResponsiveContainer>
      </CardContent>
    </Card>
  )
}
