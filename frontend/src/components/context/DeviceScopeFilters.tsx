import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from '@/components/ui/select'
import { useDeviceScopes } from '@/hooks/useDeviceScopes'
import { useScopeSystems } from '@/hooks/useScopeSystems'
import type { DeviceScopeRecord } from '@/types'

const ALL_VALUE = '__all__'

type ScopeTreeRow = {
  row: DeviceScopeRecord
  level: number
}

type DeviceScopeFiltersProps = {
  device: string
  scope: string
  system?: string
  onDeviceChange: (value: string) => void
  onScopeChange: (value: string) => void
  onSystemChange?: (value: string) => void
  compact?: boolean
}

export function DeviceScopeFilters({
  device,
  scope,
  system,
  onDeviceChange,
  onScopeChange,
  onSystemChange,
  compact = false,
}: DeviceScopeFiltersProps) {
  const { data } = useDeviceScopes()
  const { data: systemData } = useScopeSystems()
  const activeRows = data?.rows.filter((row) => row.status !== 'inactive') ?? []
  
  const deviceOptions = Array.from(new Set(activeRows.map((row) => row.deviceKey))).sort((left, right) =>
    left.localeCompare(right),
  )

  const systemOptions = systemData?.rows.filter((row) => row.status !== 'inactive') ?? []

  // Recursive function to build a tree structure for indented display
  const buildTree = (parentId: string, level: number): ScopeTreeRow[] => {
    const children = activeRows
      .filter((row) => (!device || row.deviceKey === device) && row.parentScopeId === parentId)
      .filter((row) => !system || row.systemKey === system)
      .sort((left, right) => left.scopeName.localeCompare(right.scopeName))

    let results: ScopeTreeRow[] = []
    for (const child of children) {
      results.push({ row: child, level })
      results = [...results, ...buildTree(child.id, level + 1)]
    }
    return results
  }

  const scopeTree = buildTree('', 0)

  return (
    <div className={`flex items-center gap-3 ${compact ? 'flex-row' : 'flex-col sm:flex-row'}`}>
      <div className={`flex items-center gap-1.5 ${compact ? '' : 'w-full sm:w-40'}`}>
        <label htmlFor="ctx-device" className="text-xs font-medium text-muted-foreground uppercase tracking-wide">
          Device
        </label>
        <Select value={device || ALL_VALUE} onValueChange={(value) => onDeviceChange(value === ALL_VALUE ? '' : value)}>
          <SelectTrigger id="ctx-device" className={compact ? 'h-8 w-28 text-sm' : 'w-full'}>
            <SelectValue placeholder="All devices" />
          </SelectTrigger>
          <SelectContent>
            <SelectItem value={ALL_VALUE}>All devices</SelectItem>
            {deviceOptions.map((deviceKey) => (
              <SelectItem key={deviceKey} value={deviceKey}>
                {deviceKey}
              </SelectItem>
            ))}
          </SelectContent>
        </Select>
      </div>

      {onSystemChange && (
        <div className={`flex items-center gap-1.5 ${compact ? '' : 'w-full sm:w-48'}`}>
          <label htmlFor="ctx-system" className="text-xs font-medium text-muted-foreground uppercase tracking-wide">
            System
          </label>
          <Select value={system || ALL_VALUE} onValueChange={(value) => onSystemChange(value === ALL_VALUE ? '' : value)}>
            <SelectTrigger id="ctx-system" className={compact ? 'h-8 w-36 text-sm' : 'w-full'}>
              <SelectValue placeholder="All systems" />
            </SelectTrigger>
            <SelectContent>
              <SelectItem value={ALL_VALUE}>All systems</SelectItem>
              {systemOptions.map((s) => (
                <SelectItem key={s.key} value={s.key}>
                  {s.name}
                </SelectItem>
              ))}
            </SelectContent>
          </Select>
        </div>
      )}

      <div className={`flex items-center gap-1.5 ${compact ? '' : 'w-full sm:w-64'}`}>
        <label htmlFor="ctx-scope" className="text-xs font-medium text-muted-foreground uppercase tracking-wide">
          Scope
        </label>
        <Select value={scope || ALL_VALUE} onValueChange={(value) => onScopeChange(value === ALL_VALUE ? '' : value)}>
          <SelectTrigger id="ctx-scope" className={compact ? 'h-8 w-44 text-sm' : 'w-full'}>
            <SelectValue placeholder="All scopes" />
          </SelectTrigger>
          <SelectContent>
            <SelectItem value={ALL_VALUE}>All scopes</SelectItem>
            {scopeTree.map(({ row, level }) => (
              <SelectItem key={row.id} value={row.scopeKey}>
                <span className="flex items-center">
                  {Array.from({ length: level }).map((_, i) => (
                    <span key={i} className="w-3 border-l h-4 ml-1 mr-1 border-muted-foreground/30" />
                  ))}
                  <span className={level === 0 ? 'font-medium' : 'text-muted-foreground text-sm'}>
                    {row.scopeName || row.scopeKey}
                  </span>
                </span>
              </SelectItem>
            ))}
          </SelectContent>
        </Select>
      </div>
    </div>
  )
}
