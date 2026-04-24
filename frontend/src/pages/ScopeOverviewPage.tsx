import { useState, useMemo } from 'react'
import { useSearchParams } from 'react-router-dom'
import { Card, CardContent, CardHeader, CardTitle } from '../components/ui/card'
import { Badge } from '../components/ui/badge'
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from '../components/ui/table'
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from '../components/ui/select'
import { CollapsibleFilterBar } from '../components/CollapsibleFilterBar'
import { useScopeOverview } from '../hooks/useScopeOverview'
import { useDevices } from '../hooks/useDevices'
import type { ScopeOverviewRow } from '../types'

function buildTree(rows: ScopeOverviewRow[]): (ScopeOverviewRow & { children: ScopeOverviewRow[]; depth: number })[] {
  const childMap = new Map<string, ScopeOverviewRow[]>()
  const roots: ScopeOverviewRow[] = []
  for (const row of rows) {
    if (row.parentScopeId) {
      const existing = childMap.get(row.parentScopeId) ?? []
      existing.push(row)
      childMap.set(row.parentScopeId, existing)
    } else {
      roots.push(row)
    }
  }

  const result: (ScopeOverviewRow & { children: ScopeOverviewRow[]; depth: number })[] = []
  function walk(items: ScopeOverviewRow[], depth: number) {
    for (const item of items) {
      const children = childMap.get(item.scopeId) ?? []
      result.push({ ...item, children, depth })
      if (children.length > 0) walk(children, depth + 1)
    }
  }
  walk(roots, 0)
  return result
}

export function ScopeOverviewPage() {
  const [searchParams, setSearchParams] = useSearchParams()
  const device = searchParams.get('device') ?? ''
  const [search, setSearch] = useState('')
  const { data: devices } = useDevices()
  const { data } = useScopeOverview(device || undefined)

  const tree = useMemo(() => {
    if (!data?.rows) return []
    const filtered = search
      ? data.rows.filter(
          (r) =>
            r.scopeKey.toLowerCase().includes(search.toLowerCase()) ||
            r.scopeName.toLowerCase().includes(search.toLowerCase()),
        )
      : data.rows
    return buildTree(filtered)
  }, [data, search])

  function handleDeviceChange(value: string) {
    const next = new URLSearchParams(searchParams)
    if (value === '__all__') next.delete('device')
    else next.set('device', value)
    setSearchParams(next, { replace: true })
  }

  return (
    <div className="space-y-6 p-6">
      <div className="space-y-2">
        <h1 className="text-3xl font-bold">Scope Overview</h1>
        <p className="text-muted-foreground">
          Scope tree with requirements, reservations, and shortage summary counts.
        </p>
      </div>

      <Card>
        <CardContent className="pt-6">
          <div className="flex items-center gap-4">
            <div className="w-48">
              <Select value={device || '__all__'} onValueChange={handleDeviceChange}>
                <SelectTrigger>
                  <SelectValue placeholder="All Devices" />
                </SelectTrigger>
                <SelectContent>
                  <SelectItem value="__all__">All Devices</SelectItem>
                  {devices?.rows.map((d) => (
                    <SelectItem key={d.deviceKey} value={d.deviceKey}>
                      {d.name}
                    </SelectItem>
                  ))}
                </SelectContent>
              </Select>
            </div>
            <CollapsibleFilterBar
              searchPlaceholder="Filter scopes..."
              onSearchChange={setSearch}
            />
          </div>
        </CardContent>
      </Card>

      <Card>
        <CardHeader>
          <CardTitle>Scopes ({tree.length})</CardTitle>
        </CardHeader>
        <CardContent>
          <div className="overflow-x-auto">
            <Table>
              <TableHeader>
                <TableRow>
                  <TableHead>Scope</TableHead>
                  <TableHead>Device</TableHead>
                  <TableHead>Type</TableHead>
                  <TableHead>Owner</TableHead>
                  <TableHead>Status</TableHead>
                  <TableHead className="text-right">Requirements</TableHead>
                  <TableHead className="text-right">Reservations</TableHead>
                  <TableHead className="text-right">Shortage Items</TableHead>
                </TableRow>
              </TableHeader>
              <TableBody>
                {tree.length === 0 ? (
                  <TableRow>
                    <TableCell colSpan={8} className="text-center text-muted-foreground py-8">
                      No scopes found.
                    </TableCell>
                  </TableRow>
                ) : (
                  tree.map((row) => (
                    <TableRow key={row.scopeId} className="cursor-pointer hover:bg-muted/50">
                      <TableCell>
                        <span style={{ paddingLeft: `${row.depth * 1.25}rem` }} className="flex items-center gap-1">
                          {row.depth > 0 && <span className="text-muted-foreground">{'└'}</span>}
                          <span className="font-medium">{row.scopeName || row.scopeKey}</span>
                          <span className="text-xs text-muted-foreground ml-1">({row.scopeKey})</span>
                        </span>
                      </TableCell>
                      <TableCell>{row.deviceName}</TableCell>
                      <TableCell>
                        <Badge variant="outline" className="text-xs">{row.scopeType || '—'}</Badge>
                      </TableCell>
                      <TableCell className="text-sm text-muted-foreground">{row.ownerDepartment || '—'}</TableCell>
                      <TableCell>
                        <Badge variant={row.status === 'active' ? 'default' : 'secondary'}>{row.status}</Badge>
                      </TableCell>
                      <TableCell className="text-right tabular-nums">{row.requirementsCount}</TableCell>
                      <TableCell className="text-right tabular-nums">{row.reservationsCount}</TableCell>
                      <TableCell className="text-right tabular-nums">
                        {row.shortageItemCount > 0 ? (
                          <Badge variant="destructive">{row.shortageItemCount}</Badge>
                        ) : (
                          <span className="text-muted-foreground">0</span>
                        )}
                      </TableCell>
                    </TableRow>
                  ))
                )}
              </TableBody>
            </Table>
          </div>
        </CardContent>
      </Card>
    </div>
  )
}
