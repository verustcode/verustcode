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
import type { IssueSeverityStats } from '@/types/stats'

interface IssueSeverityChartProps {
  data: IssueSeverityStats[]
  loading?: boolean
}

// Severity colors - consistent with common severity level conventions
const SEVERITY_COLORS: Record<string, string> = {
  critical: '#dc2626', // red-600
  high: '#ea580c',     // orange-600
  medium: '#ca8a04',   // yellow-600
  low: '#2563eb',      // blue-600
  info: '#6b7280',     // gray-500
}

// Severity display order
const SEVERITY_ORDER = ['critical', 'high', 'medium', 'low', 'info']

/**
 * IssueSeverityChart - 问题严重程度分布饼图
 * 展示按严重程度分类的问题数量
 */
export function IssueSeverityChart({ data, loading }: IssueSeverityChartProps) {
  const { t } = useTranslation()

  if (loading) {
    return (
      <Card>
        <CardHeader className="flex flex-row items-center justify-between">
          <CardTitle>{t('statistics.issueSeverity')}</CardTitle>
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
          <CardTitle>{t('statistics.issueSeverity')}</CardTitle>
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

  // Sort data by severity order and translate labels
  const sortedData = [...data]
    .sort((a, b) => SEVERITY_ORDER.indexOf(a.severity) - SEVERITY_ORDER.indexOf(b.severity))
    .map(item => ({
      ...item,
      name: t(`statistics.severity.${item.severity}`, { defaultValue: item.severity }),
      color: SEVERITY_COLORS[item.severity] || '#9ca3af',
    }))

  return (
    <Card>
      <CardHeader className="flex flex-row items-center justify-between">
        <CardTitle>{t('statistics.issueSeverity')}</CardTitle>
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
              data={sortedData}
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
              {sortedData.map((entry, index) => (
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
