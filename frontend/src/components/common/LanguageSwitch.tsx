import { useState, useRef, useEffect } from 'react'
import { useTranslation } from 'react-i18next'
import { Globe } from 'lucide-react'
import { Button } from '@/components/ui/button'
import { cn } from '@/lib/utils'

const languages = [
  { code: 'en', name: 'English' },
  { code: 'zh', name: '中文' },
]

/**
 * Language switch component with custom popover menu
 * Prevents accidental language changes
 */
export function LanguageSwitch() {
  const { i18n } = useTranslation()
  const [open, setOpen] = useState(false)
  const menuRef = useRef<HTMLDivElement>(null)
  const buttonRef = useRef<HTMLButtonElement>(null)

  const currentLang = languages.find((l) => l.code === i18n.language) || languages[0]

  // Close menu when clicking outside
  useEffect(() => {
    if (!open) return

    const handleClickOutside = (event: MouseEvent) => {
      if (
        menuRef.current &&
        buttonRef.current &&
        !menuRef.current.contains(event.target as Node) &&
        !buttonRef.current.contains(event.target as Node)
      ) {
        setOpen(false)
      }
    }

    document.addEventListener('mousedown', handleClickOutside)
    return () => document.removeEventListener('mousedown', handleClickOutside)
  }, [open])

  const handleLanguageSelect = (langCode: string) => {
    // Only change if it's different from current language
    if (langCode !== i18n.language) {
      i18n.changeLanguage(langCode)
    }
    setOpen(false)
  }

  return (
    <div className="relative">
      <Button
        ref={buttonRef}
        variant="ghost"
        size="icon"
        className="h-9 w-9"
        type="button"
        onClick={(e) => {
          e.stopPropagation()
          setOpen(!open)
        }}
      >
        <Globe className="h-4 w-4" />
        <span className="sr-only">Switch language</span>
      </Button>

      {open && (
        <div
          ref={menuRef}
          className="absolute right-0 top-full z-50 mt-2 w-32 overflow-hidden rounded-md border border-[hsl(var(--border))] bg-[hsl(var(--popover))] p-1 text-[hsl(var(--popover-foreground))] shadow-md"
        >
          {languages.map((lang) => (
            <button
              key={lang.code}
              type="button"
              onClick={(e) => {
                e.preventDefault()
                e.stopPropagation()
                handleLanguageSelect(lang.code)
              }}
              className={cn(
                'relative flex w-full cursor-default select-none items-center rounded-sm px-2 py-1.5 text-sm outline-none transition-colors hover:bg-[hsl(var(--accent))] focus:bg-[hsl(var(--accent))]',
                lang.code === currentLang.code && 'text-[hsl(var(--primary))] font-medium'
              )}
            >
              {lang.name}
            </button>
          ))}
        </div>
      )}
    </div>
  )
}
