import { useSearchParams } from 'react-router-dom'
import { useState, useMemo } from 'react'

import { Card, CardContent } from '@/components/ui/card'
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from '@/components/ui/table'
import { Badge } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'
import { DeviceScopeFilters } from '@/components/context/DeviceScopeFilters'
import { ItemInfoPopover } from '@/components/ItemInfoPopover'
import { CollapsibleFilterBar } from '@/components/CollapsibleFilterBar'
import { useReservations } from '@/hooks/useReservations'
import { useDeviceScopes } from '@/hooks/useDeviceScopes'
import { exportReservationsCSV } from '@/lib/additionalApi'
import { Download, ChevronRight, Layers, Package, Box, MapPin, ClipboardList, Info } from 'lucide-react'

function getStatusBadgeVariant(status: string) {
  switch (status.toLowerCase()) {
    case 'reserved':
    case 'allocated':
      return 'default' as const
    case 'partially_allocated':
      return 'secondary' as const
    case 'awaiting_stock':
      return 'destructive' as const
    default:
      return 'outline' as const
  }
}

function getScopeIcon(type: string) {
  switch (type) {
    case 'system': return <Layers className="w-3 h-3 text-blue-500" />
    case 'assembly': return <Package className="w-3 h-3 text-orange-500" />
    case 'module': return <Box className="w-3 h-3 text-green-500" />
    case 'area': return <MapPin className="w-3 h-3 text-purple-500" />
    case 'work_package': return <ClipboardList className="w-3 h-3 text-gray-500" />
    default: return <Info className="w-3 h-3" />
  }
}

export function ReservationsPage() {
  const [searchParams, setSearchParams] = useSearchParams()
  const device = searchParams.get('device') ?? ''
  const scope = searchParams.get('scope') ?? ''
  const system = searchParams.get('system') ?? ''
  const [search, setSearch] = useState('')
  const { data } = useReservations(device, scope)
  const { data: scopeData } = useDeviceScopes()

  const scopes = useMemo(() => scopeData?.rows ?? [], [scopeData?.rows])

  // Helper to get scope breadcrumbs
  const getScopePath = useMemo(() => {
    const cache: Record<string, string[]> = {}
    const resolve = (id: string): string[] => {
      if (cache[id]) return cache[id]
      const s = scopes.find(x => x.id === id || (x.deviceKey === device && x.scopeKey === id))
      if (!s) return [id]
      if (!s.parentScopeId) return [s.scopeName || s.scopeKey]
      const p = resolve(s.parentScopeId)
      cache[id] = [...p, s.scopeName || s.scopeKey]
      return cache[id]
    }
    return resolve
  }, [scopes, device])

  const filteredRows = (data?.rows ?? []).filter((row) => {
    // Apply system filter if present
    if (system) {
      const s = scopes.find(x => x.deviceKey === row.device && x.scopeKey === row.scope)
      if (s?.systemKey !== system) return false
    }

    if (!search) return true
    const q = search.toLowerCase()
    return (
      row.itemNumber.toLowerCase().includes(q) ||
      row.description.toLowerCase().includes(q) ||
      row.id.toLowerCase().includes(q)
    )
  })

  const updateContext = (key: 'device' | 'scope' | 'system', value: string) => {
    const next = new URLSearchParams(searchParams)
    if (value.trim() === '') {
      next.delete(key)
    } else {
      next.set(key, value)
    }
    if (key !== 'scope') next.delete('scope')
    setSearchParams(next, { replace: true })
  }

  async function handleExport() {
    try {
      await exportReservationsCSV(device || undefined, scope || undefined)
    } catch {
      // silent fail for now
    }
  }

  return (
    <div className="space-y-6 p-6">
      <div className="flex items-center justify-between">
        <div className="space-y-2">
          <h1 className="text-3xl font-bold tracking-tight">Reservations</h1>
          <p className="text-muted-foreground">
            Reservation visibility with hierarchical context.
          </p>
        </div>
        <Button onClick={() => void handleExport()} size="sm" variant="outline" className="gap-2">
          <Download className="w-4 h-4" />
          Export CSV
        </Button>
      </div>

      <Card>
        <CardContent className="pt-6">
          <div className="flex flex-wrap items-end gap-4">
            <DeviceScopeFilters
              device={device}
              scope={scope}
              system={system}
              onDeviceChange={(value) => updateContext('device', value)}
              onScopeChange={(value) => updateContext('scope', value)}
              onSystemChange={(value) => updateContext('system', value)}
            />
          </div>
        </CardContent>
      </Card>

      <Card>
        <CardContent className="pt-6">
          <CollapsibleFilterBar
            searchPlaceholder="Filter reservations..."
            onSearchChange={setSearch}
          />
          <div className="overflow-x-auto mt-4">
            <Table>
              <TableHeader>
                <TableRow>
                  <TableHead className="w-16">ID</TableHead>
                  <TableHead>Item</TableHead>
                  <TableHead>Description</TableHead>
                  <TableHead className="w-16 text-right">Qty</TableHead>
                  <TableHead>Device</TableHead>
                  <TableHead>Scope Path</TableHead>
                  <TableHead className="w-28">Status</TableHead>
                </TableRow>
              </TableHeader>
              <TableBody>
                {filteredRows.length === 0 ? (
                  <TableRow>
                    <TableCell colSpan={7} className="text-center text-muted-foreground py-8">
                      No reservations found.
                    </TableCell>
                  </TableRow>
                ) : (
                  filteredRows.map((row) => {
                    const s = scopes.find(x => x.deviceKey === row.device && x.scopeKey === row.scope)
                    const path = getScopePath(s?.id || row.scope)
                    return (
                      <TableRow key={row.id}>
                        <TableCell className="font-medium text-xs text-muted-foreground">{row.id.slice(0, 8)}</TableCell>
                        <TableCell>
                          <ItemInfoPopover
                            itemNumber={row.itemNumber}
                            description={row.description}
                          />
                        </TableCell>
                        <TableCell className="text-sm text-muted-foreground max-w-[10rem] truncate">
                          {row.description}
                        </TableCell>
                        <TableCell className="text-sm text-right tabular-nums">{row.quantity}</TableCell>
                        <TableCell className="text-sm font-mono">{row.device}</TableCell>
                        <TableCell>
                          <div className="flex items-center gap-1 flex-wrap">
                            {s && getScopeIcon(s.scopeType)}
                            {path.map((segment, idx) => (
                              <span key={idx} className="flex items-center gap-1">
                                {idx > 0 && <ChevronRight className="w-2.5 h-2.5 text-muted-foreground" />}
                                <span className={idx === path.length - 1 ? 'text-sm font-medium' : 'text-[10px] text-muted-foreground uppercase'}>
                                  {segment}
                                </span>
                              </span>
                            ))}
                          </div>
                        </TableCell>
                        <TableCell>
                          <Badge variant={getStatusBadgeVariant(row.status)} className="text-xs">
                            {row.status.replace(/_/g, ' ')}
                          </Badge>
                        </TableCell>
                      </TableRow>
                    )
                  })
                )}
              </TableBody>
            </Table>
          </div>
        </CardContent>
      </Card>
    </div>
  )
}
