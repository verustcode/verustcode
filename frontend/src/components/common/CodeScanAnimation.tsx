import { cn } from '@/lib/utils'

interface CodeScanAnimationProps {
  className?: string
  size?: 'sm' | 'md' | 'lg'
}

/**
 * Code scan animation component
 * Displays animated code lines being scanned for analysis visualization
 */
export function CodeScanAnimation({ className, size = 'md' }: CodeScanAnimationProps) {
  const sizeConfig = {
    sm: { width: 120, height: 80, lineHeight: 6, gap: 8 },
    md: { width: 180, height: 120, lineHeight: 8, gap: 10 },
    lg: { width: 240, height: 160, lineHeight: 10, gap: 12 },
  }

  const config = sizeConfig[size]
  const { width, height, lineHeight, gap } = config

  // Code line widths (percentage of total width)
  const codeLines = [0.85, 0.6, 0.75, 0.5, 0.9, 0.65]
  const startX = 10
  const maxLineWidth = width - 20

  return (
    <div className={cn('relative', className)}>
      <svg
        width={width}
        height={height}
        viewBox={`0 0 ${width} ${height}`}
        fill="none"
        xmlns="http://www.w3.org/2000/svg"
        className="overflow-visible"
      >
        {/* Background panel */}
        <rect
          x="0"
          y="0"
          width={width}
          height={height}
          rx="8"
          className="fill-[hsl(var(--muted))]"
        />

        {/* Code lines */}
        {codeLines.map((widthPercent, index) => {
          const y = 15 + index * (lineHeight + gap)
          const lineWidth = maxLineWidth * widthPercent
          // Stagger animation delay for each line
          const animationDelay = `${index * 0.3}s`

          return (
            <rect
              key={index}
              x={startX}
              y={y}
              width={lineWidth}
              height={lineHeight}
              rx={lineHeight / 2}
              className="fill-[hsl(var(--muted-foreground)/0.2)] code-line-highlight"
              style={{
                animationDelay,
              }}
            />
          )
        })}

        {/* Scan line with glow effect */}
        <g className="scan-line-group">
          {/* Glow effect */}
          <rect
            x="0"
            y="-4"
            width={width}
            height="12"
            className="fill-[hsl(var(--primary)/0.15)]"
          />
          {/* Main scan line */}
          <rect
            x="0"
            y="0"
            width={width}
            height="2"
            className="fill-[hsl(var(--primary))]"
          />
        </g>
      </svg>
    </div>
  )
}

