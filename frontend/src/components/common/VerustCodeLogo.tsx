import { cn } from '@/lib/utils'

interface VerustCodeLogoProps {
  className?: string
  size?: 'sm' | 'md' | 'lg' | 'xl'
  variant?: 'full' | 'icon'  // full: complete Logo, icon: icon only
  showText?: boolean         // Whether to show text (only effective when variant='full')
}

// Size mapping
const sizeMap = {
  sm: { icon: 24, text: 14 },
  md: { icon: 32, text: 18 },
  lg: { icon: 48, text: 24 },
  xl: { icon: 64, text: 32 },
}

/**
 * VerustCode Logo component
 * Uses /logo.svg as the single source of truth for the logo design.
 * Logo is served from frontend/public/logo.svg
 */
export function VerustCodeLogo({
  className,
  size = 'md',
  variant = 'icon',
  showText = false,
}: VerustCodeLogoProps) {
  const dimensions = sizeMap[size]

  return (
    <div className={cn('inline-flex items-center gap-2', className)}>
      {/* Logo from /admin/logo.svg - single source of truth */}
      <img
        src="/admin/logo.svg"
        width={dimensions.icon}
        height={dimensions.icon}
        alt="VerustCode Logo"
      />

      {/* Text part */}
      {variant === 'full' && showText && (
        <span
          className="font-bold text-[hsl(var(--primary))]"
          style={{ fontSize: dimensions.text }}
        >
          VerustCode
        </span>
      )}
    </div>
  )
}

/**
 * Pure SVG Logo (for export or direct embedding)
 * Uses /logo.svg as the single source of truth
 */
export function VerustCodeLogoSVG({ size = 64 }: { size?: number }) {
  return (
    <img
      src="/admin/logo.svg"
      width={size}
      height={size}
      alt="VerustCode Logo"
    />
  )
}

export default VerustCodeLogo











