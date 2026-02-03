import { useTranslation } from 'react-i18next'
import { Badge } from '@/components/ui/badge'
import type { ReviewStatus, RuleStatus, RunStatus } from '@/types/review'

// Report status types
type ReportStatus = 'pending' | 'analyzing' | 'generating' | 'completed' | 'failed' | 'cancelled'

type Status = ReviewStatus | RuleStatus | RunStatus | ReportStatus

interface StatusBadgeProps {
  status: Status
  className?: string
  showPulse?: boolean
  type?: 'review' | 'report' // Add type to determine translation namespace
}

/**
 * Status badge component with appropriate colors
 */
export function StatusBadge({ status, className, showPulse = true, type = 'review' }: StatusBadgeProps) {
  const { t } = useTranslation()

  const getVariant = () => {
    switch (status) {
      case 'completed':
        return 'success'
      case 'running':
      case 'analyzing':
      case 'generating':
        return 'info'
      case 'pending':
        return 'secondary'
      case 'failed':
        return 'destructive'
      case 'cancelled':
        return 'warning'
      case 'skipped':
        return 'outline'
      default:
        return 'secondary'
    }
  }

  const isRunning = status === 'running' || status === 'analyzing' || status === 'generating'

  // Determine translation key based on type
  const getTranslationKey = () => {
    if (type === 'report' || status === 'analyzing' || status === 'generating') {
      return `reports.status.${status}`
    }
    return `reviews.status.${status}`
  }

  return (
    <div className="flex items-center gap-2">
      {/* 蓝色闪烁小圆点 - 仅在运行状态时显示 */}
      {isRunning && showPulse && (
        <span className="relative flex h-2 w-2">
          <span className="animate-ping absolute inline-flex h-full w-full rounded-full bg-blue-500 opacity-75"></span>
          <span className="relative inline-flex rounded-full h-2 w-2 bg-blue-500"></span>
        </span>
      )}
      <Badge
        variant={getVariant()}
        className={className}
      >
        {t(getTranslationKey())}
      </Badge>
    </div>
  )
}
