import { useState } from 'react'
import { Popover, PopoverContent, PopoverTrigger } from './ui/popover'

type ItemInfo = {
  itemNumber: string
  description?: string
  manufacturer?: string
  category?: string
}

export function ItemInfoPopover({ itemNumber, description, manufacturer, category }: ItemInfo) {
  const [open, setOpen] = useState(false)

  return (
    <Popover open={open} onOpenChange={setOpen}>
      <PopoverTrigger asChild>
        <button
          type="button"
          className="text-left font-mono text-sm underline decoration-dotted underline-offset-2 hover:text-primary cursor-help"
          onMouseEnter={() => setOpen(true)}
          onMouseLeave={() => setOpen(false)}
        >
          {itemNumber}
        </button>
      </PopoverTrigger>
      <PopoverContent className="w-72 p-3" side="top" align="start">
        <div className="space-y-1.5 text-sm">
          <p className="font-semibold">{itemNumber}</p>
          {manufacturer && (
            <p className="text-muted-foreground">
              <span className="font-medium text-foreground">Manufacturer:</span> {manufacturer}
            </p>
          )}
          {description && (
            <p className="text-muted-foreground">
              <span className="font-medium text-foreground">Description:</span> {description}
            </p>
          )}
          {category && (
            <p className="text-muted-foreground">
              <span className="font-medium text-foreground">Category:</span> {category}
            </p>
          )}
        </div>
      </PopoverContent>
    </Popover>
  )
}
