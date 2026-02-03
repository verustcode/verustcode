import * as React from 'react'
import { useTranslation } from 'react-i18next'
import { useToast } from '@/hooks/useToast'
import {
  Toast,
  ToastClose,
  ToastDescription,
  ToastProvider,
  ToastTitle,
  ToastViewport,
} from '@/components/ui/toast'
import { Copy, Check, CheckCircle2, AlertTriangle, XCircle, Info } from 'lucide-react'

// Get icon component based on toast variant
function getVariantIcon(variant?: string) {
  switch (variant) {
    case 'success':
      return <CheckCircle2 className="h-4 w-4 text-green-500 shrink-0" />
    case 'warning':
      return <AlertTriangle className="h-4 w-4 text-amber-500 shrink-0" />
    case 'error':
    case 'destructive':
      return <XCircle className="h-4 w-4 text-red-500 shrink-0" />
    case 'info':
      return <Info className="h-4 w-4 text-blue-500 shrink-0" />
    default:
      return null
  }
}

export function Toaster() {
  const { t } = useTranslation()
  const { toasts } = useToast()
  // Track which toast content has been copied
  const [copiedId, setCopiedId] = React.useState<string | null>(null)

  // Handle copy to clipboard
  const handleCopy = async (id: string, title?: React.ReactNode, description?: React.ReactNode) => {
    // Build text content to copy
    const textParts: string[] = []
    if (title) {
      textParts.push(typeof title === 'string' ? title : String(title))
    }
    if (description) {
      textParts.push(typeof description === 'string' ? description : String(description))
    }
    const textToCopy = textParts.join('\n')

    try {
      await navigator.clipboard.writeText(textToCopy)
      setCopiedId(id)
      // Reset copied state after 2 seconds
      setTimeout(() => setCopiedId(null), 2000)
    } catch (err) {
      console.error('Failed to copy toast content:', err)
    }
  }

  return (
    <ToastProvider>
      {toasts.map(function ({ id, title, description, action, variant, ...props }) {
        const isCopied = copiedId === id
        const variantIcon = getVariantIcon(variant ?? undefined)
        return (
          <Toast key={id} variant={variant} {...props}>
            {/* Variant icon */}
            {variantIcon}
            <div 
              className="grid gap-1 flex-1 cursor-pointer select-text"
              onClick={() => handleCopy(id, title, description)}
              title={t('common.clickToCopy')}
            >
              {title && <ToastTitle>{title}</ToastTitle>}
              {description && (
                <ToastDescription>{description}</ToastDescription>
              )}
            </div>
            {/* Copy indicator icon */}
            <div className="flex items-center gap-2">
              <button
                onClick={(e) => {
                  e.stopPropagation()
                  handleCopy(id, title, description)
                }}
                className="p-1 rounded hover:bg-black/10 dark:hover:bg-white/10 transition-colors"
                title={t('common.copy')}
              >
                {isCopied ? (
                  <Check className="h-4 w-4 text-green-500" />
                ) : (
                  <Copy className="h-4 w-4 opacity-50 hover:opacity-100" />
                )}
              </button>
              {action}
            </div>
            <ToastClose />
          </Toast>
        )
      })}
      <ToastViewport />
    </ToastProvider>
  )
}












