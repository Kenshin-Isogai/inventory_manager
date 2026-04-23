import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from '@/components/ui/select'
import { useDeviceScopes } from '@/hooks/useDeviceScopes'

const ALL_VALUE = '__all__'

type DeviceScopeFiltersProps = {
  device: string
  scope: string
  onDeviceChange: (value: string) => void
  onScopeChange: (value: string) => void
  compact?: boolean
}

export function DeviceScopeFilters({
  device,
  scope,
  onDeviceChange,
  onScopeChange,
  compact = false,
}: DeviceScopeFiltersProps) {
  const { data } = useDeviceScopes()
  const activeRows = data?.rows.filter((row) => row.status !== 'inactive') ?? []
  const deviceOptions = Array.from(new Set(activeRows.map((row) => row.deviceKey))).sort((left, right) =>
    left.localeCompare(right),
  )
  const scopeOptions = Array.from(
    new Map(
      activeRows
        .filter((row) => !device || row.deviceKey === device)
        .map((row) => [row.scopeKey, row]),
    ).values(),
  ).sort((left, right) => left.scopeKey.localeCompare(right.scopeKey))

  return (
    <div className={`flex items-center gap-3 ${compact ? 'flex-row' : 'flex-col sm:flex-row'}`}>
      <div className={`flex items-center gap-1.5 ${compact ? '' : 'w-full sm:w-48'}`}>
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
      <div className={`flex items-center gap-1.5 ${compact ? '' : 'w-full sm:w-64'}`}>
        <label htmlFor="ctx-scope" className="text-xs font-medium text-muted-foreground uppercase tracking-wide">
          Scope
        </label>
        <Select value={scope || ALL_VALUE} onValueChange={(value) => onScopeChange(value === ALL_VALUE ? '' : value)}>
          <SelectTrigger id="ctx-scope" className={compact ? 'h-8 w-40 text-sm' : 'w-full'}>
            <SelectValue placeholder="All scopes" />
          </SelectTrigger>
          <SelectContent>
            <SelectItem value={ALL_VALUE}>All scopes</SelectItem>
            {scopeOptions.map((row) => (
              <SelectItem key={`${row.deviceKey}-${row.scopeKey}`} value={row.scopeKey}>
                {row.scopeName || row.scopeKey}
              </SelectItem>
            ))}
          </SelectContent>
        </Select>
      </div>
    </div>
  )
}
