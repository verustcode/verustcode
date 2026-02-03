import { Plus, X, GripVertical } from 'lucide-react'
import { Input } from '@/components/ui/input'
import { Button } from '@/components/ui/button'
import { Textarea } from '@/components/ui/textarea'
import { cn } from '@/lib/utils'

interface ListInputProps {
  value: string[]
  onChange: (value: string[]) => void
  placeholder?: string
  className?: string
  disabled?: boolean
  multiline?: boolean
}

/**
 * List input component for managing ordered string arrays
 */
export function ListInput({
  value = [],
  onChange,
  placeholder = 'Enter value...',
  className,
  disabled,
  multiline = false,
}: ListInputProps) {
  const addItem = () => {
    onChange([...value, ''])
  }

  const updateItem = (index: number, newValue: string) => {
    const updated = [...value]
    updated[index] = newValue
    onChange(updated)
  }

  const removeItem = (index: number) => {
    onChange(value.filter((_, i) => i !== index))
  }

  const InputComponent = multiline ? Textarea : Input

  return (
    <div className={cn('space-y-2', className)}>
      {value.map((item, index) => (
        <div key={index} className="flex items-start gap-2">
          <div className="flex h-9 items-center text-[hsl(var(--muted-foreground))]">
            <GripVertical className="h-4 w-4" />
          </div>
          <InputComponent
            value={item}
            onChange={(e) => updateItem(index, e.target.value)}
            placeholder={placeholder}
            disabled={disabled}
            className="flex-1"
          />
          {!disabled && (
            <Button
              type="button"
              variant="ghost"
              size="icon"
              onClick={() => removeItem(index)}
              className="h-9 w-9 shrink-0 text-[hsl(var(--destructive))]"
            >
              <X className="h-4 w-4" />
            </Button>
          )}
        </div>
      ))}

      {!disabled && (
        <Button
          type="button"
          variant="outline"
          size="sm"
          onClick={addItem}
          className="w-full"
        >
          <Plus className="mr-2 h-4 w-4" />
          Add Item
        </Button>
      )}
    </div>
  )
}
