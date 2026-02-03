import { useTranslation } from 'react-i18next'
import {
  BarChart,
  Bar,
  XAxis,
  YAxis,
  CartesianGrid,
  Tooltip,
  ResponsiveContainer,
} from 'recharts'
import { AlertCircle } from 'lucide-react'
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'
import { Skeleton } from '@/components/ui/skeleton'
import type { WeeklyMRCount, TimeRange } from '@/types/stats'

interface MRCountChartProps {
  data: WeeklyMRCount[]
  loading?: boolean
  timeRange?: TimeRange
}

// Modern color - Sky blue for MR count
const CHART_COLOR = '#0ea5e9'

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
 * MRCountChart - MR数目图表
 * 展示每周MR数量的柱状图
 */
export function MRCountChart({ data, loading }: MRCountChartProps) {
  const { t } = useTranslation()
  
  // Adjust x-axis interval based on data length to reduce label density
  // Show at most ~12 labels for readability
  const xAxisInterval = data.length <= 12 ? 0 : Math.ceil(data.length / 12) - 1

  if (loading) {
    return (
      <Card>
        <CardHeader>
          <CardTitle>{t('statistics.mrCount')}</CardTitle>
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
          <CardTitle>{t('statistics.mrCount')}</CardTitle>
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

  return (
    <Card>
      <CardHeader>
        <CardTitle>{t('statistics.mrCount')}</CardTitle>
      </CardHeader>
      <CardContent className="[&_*:focus]:outline-none [&_*:focus-visible]:outline-none">
        <ResponsiveContainer width="100%" height={340}>
          <BarChart data={data} margin={{ bottom: 5 }}>
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
                value: t('statistics.count'),
                angle: -90,
                position: 'insideLeft',
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
              labelFormatter={weekToDate}
            />
            <Bar
              dataKey="count"
              name={t('statistics.mrCount')}
              fill={CHART_COLOR}
              radius={[4, 4, 0, 0]}
              stroke="none"
              strokeWidth={0}
              isAnimationActive={false}
            />
          </BarChart>
        </ResponsiveContainer>
      </CardContent>
    </Card>
  )
}
