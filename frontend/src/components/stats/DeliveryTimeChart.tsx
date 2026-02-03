import { useTranslation } from 'react-i18next'
import {
  LineChart,
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
import type { WeeklyDeliveryTime, TimeRange } from '@/types/stats'

interface DeliveryTimeChartProps {
  data: WeeklyDeliveryTime[]
  loading?: boolean
  timeRange?: TimeRange
}

// Modern color palette - distinct and easy to differentiate
const COLORS = {
  p50: '#0ea5e9',  // Sky blue - median (most common)
  p90: '#f59e0b',  // Amber - warning level
  p95: '#ef4444',  // Red - critical level
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
 * Format time intelligently based on value
 * Returns appropriate unit (minutes/hours/days) based on the value
 */
function formatTime(hours: number, t: (key: string) => string): { value: string; unit: string } {
  if (hours < 1) {
    // Less than 1 hour, show minutes
    const minutes = Math.round(hours * 60)
    return { value: minutes.toFixed(0), unit: t('statistics.minutes') }
  } else if (hours <= 48) {
    // 1-48 hours, show hours
    return { value: hours.toFixed(1), unit: t('statistics.hours') }
  } else {
    // More than 48 hours, show days
    const days = hours / 24
    return { value: days.toFixed(1), unit: t('statistics.days') }
  }
}

/**
 * DeliveryTimeChart - MR交付时间图表
 * 展示P50、P90、P95百分位数的时间趋势
 */
export function DeliveryTimeChart({ data, loading }: DeliveryTimeChartProps) {
  const { t } = useTranslation()
  
  // Adjust x-axis interval based on data length to reduce label density
  // Show at most ~12 labels for readability
  const xAxisInterval = data.length <= 12 ? 0 : Math.ceil(data.length / 12) - 1

  // Format Y-axis value (number only, no unit)
  const formatYAxisValue = (hours: number) => {
    return formatTime(hours, t).value
  }

  // Format tooltip value (with unit)
  const tooltipFormatter = (value: number | undefined, name: string | undefined) => {
    if (value === undefined) {
      return ['', name ?? '']
    }
    const formatted = formatTime(value, t)
    return [`${formatted.value} ${formatted.unit}`, name ?? '']
  }

  if (loading) {
    return (
      <Card>
        <CardHeader>
          <CardTitle>{t('statistics.deliveryTime')}</CardTitle>
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
          <CardTitle>{t('statistics.deliveryTime')}</CardTitle>
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
        <CardTitle>{t('statistics.deliveryTime')}</CardTitle>
      </CardHeader>
      <CardContent className="[&_*:focus]:outline-none [&_*:focus-visible]:outline-none">
        <ResponsiveContainer width="100%" height={340}>
          <LineChart data={data} margin={{ top: 5, right: 20, left: 10, bottom: 5 }}>
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
              tickFormatter={formatYAxisValue}
              label={{
                value: t('statistics.deliveryTimeUnit'),
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
              formatter={tooltipFormatter}
              labelFormatter={weekToDate}
            />
            <Legend 
              wrapperStyle={{ fontSize: 12, paddingTop: 10 }}
            />
            <Line
              type="monotone"
              dataKey="p50"
              name="P50"
              stroke={COLORS.p50}
              strokeWidth={2}
              dot={{ fill: COLORS.p50, strokeWidth: 0, r: 3 }}
              activeDot={{ r: 5, strokeWidth: 0 }}
              isAnimationActive={false}
            />
            <Line
              type="monotone"
              dataKey="p90"
              name="P90"
              stroke={COLORS.p90}
              strokeWidth={2}
              dot={{ fill: COLORS.p90, strokeWidth: 0, r: 3 }}
              activeDot={{ r: 5, strokeWidth: 0 }}
              isAnimationActive={false}
            />
            <Line
              type="monotone"
              dataKey="p95"
              name="P95"
              stroke={COLORS.p95}
              strokeWidth={2}
              dot={{ fill: COLORS.p95, strokeWidth: 0, r: 3 }}
              activeDot={{ r: 5, strokeWidth: 0 }}
              isAnimationActive={false}
            />
          </LineChart>
        </ResponsiveContainer>
      </CardContent>
    </Card>
  )
}
