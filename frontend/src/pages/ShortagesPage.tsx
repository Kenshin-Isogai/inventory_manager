import { useSearchParams } from 'react-router-dom'
import { useState, useMemo } from 'react'

import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card'
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
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from '@/components/ui/select'
import { useImports } from '@/hooks/useImports'
import { useEnhancedShortages } from '@/hooks/useEnhancedShortages'
import { useDeviceScopes } from '@/hooks/useDeviceScopes'
import { ShortageTimelineView } from '@/components/ShortageTimelineView'
import { DeviceScopeFilters } from '@/components/context/DeviceScopeFilters'
import { ItemInfoPopover } from '@/components/ItemInfoPopover'
import { CollapsibleFilterBar } from '@/components/CollapsibleFilterBar'
import { exportShortagesCSV } from '@/lib/mockApi'
import { Download, ChevronRight, Layers, Package, Box, MapPin, ClipboardList, Info } from 'lucide-react'

function getStatusBadgeVariant(status: string) {
  switch (status.toLowerCase()) {
    case 'completed':
      return 'default' as const
    case 'pending':
      return 'secondary' as const
    case 'failed':
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

const COVERAGE_RULES = [
  { value: 'none', label: 'No coverage' },
  { value: 'submitted', label: 'Submitted+' },
  { value: 'approved', label: 'Approved+ (default)' },
  { value: 'ordered', label: 'Ordered+' },
  { value: 'received', label: 'Received only' },
]

export function ShortagesPage() {
  const [searchParams, setSearchParams] = useSearchParams()
  const device = searchParams.get('device') ?? ''
  const scope = searchParams.get('scope') ?? ''
  const system = searchParams.get('system') ?? ''
  const [coverageRule, setCoverageRule] = useState('approved')
  const [search, setSearch] = useState('')
  const { data: shortages } = useEnhancedShortages(device || undefined, scope || undefined, coverageRule)
  const { data: scopeData } = useDeviceScopes()
  const { data: imports } = useImports()
  const [exportMessage, setExportMessage] = useState('')
  const [isExporting, setIsExporting] = useState(false)

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

  const filteredRows = (shortages?.rows ?? []).filter((row) => {
    // Apply system filter if present (shortage rows might not have systemKey directly, so we look it up)
    if (system) {
      const s = scopes.find(x => x.deviceKey === row.device && x.scopeKey === row.scope)
      if (s?.systemKey !== system) return false
    }

    if (!search) return true
    const q = search.toLowerCase()
    return (
      row.itemNumber.toLowerCase().includes(q) ||
      row.manufacturer.toLowerCase().includes(q) ||
      row.description.toLowerCase().includes(q)
    )
  })

  function updateContext(key: 'device' | 'scope' | 'system', value: string) {
    const next = new URLSearchParams(searchParams)
    if (value.trim() === '') next.delete(key)
    else next.set(key, value)
    
    // Reset scope if device or system changes
    if (key !== 'scope') next.delete('scope')
    
    setSearchParams(next, { replace: true })
  }

  async function handleExport() {
    setIsExporting(true)
    try {
      const csv = await exportShortagesCSV(device, scope)
      const blob = new Blob([csv], { type: 'text/csv;charset=utf-8' })
      const url = URL.createObjectURL(blob)
      const link = document.createElement('a')
      link.href = url
      link.download = `shortages${device ? `-${device}` : ''}${scope ? `-${scope}` : ''}.csv`
      link.click()
      URL.revokeObjectURL(url)
      setExportMessage('CSV exported successfully.')
      setTimeout(() => setExportMessage(''), 3000)
    } finally {
      setIsExporting(false)
    }
  }

  return (
    <div className="space-y-6 p-6">
      <div className="space-y-2">
        <h1 className="text-3xl font-bold tracking-tight">Shortages</h1>
        <p className="text-muted-foreground">
          Shortage analysis with procurement pipeline visibility.
        </p>
      </div>

      <Card>
        <CardContent className="pt-6">
          <div className="flex flex-wrap items-end gap-4">
            <DeviceScopeFilters
              device={device}
              scope={scope}
              system={system}
              onDeviceChange={(v) => updateContext('device', v)}
              onScopeChange={(v) => updateContext('scope', v)}
              onSystemChange={(v) => updateContext('system', v)}
            />
            <div className="w-48">
              <label className="text-xs text-muted-foreground mb-1 block">Coverage Rule</label>
              <Select value={coverageRule} onValueChange={setCoverageRule}>
                <SelectTrigger className="h-8 text-sm">
                  <SelectValue />
                </SelectTrigger>
                <SelectContent>
                  {COVERAGE_RULES.map((r) => (
                    <SelectItem key={r.value} value={r.value}>{r.label}</SelectItem>
                  ))}
                </SelectContent>
              </Select>
            </div>
          </div>
        </CardContent>
      </Card>

      {device && scope && (
        <ShortageTimelineView device={device} scope={scope} />
      )}

      <Card>
        <CardHeader className="flex flex-row items-center justify-between space-y-0">
          <div className="space-y-1">
            <CardTitle>Shortage List ({filteredRows.length})</CardTitle>
            <CardDescription>
              Coverage rule: {COVERAGE_RULES.find((r) => r.value === coverageRule)?.label}
            </CardDescription>
          </div>
          <Button
            onClick={() => void handleExport()}
            disabled={isExporting}
            size="sm"
            variant="outline"
            className="gap-2"
          >
            <Download className="w-4 h-4" />
            {isExporting ? 'Exporting...' : 'Export CSV'}
          </Button>
        </CardHeader>
        <CardContent>
          {exportMessage && (
            <div className="mb-4 p-3 bg-green-50 border border-green-200 text-green-800 text-sm rounded">
              {exportMessage}
            </div>
          )}
          <CollapsibleFilterBar
            searchPlaceholder="Filter by item, manufacturer..."
            onSearchChange={setSearch}
          />
          <div className="overflow-x-auto mt-4">
            <Table>
              <TableHeader>
                <TableRow>
                  <TableHead>Device</TableHead>
                  <TableHead>Scope Path</TableHead>
                  <TableHead>Item</TableHead>
                  <TableHead>Description</TableHead>
                  <TableHead className="text-right">Required</TableHead>
                  <TableHead className="text-right">Reserved</TableHead>
                  <TableHead className="text-right">Available</TableHead>
                  <TableHead className="text-right">Raw Short</TableHead>
                  <TableHead className="text-right">In Flow</TableHead>
                  <TableHead className="text-right">Ordered</TableHead>
                  <TableHead className="text-right">Received</TableHead>
                  <TableHead className="text-right">Actionable</TableHead>
                </TableRow>
              </TableHeader>
              <TableBody>
                {filteredRows.length === 0 ? (
                  <TableRow>
                    <TableCell colSpan={12} className="text-center text-muted-foreground py-8">
                      No shortages found.
                    </TableCell>
                  </TableRow>
                ) : (
                  filteredRows.map((row) => {
                    const s = scopes.find(x => x.deviceKey === row.device && x.scopeKey === row.scope)
                    const path = getScopePath(s?.id || row.scope)
                    return (
                      <TableRow key={`${row.device}-${row.scope}-${row.itemNumber}`}>
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
                          <ItemInfoPopover
                            itemNumber={row.itemNumber}
                            description={row.description}
                            manufacturer={row.manufacturer}
                          />
                        </TableCell>
                        <TableCell className="text-sm text-muted-foreground max-w-[10rem] truncate">
                          {row.description}
                        </TableCell>
                        <TableCell className="text-right tabular-nums">{row.requiredQuantity}</TableCell>
                        <TableCell className="text-right tabular-nums">{row.reservedQuantity}</TableCell>
                        <TableCell className="text-right tabular-nums">{row.availableQuantity}</TableCell>
                        <TableCell className="text-right tabular-nums">
                          {row.rawShortage > 0 ? (
                            <Badge variant="destructive" className="text-xs">{row.rawShortage}</Badge>
                          ) : (
                            <span className="text-muted-foreground">0</span>
                          )}
                        </TableCell>
                        <TableCell className="text-right tabular-nums text-blue-600">
                          {row.inRequestFlowQuantity || '—'}
                        </TableCell>
                        <TableCell className="text-right tabular-nums text-blue-600">
                          {row.orderedQuantity || '—'}
                        </TableCell>
                        <TableCell className="text-right tabular-nums text-green-600">
                          {row.receivedQuantity || '—'}
                        </TableCell>
                        <TableCell className="text-right tabular-nums font-semibold">
                          {row.actionableShortage > 0 ? (
                            <Badge variant="destructive">{row.actionableShortage}</Badge>
                          ) : (
                            <span className="text-green-600">0</span>
                          )}
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

      <Card>
        <CardHeader>
          <CardTitle>Import History</CardTitle>
          <CardDescription>Recent item and alias imports</CardDescription>
        </CardHeader>
        <CardContent>
          <div className="overflow-x-auto">
            <Table>
              <TableHeader>
                <TableRow>
                  <TableHead className="w-16">ID</TableHead>
                  <TableHead>Type</TableHead>
                  <TableHead className="w-28">Status</TableHead>
                  <TableHead>File</TableHead>
                  <TableHead className="max-w-xs">Summary</TableHead>
                </TableRow>
              </TableHeader>
              <TableBody>
                {imports?.rows.map((row) => (
                  <TableRow key={row.id}>
                    <TableCell className="font-medium text-sm">{row.id}</TableCell>
                    <TableCell className="text-sm">{row.importType}</TableCell>
                    <TableCell>
                      <Badge variant={getStatusBadgeVariant(row.status)} className="text-xs">
                        {row.status}
                      </Badge>
                    </TableCell>
                    <TableCell className="text-sm">{row.fileName}</TableCell>
                    <TableCell className="text-sm text-muted-foreground">{row.summary}</TableCell>
                  </TableRow>
                ))}
              </TableBody>
            </Table>
          </div>
          {imports?.rows.length === 0 && (
            <div className="text-center py-8 text-muted-foreground">
              No import history.
            </div>
          )}
        </CardContent>
      </Card>
    </div>
  )
}
