import { useTranslation } from 'react-i18next'
import { Link } from 'react-router-dom'
import {
  PieChart,
  Pie,
  Cell,
  Tooltip,
  ResponsiveContainer,
  Legend,
} from 'recharts'
import { AlertCircle, ArrowRight } from 'lucide-react'
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'
import { Button } from '@/components/ui/button'
import { Skeleton } from '@/components/ui/skeleton'
import type { IssueCategoryStats } from '@/types/stats'

interface IssueCategoryChartProps {
  data: IssueCategoryStats[]
  loading?: boolean
}

// Modern color palette for categories
const CATEGORY_COLORS = [
  '#8b5cf6', // violet-500
  '#06b6d4', // cyan-500
  '#10b981', // emerald-500
  '#f59e0b', // amber-500
  '#ef4444', // red-500
  '#ec4899', // pink-500
  '#6366f1', // indigo-500
  '#14b8a6', // teal-500
  '#f97316', // orange-500
  '#84cc16', // lime-500
  '#3b82f6', // blue-500
  '#a855f7', // purple-500
]

/**
 * IssueCategoryChart - 问题分类分布饼图
 * 展示按分类归类的问题数量
 */
export function IssueCategoryChart({ data, loading }: IssueCategoryChartProps) {
  const { t } = useTranslation()

  if (loading) {
    return (
      <Card>
        <CardHeader className="flex flex-row items-center justify-between">
          <CardTitle>{t('statistics.issueCategory')}</CardTitle>
          <Button variant="ghost" size="sm" asChild>
            <Link to="/admin/findings">
              {t('common.view')}
              <ArrowRight className="ml-1 h-4 w-4" />
            </Link>
          </Button>
        </CardHeader>
        <CardContent>
          <Skeleton className="h-[300px] w-full" />
        </CardContent>
      </Card>
    )
  }

  // Check if data is empty or all zeros
  const totalCount = data?.reduce((sum, item) => sum + item.count, 0) || 0
  if (!data || data.length === 0 || totalCount === 0) {
    return (
      <Card>
        <CardHeader className="flex flex-row items-center justify-between">
          <CardTitle>{t('statistics.issueCategory')}</CardTitle>
          <Button variant="ghost" size="sm" asChild>
            <Link to="/admin/findings">
              {t('common.view')}
              <ArrowRight className="ml-1 h-4 w-4" />
            </Link>
          </Button>
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

  // Translate category labels and assign colors
  const chartData = data.map((item, index) => ({
    ...item,
    name: t(`statistics.category.${item.category}`, { defaultValue: item.category }),
    color: CATEGORY_COLORS[index % CATEGORY_COLORS.length],
  }))

  return (
    <Card>
      <CardHeader className="flex flex-row items-center justify-between">
        <CardTitle>{t('statistics.issueCategory')}</CardTitle>
        <Button variant="ghost" size="sm" asChild>
          <Link to="/admin/findings">
            {t('common.view')}
            <ArrowRight className="ml-1 h-4 w-4" />
          </Link>
        </Button>
      </CardHeader>
      <CardContent className="[&_*:focus]:outline-none [&_*:focus-visible]:outline-none">
        <ResponsiveContainer width="100%" height={300}>
          <PieChart>
            <Pie
              data={chartData}
              dataKey="count"
              nameKey="name"
              cx="50%"
              cy="50%"
              outerRadius={100}
              innerRadius={50}
              paddingAngle={2}
              isAnimationActive={false}
              label={({ name, percent }) => `${name} ${((percent ?? 0) * 100).toFixed(0)}%`}
              labelLine={{ stroke: 'hsl(var(--muted-foreground))', strokeWidth: 1 }}
            >
              {chartData.map((entry, index) => (
                <Cell key={`cell-${index}`} fill={entry.color} stroke="none" />
              ))}
            </Pie>
            <Tooltip
              animationDuration={0}
              contentStyle={{
                backgroundColor: 'hsl(var(--popover))',
                border: '1px solid hsl(var(--border))',
                borderRadius: '8px',
                boxShadow: '0 4px 6px -1px rgb(0 0 0 / 0.1)',
                fontSize: 12,
                color: 'hsl(var(--popover-foreground))',
              }}
              labelStyle={{ color: 'hsl(var(--popover-foreground))', fontWeight: 500 }}
              itemStyle={{ color: 'hsl(var(--popover-foreground))' }}
              formatter={(value, name) => [
                `${value ?? 0} ${t('statistics.issues')}`,
                String(name),
              ]}
            />
            <Legend
              verticalAlign="bottom"
              height={36}
              formatter={(value) => (
                <span style={{ color: 'hsl(var(--foreground))', fontSize: 12 }}>
                  {value}
                </span>
              )}
            />
          </PieChart>
        </ResponsiveContainer>
      </CardContent>
    </Card>
  )
}
