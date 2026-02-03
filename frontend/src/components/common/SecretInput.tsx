import { useState } from 'react'
import { Eye, EyeOff } from 'lucide-react'
import { Input } from '@/components/ui/input'
import { Button } from '@/components/ui/button'
import { cn } from '@/lib/utils'

interface SecretInputProps {
  value: string
  onChange?: (value: string) => void
  placeholder?: string
  className?: string
  disabled?: boolean
}

/**
 * Secret input component with show/hide toggle
 */
export function SecretInput({
  value,
  onChange,
  placeholder,
  className,
  disabled,
}: SecretInputProps) {
  const [showSecret, setShowSecret] = useState(false)

  return (
    <div className={cn('relative', className)}>
      <Input
        type={showSecret ? 'text' : 'password'}
        value={value}
        onChange={(e) => onChange?.(e.target.value)}
        placeholder={placeholder}
        disabled={disabled}
        className="pr-10"
      />
      <Button
        type="button"
        variant="ghost"
        size="icon"
        className="absolute right-0 top-0 h-full px-3 hover:bg-transparent"
        onClick={() => setShowSecret(!showSecret)}
        disabled={disabled}
      >
        {showSecret ? (
          <EyeOff className="h-4 w-4 text-[hsl(var(--muted-foreground))]" />
        ) : (
          <Eye className="h-4 w-4 text-[hsl(var(--muted-foreground))]" />
        )}
      </Button>
    </div>
  )
}
