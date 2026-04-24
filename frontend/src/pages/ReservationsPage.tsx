import { useSearchParams } from 'react-router-dom'
import { useState } from 'react'

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
import { exportReservationsCSV } from '@/lib/additionalApi'
import { Download } from 'lucide-react'

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

export function ReservationsPage() {
  const [searchParams, setSearchParams] = useSearchParams()
  const device = searchParams.get('device') ?? ''
  const scope = searchParams.get('scope') ?? ''
  const [search, setSearch] = useState('')
  const { data } = useReservations(device, scope)

  const filteredRows = (data?.rows ?? []).filter((row) => {
    if (!search) return true
    const q = search.toLowerCase()
    return (
      row.itemNumber.toLowerCase().includes(q) ||
      row.description.toLowerCase().includes(q) ||
      row.id.toLowerCase().includes(q)
    )
  })

  const updateContext = (key: 'device' | 'scope', value: string) => {
    const next = new URLSearchParams(searchParams)
    if (value.trim() === '') {
      next.delete(key)
    } else {
      next.set(key, value)
    }
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
            Reservation visibility {device || scope ? `filtered by Device: ${device || 'all'}, Scope: ${scope || 'all'}` : 'for all contexts'}.
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
              onDeviceChange={(value) => updateContext('device', value)}
              onScopeChange={(value) => updateContext('scope', value)}
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
                  <TableHead>Scope</TableHead>
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
                  filteredRows.map((row) => (
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
                      <TableCell className="text-sm">{row.device}</TableCell>
                      <TableCell className="text-sm">{row.scope}</TableCell>
                      <TableCell>
                        <Badge variant={getStatusBadgeVariant(row.status)} className="text-xs">
                          {row.status.replace(/_/g, ' ')}
                        </Badge>
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
