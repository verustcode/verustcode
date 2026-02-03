import { useTranslation } from 'react-i18next'
import { FileText, Code } from 'lucide-react'
import { Button } from '@/components/ui/button'
import { cn } from '@/lib/utils'

type EditorMode = 'form' | 'yaml'

interface ModeSwitchProps {
  mode: EditorMode
  onModeChange: (mode: EditorMode) => void
  disabled?: boolean
}

/**
 * Form/YAML mode switch component
 */
export function ModeSwitch({ mode, onModeChange, disabled }: ModeSwitchProps) {
  const { t } = useTranslation()

  return (
    <div className="inline-flex rounded-md border border-[hsl(var(--border))] p-0.5">
      {/* YAML button on left, Form button on right */}
      <Button
        variant="ghost"
        size="sm"
        onClick={() => onModeChange('yaml')}
        disabled={disabled}
        className={cn(
          'h-7 gap-1 px-2 text-xs',
          mode === 'yaml'
            ? 'bg-[hsl(var(--accent))] text-[hsl(var(--accent-foreground))]'
            : 'hover:bg-[hsl(var(--muted))]'
        )}
      >
        <Code className="h-3.5 w-3.5" />
        {t('common.yaml')}
      </Button>
      <Button
        variant="ghost"
        size="sm"
        onClick={() => onModeChange('form')}
        disabled={disabled}
        className={cn(
          'h-7 gap-1 px-2 text-xs',
          mode === 'form'
            ? 'bg-[hsl(var(--accent))] text-[hsl(var(--accent-foreground))]'
            : 'hover:bg-[hsl(var(--muted))]'
        )}
      >
        <FileText className="h-3.5 w-3.5" />
        {t('common.form')}
      </Button>
    </div>
  )
}
