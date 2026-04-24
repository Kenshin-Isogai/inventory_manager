import { useMemo, useState } from 'react'
import { Check, ChevronsUpDown, Plus, Search } from 'lucide-react'

import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Popover, PopoverContent, PopoverTrigger } from '@/components/ui/popover'
import { cn } from '@/lib/utils'

export type ItemComboboxOption = {
  itemId: string
  itemNumber: string
  description: string
  manufacturer?: string
  category?: string
}

type ItemComboboxProps = {
  items: ItemComboboxOption[]
  value: string
  onValueChange: (value: string) => void
  onCreateNew?: (query: string) => void
  placeholder?: string
  className?: string
  triggerClassName?: string
  disabled?: boolean
}

export function ItemCombobox({
  items,
  value,
  onValueChange,
  onCreateNew,
  placeholder = 'Select item',
  className,
  triggerClassName,
  disabled,
}: ItemComboboxProps) {
  const [open, setOpen] = useState(false)
  const [query, setQuery] = useState('')
  const selectedItem = items.find((item) => item.itemId === value)

  const filteredItems = useMemo(() => {
    const normalized = query.trim().toLowerCase()
    if (!normalized) return items.slice(0, 50)
    return items
      .filter((item) =>
        [item.itemNumber, item.description, item.manufacturer ?? '', item.category ?? '']
          .join(' ')
          .toLowerCase()
          .includes(normalized),
      )
      .slice(0, 50)
  }, [items, query])

  function handleSelect(itemId: string) {
    onValueChange(itemId)
    setOpen(false)
    setQuery('')
  }

  function handleCreateNew() {
    onCreateNew?.(query.trim())
    setOpen(false)
  }

  return (
    <Popover open={open} onOpenChange={setOpen}>
      <PopoverTrigger asChild>
        <Button
          type="button"
          variant="outline"
          role="combobox"
          aria-expanded={open}
          disabled={disabled}
          className={cn('w-full justify-between text-left font-normal', triggerClassName)}
        >
          <span className="min-w-0 truncate">
            {selectedItem ? `${selectedItem.itemNumber} / ${selectedItem.description}` : placeholder}
          </span>
          <ChevronsUpDown className="ml-2 h-4 w-4 shrink-0 opacity-50" />
        </Button>
      </PopoverTrigger>
      <PopoverContent align="start" className={cn('w-[var(--radix-popover-trigger-width)] p-0', className)}>
        <div className="flex items-center gap-2 border-b px-3 py-2">
          <Search className="h-4 w-4 text-muted-foreground" />
          <Input
            autoFocus
            value={query}
            onChange={(event) => setQuery(event.target.value)}
            placeholder="Filter by item number or description..."
            className="h-8 border-0 px-0 shadow-none focus-visible:ring-0"
          />
        </div>
        <div className="max-h-64 overflow-auto py-1">
          {filteredItems.map((item) => (
            <button
              key={item.itemId}
              type="button"
              className="flex w-full items-start gap-2 px-3 py-2 text-left text-sm hover:bg-muted"
              onClick={() => handleSelect(item.itemId)}
            >
              <Check className={cn('mt-0.5 h-4 w-4 shrink-0', value === item.itemId ? 'opacity-100' : 'opacity-0')} />
              <span className="min-w-0">
                <span className="block truncate font-medium">{item.itemNumber}</span>
                <span className="block truncate text-xs text-muted-foreground">{item.description}</span>
              </span>
            </button>
          ))}
          {filteredItems.length === 0 && (
            <div className="px-3 py-6 text-center text-sm text-muted-foreground">No matching items.</div>
          )}
        </div>
        {onCreateNew && (
          <div className="border-t p-2">
            <Button type="button" variant="ghost" className="w-full justify-start gap-2" onClick={handleCreateNew}>
              <Plus className="h-4 w-4" />
              + 新規アイテム登録
            </Button>
          </div>
        )}
      </PopoverContent>
    </Popover>
  )
}
