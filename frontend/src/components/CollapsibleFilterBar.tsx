import { useState } from 'react'
import { Input } from './ui/input'
import { Button } from './ui/button'

type FilterBarProps = {
  searchPlaceholder?: string
  onSearchChange: (query: string) => void
  children?: React.ReactNode
}

export function CollapsibleFilterBar({
  searchPlaceholder = 'Search...',
  onSearchChange,
  children,
}: FilterBarProps) {
  const [expanded, setExpanded] = useState(false)
  const [search, setSearch] = useState('')

  function handleSearchChange(value: string) {
    setSearch(value)
    onSearchChange(value)
  }

  return (
    <div className="space-y-2">
      <div className="flex items-center gap-2">
        <Input
          type="search"
          placeholder={searchPlaceholder}
          value={search}
          onChange={(e) => handleSearchChange(e.target.value)}
          className="max-w-sm h-8 text-sm"
        />
        {children && (
          <Button
            variant="ghost"
            size="sm"
            onClick={() => setExpanded(!expanded)}
            className="text-xs text-muted-foreground"
          >
            {expanded ? 'Hide filters' : 'More filters'}
          </Button>
        )}
        {search && (
          <Button
            variant="ghost"
            size="sm"
            onClick={() => handleSearchChange('')}
            className="text-xs text-muted-foreground"
          >
            Clear
          </Button>
        )}
      </div>
      {expanded && children && <div className="flex flex-wrap gap-2 pt-1">{children}</div>}
    </div>
  )
}
