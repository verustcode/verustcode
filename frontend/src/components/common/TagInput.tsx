import { useState, type KeyboardEvent } from 'react'
import { X, Plus } from 'lucide-react'
import { Input } from '@/components/ui/input'
import { Badge } from '@/components/ui/badge'
import { cn } from '@/lib/utils'

interface TagInputProps {
  value: string[]
  onChange?: (value: string[]) => void
  placeholder?: string
  className?: string
  disabled?: boolean
  suggestions?: string[]
}

/**
 * Tag input component for managing string arrays
 */
export function TagInput({
  value = [],
  onChange,
  placeholder = 'Add item...',
  className,
  disabled,
  suggestions = [],
}: TagInputProps) {
  const [inputValue, setInputValue] = useState('')
  const [showSuggestions, setShowSuggestions] = useState(false)

  const handleKeyDown = (e: KeyboardEvent<HTMLInputElement>) => {
    if (e.key === 'Enter' && inputValue.trim()) {
      e.preventDefault()
      addTag(inputValue.trim())
    } else if (e.key === 'Backspace' && !inputValue && value.length > 0) {
      // Remove last tag on backspace when input is empty
      removeTag(value.length - 1)
    }
  }

  const addTag = (tag: string) => {
    if (!value.includes(tag)) {
      onChange?.([...value, tag])
    }
    setInputValue('')
    setShowSuggestions(false)
  }

  const removeTag = (index: number) => {
    onChange?.(value.filter((_, i) => i !== index))
  }

  // Filter suggestions based on input
  const filteredSuggestions = suggestions.filter(
    (s) => s.toLowerCase().includes(inputValue.toLowerCase()) && !value.includes(s)
  )

  return (
    <div className={cn('relative', className)}>
      <div className="flex flex-wrap gap-2 rounded-md border border-[hsl(var(--input))] bg-transparent p-2">
        {/* Tags */}
        {value.map((tag, index) => (
          <Badge
            key={index}
            variant="secondary"
            className="flex items-center gap-1 pr-1"
          >
            {tag}
            {!disabled && (
              <button
                type="button"
                onClick={() => removeTag(index)}
                className="ml-1 rounded-full p-0.5 hover:bg-[hsl(var(--destructive))]/20"
              >
                <X className="h-3 w-3" />
              </button>
            )}
          </Badge>
        ))}

        {/* Input */}
        {!disabled && (
          <Input
            type="text"
            value={inputValue}
            onChange={(e) => {
              setInputValue(e.target.value)
              setShowSuggestions(true)
            }}
            onKeyDown={handleKeyDown}
            onFocus={() => setShowSuggestions(true)}
            onBlur={() => setTimeout(() => setShowSuggestions(false), 200)}
            placeholder={value.length === 0 ? placeholder : ''}
            className="flex-1 min-w-[120px] border-0 p-0 shadow-none focus-visible:ring-0"
          />
        )}
      </div>

      {/* Suggestions dropdown */}
      {showSuggestions && filteredSuggestions.length > 0 && (
        <div className="absolute z-10 mt-1 w-full rounded-md border border-[hsl(var(--border))] bg-[hsl(var(--popover))] p-1">
          {filteredSuggestions.slice(0, 8).map((suggestion) => (
            <button
              key={suggestion}
              type="button"
              onClick={() => addTag(suggestion)}
              className="flex w-full items-center gap-2 rounded-sm px-2 py-1.5 text-sm hover:bg-[hsl(var(--accent))]"
            >
              <Plus className="h-3 w-3" />
              {suggestion}
            </button>
          ))}
        </div>
      )}
    </div>
  )
}
