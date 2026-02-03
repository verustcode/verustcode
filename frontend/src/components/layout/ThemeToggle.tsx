import { Moon, Sun } from 'lucide-react'
import { useTranslation } from 'react-i18next'
import { Button } from '@/components/ui/button'
import { useTheme } from '@/hooks/useTheme.tsx'

/**
 * Theme toggle button component
 */
export function ThemeToggle() {
  const { t } = useTranslation()
  const { toggleTheme, isDark } = useTheme()

  return (
    <Button
      variant="ghost"
      size="icon"
      onClick={toggleTheme}
      className="h-9 w-9"
      title={t('header.theme')}
    >
      {isDark ? (
        <Sun className="h-4 w-4" />
      ) : (
        <Moon className="h-4 w-4" />
      )}
      <span className="sr-only">{t('header.theme')}</span>
    </Button>
  )
}
