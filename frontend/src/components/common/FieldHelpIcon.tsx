import { Info } from 'lucide-react'
import {
  Tooltip,
  TooltipContent,
  TooltipTrigger,
} from '@/components/ui/tooltip'

interface FieldHelpIconProps {
  content: string
}

/**
 * Help icon with tooltip for form field descriptions
 */
export function FieldHelpIcon({ content }: FieldHelpIconProps) {
  return (
    <Tooltip delayDuration={200}>
      <TooltipTrigger asChild>
        <Info className="h-3.5 w-3.5 text-[hsl(var(--muted-foreground))] hover:text-[hsl(var(--foreground))] cursor-help transition-colors" />
      </TooltipTrigger>
      <TooltipContent side="top" sideOffset={6}>
        {content}
      </TooltipContent>
    </Tooltip>
  )
}

