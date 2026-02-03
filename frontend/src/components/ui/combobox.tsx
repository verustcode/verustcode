import * as React from 'react'
import { Check, ChevronsUpDown } from 'lucide-react'
import { cn } from '@/lib/utils'
import { Input } from '@/components/ui/input'
import { Button } from '@/components/ui/button'

export interface ComboboxOption {
  value: string
  label: string
}

export interface ComboboxProps {
  options: ComboboxOption[]
  value?: string
  onValueChange?: (value: string) => void
  placeholder?: string
  className?: string
  disabled?: boolean
  filterFunction?: (option: ComboboxOption, search: string) => boolean
  allowCustomValue?: boolean // Allow users to input custom values not in the options list
}

const defaultFilter = (option: ComboboxOption, search: string) => {
  const searchLower = search.toLowerCase()
  return (
    option.value.toLowerCase().includes(searchLower) ||
    option.label.toLowerCase().includes(searchLower)
  )
}

export function Combobox({
  options,
  value,
  onValueChange,
  placeholder = 'Select or type...',
  className,
  disabled = false,
  filterFunction = defaultFilter,
  allowCustomValue = false,
}: ComboboxProps) {
  const [open, setOpen] = React.useState(false)
  const [search, setSearch] = React.useState('')
  const inputRef = React.useRef<HTMLInputElement>(null)
  const containerRef = React.useRef<HTMLDivElement>(null)

  // Sync search with value when allowCustomValue is true
  React.useEffect(() => {
    if (allowCustomValue && value !== undefined && !open) {
      setSearch(value)
    }
  }, [value, allowCustomValue, open])

  // Filter options based on search
  const filteredOptions = React.useMemo(() => {
    if (!search) return options
    return options.filter((option) => filterFunction(option, search))
  }, [options, search, filterFunction])

  // Find selected option
  const selectedOption = options.find((opt) => opt.value === value)

  // Handle click outside
  React.useEffect(() => {
    const handleClickOutside = (event: MouseEvent) => {
      if (
        containerRef.current &&
        !containerRef.current.contains(event.target as Node)
      ) {
        setOpen(false)
      }
    }

    if (open) {
      document.addEventListener('mousedown', handleClickOutside)
      return () => {
        document.removeEventListener('mousedown', handleClickOutside)
      }
    }
  }, [open])

  const handleSelect = (optionValue: string) => {
    onValueChange?.(optionValue)
    setSearch('')
    setOpen(false)
  }

  const handleInputChange = (e: React.ChangeEvent<HTMLInputElement>) => {
    const newValue = e.target.value
    setSearch(newValue)
    if (!open) {
      setOpen(true)
    }
    // When allowCustomValue is true, update parent component value in real-time
    if (allowCustomValue) {
      onValueChange?.(newValue)
    }
  }

  const handleInputFocus = () => {
    setOpen(true)
    // When allowCustomValue is true, initialize search with current value
    if (allowCustomValue && value) {
      setSearch(value)
    }
  }

  const handleKeyDown = (e: React.KeyboardEvent<HTMLInputElement>) => {
    if (e.key === 'Enter') {
      e.preventDefault()
      if (allowCustomValue && search) {
        // Use current input value
        onValueChange?.(search)
        setOpen(false)
      } else if (filteredOptions.length > 0) {
        // Select first filtered option
        handleSelect(filteredOptions[0].value)
      }
    } else if (e.key === 'Escape') {
      setOpen(false)
      inputRef.current?.blur()
    }
  }

  return (
    <div ref={containerRef} className={cn('relative', className)}>
      <div className="relative">
        <Input
          ref={inputRef}
          type="text"
          value={allowCustomValue ? (value || '') : (open ? search : selectedOption?.label || value || '')}
          onChange={handleInputChange}
          onFocus={handleInputFocus}
          onKeyDown={handleKeyDown}
          placeholder={placeholder}
          disabled={disabled}
          className="pr-8"
        />
        <Button
          type="button"
          variant="ghost"
          size="icon"
          className="absolute right-0 top-0 h-full px-2"
          onClick={() => {
            if (!disabled) {
              setOpen(!open)
              if (!open) {
                inputRef.current?.focus()
              }
            }
          }}
          disabled={disabled}
        >
          <ChevronsUpDown className="h-4 w-4 opacity-50" />
        </Button>
      </div>
      {open && filteredOptions.length > 0 && (
        <div className="absolute z-50 mt-1 w-full rounded-md border border-[hsl(var(--border))] bg-[hsl(var(--background))] shadow-md">
          <div className="max-h-[300px] overflow-auto p-1">
            {filteredOptions.map((option) => (
              <div
                key={option.value}
                className={cn(
                  'relative flex cursor-pointer select-none items-center rounded-sm px-2 py-1.5 text-sm outline-none hover:bg-[hsl(var(--accent))] hover:text-[hsl(var(--accent-foreground))]',
                  value === option.value && 'bg-[hsl(var(--accent))] text-[hsl(var(--accent-foreground))]'
                )}
                onClick={() => handleSelect(option.value)}
              >
                <Check
                  className={cn(
                    'mr-2 h-4 w-4',
                    value === option.value ? 'opacity-100' : 'opacity-0'
                  )}
                />
                {option.label}
              </div>
            ))}
          </div>
        </div>
      )}
      {open && filteredOptions.length === 0 && search && (
        <div className="absolute z-50 mt-1 w-full rounded-md border border-[hsl(var(--border))] bg-[hsl(var(--background))] shadow-md p-4 text-sm text-[hsl(var(--muted-foreground))]">
          {allowCustomValue ? (
            <div className="space-y-1">
              <div>No matches found.</div>
              <div className="text-xs opacity-75">Press Enter to use current input</div>
            </div>
          ) : (
            'No results found.'
          )}
        </div>
      )}
    </div>
  )
}

