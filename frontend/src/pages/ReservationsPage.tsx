import type { ChangeEvent, FormEvent } from 'react'
import { useMemo, useRef, useState } from 'react'
import { useSearchParams } from 'react-router-dom'
import { useSWRConfig } from 'swr'
import { CalendarCheck, Download, Eye, FileDown, FileUp, Loader2, PackagePlus } from 'lucide-react'

import { DeviceScopeFilters } from '@/components/context/DeviceScopeFilters'
import { ItemInfoPopover } from '@/components/ItemInfoPopover'
import { Badge } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card'
import { Dialog, DialogContent, DialogDescription, DialogFooter, DialogHeader, DialogTitle } from '@/components/ui/dialog'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from '@/components/ui/select'
import { Sheet, SheetContent, SheetDescription, SheetHeader, SheetTitle } from '@/components/ui/sheet'
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from '@/components/ui/table'
import { Textarea } from '@/components/ui/textarea'
import { useAuthSession } from '@/hooks/useAuthSession'
import { useDeviceScopes } from '@/hooks/useDeviceScopes'
import { useInventoryOverview } from '@/hooks/useInventoryOverview'
import { multiWordMatch } from '@/lib/search'
import { useReservations } from '@/hooks/useReservations'
import { applyAllocationsImport, applyReservationsImport, exportReservationsCSV, previewAllocationsImport, previewReservationsImport } from '@/lib/additionalApi'
import { resolveActorId } from '@/lib/auth'
import { downloadTextFile } from '@/lib/csv'
import { createReservation, fetchReservationDetail, reservationAction } from '@/lib/mockApi'
import type { ImportPreviewResult, ReservationDetail } from '@/types'

const RESERVATION_TEMPLATE =
  'device_scope_id,item_id,quantity,requested_by,purpose,priority,needed_by_at,planned_use_at,hold_until_at,note\n' +
  'ds-er2-powerboard,item-er2,5,session-user,Build allocation,normal,2026-05-10,2026-05-10,2026-05-31,Initial reservation\n'
const ALLOCATION_TEMPLATE =
  'reservation_id,location_code,quantity,actor_id,note\nRES-001,TOKYO-A1,3,session-user,Allocate available stock\n'

function getStatusBadgeVariant(status: string) {
  switch (status.toLowerCase()) {
    case 'reserved':
    case 'allocated':
      return 'default' as const
    case 'requested':
    case 'partially_allocated':
      return 'secondary' as const
    case 'awaiting_stock':
    case 'cancelled':
      return 'destructive' as const
    default:
      return 'outline' as const
  }
}

export function ReservationsPage() {
  const [searchParams, setSearchParams] = useSearchParams()
  const device = searchParams.get('device') ?? ''
  const scope = searchParams.get('scope') ?? ''
  const system = searchParams.get('system') ?? ''
  const { mutate } = useSWRConfig()
  const { data: session } = useAuthSession()
  const { data } = useReservations(device, scope)
  const { data: scopeData } = useDeviceScopes()
  const { data: inventory } = useInventoryOverview()
  const csvInputRef = useRef<HTMLInputElement>(null)
  const allocationCsvInputRef = useRef<HTMLInputElement>(null)

  const [search, setSearch] = useState('')
  const [formScopeId, setFormScopeId] = useState('')
  const [formItemId, setFormItemId] = useState('')
  const [quantity, setQuantity] = useState(1)
  const [purpose, setPurpose] = useState('Build allocation')
  const [priority, setPriority] = useState('normal')
  const [neededByAt, setNeededByAt] = useState('')
  const [plannedUseAt, setPlannedUseAt] = useState('')
  const [holdUntilAt, setHoldUntilAt] = useState('')
  const [note, setNote] = useState('')
  const [feedback, setFeedback] = useState<{ tone: 'success' | 'error'; text: string } | null>(null)
  const [isCreating, setIsCreating] = useState(false)
  const [isBulkCreating, setIsBulkCreating] = useState(false)
  const [isBulkAllocating, setIsBulkAllocating] = useState(false)
  const [csvDialogOpen, setCsvDialogOpen] = useState(false)
  const [csvPreview, setCsvPreview] = useState<ImportPreviewResult | null>(null)
  const [csvFile, setCsvFile] = useState<File | null>(null)
  const [csvKind, setCsvKind] = useState<'reservations' | 'allocations'>('reservations')
  const [detailOpen, setDetailOpen] = useState(false)
  const [detail, setDetail] = useState<ReservationDetail | null>(null)
  const [loadingDetailId, setLoadingDetailId] = useState('')
  const [actionDialog, setActionDialog] = useState<null | { action: 'allocate' | 'release' | 'fulfill' | 'cancel' | 'undo' }>(null)
  const [actionLocation, setActionLocation] = useState('')
  const [actionQuantity, setActionQuantity] = useState(1)
  const [actionReason, setActionReason] = useState('')
  const [actionBusy, setActionBusy] = useState(false)

  const scopes = scopeData?.rows ?? []
  const activeScopes = scopes.filter((row) => row.status !== 'inactive')
  const items = useMemo(() => {
    const byId = new Map((inventory?.balances ?? []).map((row) => [row.itemId, row]))
    return Array.from(byId.values()).sort((left, right) => left.itemNumber.localeCompare(right.itemNumber))
  }, [inventory?.balances])
  const actorId = resolveActorId(session)
  const csvInvalidRows = (csvPreview?.rows ?? []).filter((row) => row.status === 'invalid')

  const filteredRows = (data?.rows ?? []).filter((row) => {
    if (system) {
      const matchedScope = scopes.find((candidate) => candidate.deviceKey === row.device && candidate.scopeKey === row.scope)
      if (matchedScope?.systemKey !== system) return false
    }
    if (!search) return true
    return multiWordMatch(search, [row.itemNumber, row.description, row.id])
  })

  function updateContext(key: 'device' | 'scope' | 'system', value: string) {
    const next = new URLSearchParams(searchParams)
    if (value.trim() === '') next.delete(key)
    else next.set(key, value)
    if (key !== 'scope') next.delete('scope')
    setSearchParams(next, { replace: true })
  }

  async function refreshReservations() {
    await Promise.all([mutate(['reservations', device, scope]), mutate(['scope-overview', device])])
  }

  async function handleCreate(event: FormEvent<HTMLFormElement>) {
    event.preventDefault()
    setIsCreating(true)
    setFeedback(null)
    try {
      await createReservation({
        itemId: formItemId,
        deviceScopeId: formScopeId,
        quantity,
        requestedBy: actorId,
        purpose,
        priority,
        neededByAt,
        plannedUseAt,
        holdUntilAt,
        note,
      })
      await refreshReservations()
      setFeedback({ tone: 'success', text: 'Reservation created.' })
      setQuantity(1)
      setNote('')
    } catch (caught) {
      setFeedback({ tone: 'error', text: caught instanceof Error ? caught.message : 'Failed to create reservation.' })
    } finally {
      setIsCreating(false)
    }
  }

  async function handleBulkFile(event: ChangeEvent<HTMLInputElement>) {
    const file = event.target.files?.[0]
    event.target.value = ''
    if (!file) return
    setIsBulkCreating(true)
    setFeedback(null)
    try {
      const preview = await previewReservationsImport(file)
      setCsvKind('reservations')
      setCsvFile(file)
      setCsvPreview(preview)
      setCsvDialogOpen(true)
    } catch (caught) {
      setFeedback({ tone: 'error', text: caught instanceof Error ? caught.message : 'Failed to preview reservation CSV.' })
    } finally {
      setIsBulkCreating(false)
    }
  }

  async function handleAllocationBulkFile(event: ChangeEvent<HTMLInputElement>) {
    const file = event.target.files?.[0]
    event.target.value = ''
    if (!file) return
    setIsBulkAllocating(true)
    setFeedback(null)
    try {
      const preview = await previewAllocationsImport(file)
      setCsvKind('allocations')
      setCsvFile(file)
      setCsvPreview(preview)
      setCsvDialogOpen(true)
    } catch (caught) {
      setFeedback({ tone: 'error', text: caught instanceof Error ? caught.message : 'Failed to preview allocation CSV.' })
    } finally {
      setIsBulkAllocating(false)
    }
  }

  async function handleApplyCSVImport() {
    if (!csvFile || !csvPreview || csvInvalidRows.length > 0) return
    const isAllocation = csvKind === 'allocations'
    if (isAllocation) setIsBulkAllocating(true)
    else setIsBulkCreating(true)
    setFeedback(null)
    try {
      const result = isAllocation
        ? await applyAllocationsImport(csvFile, actorId)
        : await applyReservationsImport(csvFile, actorId)
      await refreshReservations()
      setFeedback({ tone: 'success', text: `${csvKind} import completed. created=${result.created}, job=${result.jobId}.` })
      setCsvDialogOpen(false)
      setCsvFile(null)
      setCsvPreview(null)
    } catch (caught) {
      setFeedback({ tone: 'error', text: caught instanceof Error ? caught.message : `Failed to apply ${csvKind} CSV.` })
    } finally {
      setIsBulkAllocating(false)
      setIsBulkCreating(false)
    }
  }

  async function openDetail(id: string) {
    setDetailOpen(true)
    setLoadingDetailId(id)
    setFeedback(null)
    try {
      const result = await fetchReservationDetail(id)
      setDetail(result)
      setActionQuantity(Math.max(1, result.quantity - result.allocatedQuantity))
    } catch (caught) {
      setFeedback({ tone: 'error', text: caught instanceof Error ? caught.message : 'Failed to load reservation detail.' })
      setDetailOpen(false)
    } finally {
      setLoadingDetailId('')
    }
  }

  async function handleActionConfirm() {
    if (!detail || !actionDialog) return
    setActionBusy(true)
    setFeedback(null)
    try {
      const result = await reservationAction(detail.id, actionDialog.action, {
        locationCode: actionLocation,
        quantity: actionQuantity,
        reason: actionReason,
        actorId,
        note: actionReason,
      })
      setDetail(result)
      await refreshReservations()
      setActionDialog(null)
      setFeedback({ tone: 'success', text: `Reservation ${actionDialog.action} completed.` })
    } catch (caught) {
      setFeedback({ tone: 'error', text: caught instanceof Error ? caught.message : `Failed to ${actionDialog.action} reservation.` })
    } finally {
      setActionBusy(false)
    }
  }

  return (
    <div className="space-y-6 p-6">
      <div className="flex flex-wrap items-start justify-between gap-4">
        <div className="space-y-2">
          <h1 className="text-3xl font-bold tracking-tight">Reservations</h1>
          <p className="text-muted-foreground">Create, allocate, release, and audit scope reservations.</p>
        </div>
        <div className="flex flex-wrap gap-2">
          <Button variant="outline" size="sm" className="gap-2" onClick={() => downloadTextFile('reservations-template.csv', RESERVATION_TEMPLATE)}>
            <FileDown className="h-4 w-4" />
            Reservation Template
          </Button>
          <Button variant="outline" size="sm" className="gap-2" onClick={() => downloadTextFile('allocations-template.csv', ALLOCATION_TEMPLATE)}>
            <FileDown className="h-4 w-4" />
            Allocation Template
          </Button>
          <input ref={csvInputRef} type="file" accept=".csv,text/csv" className="hidden" onChange={(event) => void handleBulkFile(event)} />
          <Button variant="outline" size="sm" className="gap-2" disabled={isBulkCreating} onClick={() => csvInputRef.current?.click()}>
            {isBulkCreating ? <Loader2 className="h-4 w-4 animate-spin" /> : <FileUp className="h-4 w-4" />}
            Upload Reservations
          </Button>
          <input ref={allocationCsvInputRef} type="file" accept=".csv,text/csv" className="hidden" onChange={(event) => void handleAllocationBulkFile(event)} />
          <Button variant="outline" size="sm" className="gap-2" disabled={isBulkAllocating} onClick={() => allocationCsvInputRef.current?.click()}>
            {isBulkAllocating ? <Loader2 className="h-4 w-4 animate-spin" /> : <FileUp className="h-4 w-4" />}
            Upload Allocations
          </Button>
          <Button variant="outline" size="sm" className="gap-2" onClick={() => void exportReservationsCSV(device || undefined, scope || undefined)}>
            <Download className="h-4 w-4" />
            Export CSV
          </Button>
        </div>
      </div>

      {feedback && (
        <div className={`rounded-lg border px-4 py-3 text-sm ${feedback.tone === 'success' ? 'border-green-200 bg-green-50 text-green-800' : 'border-red-200 bg-red-50 text-red-800'}`}>
          {feedback.text}
        </div>
      )}

      <Card>
        <CardContent className="pt-6">
          <DeviceScopeFilters
            device={device}
            scope={scope}
            system={system}
            onDeviceChange={(value) => updateContext('device', value)}
            onScopeChange={(value) => updateContext('scope', value)}
            onSystemChange={(value) => updateContext('system', value)}
          />
        </CardContent>
      </Card>

      <div className="grid gap-6 xl:grid-cols-[minmax(0,1fr)_390px]">
        <Card>
          <CardHeader>
            <div className="flex flex-wrap items-center justify-between gap-3">
              <div>
                <CardTitle>Reservation List ({filteredRows.length})</CardTitle>
                <CardDescription>Open a row for allocation and lifecycle actions.</CardDescription>
              </div>
              <Input className="w-full sm:w-72" placeholder="Filter reservations..." value={search} onChange={(event) => setSearch(event.target.value)} />
            </div>
          </CardHeader>
          <CardContent>
            <div className="overflow-x-auto">
              <Table>
                <TableHeader>
                  <TableRow>
                    <TableHead>ID</TableHead>
                    <TableHead>Item</TableHead>
                    <TableHead>Description</TableHead>
                    <TableHead className="text-right">Qty</TableHead>
                    <TableHead>Device</TableHead>
                    <TableHead>Scope</TableHead>
                    <TableHead>Status</TableHead>
                    <TableHead className="text-right">Actions</TableHead>
                  </TableRow>
                </TableHeader>
                <TableBody>
                  {filteredRows.map((row) => (
                    <TableRow key={row.id}>
                      <TableCell className="font-mono text-xs">{row.id}</TableCell>
                      <TableCell><ItemInfoPopover itemNumber={row.itemNumber} description={row.description} /></TableCell>
                      <TableCell className="max-w-[14rem] truncate text-sm text-muted-foreground">{row.description}</TableCell>
                      <TableCell className="text-right tabular-nums">{row.quantity}</TableCell>
                      <TableCell className="font-mono text-sm">{row.device}</TableCell>
                      <TableCell className="font-mono text-sm">{row.scope}</TableCell>
                      <TableCell><Badge variant={getStatusBadgeVariant(row.status)}>{row.status.replace(/_/g, ' ')}</Badge></TableCell>
                      <TableCell className="text-right">
                        <Button variant="outline" size="sm" className="gap-2" onClick={() => void openDetail(row.id)}>
                          <Eye className="h-4 w-4" />
                          Open
                        </Button>
                      </TableCell>
                    </TableRow>
                  ))}
                  {filteredRows.length === 0 && (
                    <TableRow>
                      <TableCell colSpan={8} className="py-8 text-center text-muted-foreground">No reservations found.</TableCell>
                    </TableRow>
                  )}
                </TableBody>
              </Table>
            </div>
          </CardContent>
        </Card>

        <Card>
          <CardHeader>
            <CardTitle className="flex items-center gap-2"><PackagePlus className="h-5 w-5" />New Reservation</CardTitle>
            <CardDescription>Create a reservation for a selected scope and item.</CardDescription>
          </CardHeader>
          <CardContent>
            <form onSubmit={handleCreate} className="space-y-4">
              <div className="space-y-2">
                <Label htmlFor="reservation-scope">Scope</Label>
                <Select value={formScopeId} onValueChange={setFormScopeId}>
                  <SelectTrigger id="reservation-scope"><SelectValue placeholder="Select scope" /></SelectTrigger>
                  <SelectContent>
                    {activeScopes.map((row) => (
                      <SelectItem key={row.id} value={row.id}>{row.deviceKey} / {row.scopeName || row.scopeKey}</SelectItem>
                    ))}
                  </SelectContent>
                </Select>
              </div>
              <div className="space-y-2">
                <Label htmlFor="reservation-item">Item</Label>
                <Select value={formItemId} onValueChange={setFormItemId}>
                  <SelectTrigger id="reservation-item"><SelectValue placeholder="Select item" /></SelectTrigger>
                  <SelectContent>
                    {items.map((row) => (
                      <SelectItem key={row.itemId} value={row.itemId}>{row.itemNumber} / {row.description}</SelectItem>
                    ))}
                  </SelectContent>
                </Select>
              </div>
              <div className="grid grid-cols-2 gap-3">
                <div className="space-y-2">
                  <Label htmlFor="reservation-qty">Quantity</Label>
                  <Input id="reservation-qty" type="number" min={1} value={quantity} onChange={(event) => setQuantity(Math.max(1, Number(event.target.value) || 1))} />
                </div>
                <div className="space-y-2">
                  <Label htmlFor="reservation-priority">Priority</Label>
                  <Select value={priority} onValueChange={setPriority}>
                    <SelectTrigger id="reservation-priority"><SelectValue /></SelectTrigger>
                    <SelectContent>
                      <SelectItem value="low">Low</SelectItem>
                      <SelectItem value="normal">Normal</SelectItem>
                      <SelectItem value="high">High</SelectItem>
                      <SelectItem value="urgent">Urgent</SelectItem>
                    </SelectContent>
                  </Select>
                </div>
              </div>
              <div className="grid grid-cols-3 gap-3">
                <div className="space-y-2"><Label htmlFor="needed-by">Needed</Label><Input id="needed-by" type="date" value={neededByAt} onChange={(event) => setNeededByAt(event.target.value)} /></div>
                <div className="space-y-2"><Label htmlFor="planned-use">Use</Label><Input id="planned-use" type="date" value={plannedUseAt} onChange={(event) => setPlannedUseAt(event.target.value)} /></div>
                <div className="space-y-2"><Label htmlFor="hold-until">Hold</Label><Input id="hold-until" type="date" value={holdUntilAt} onChange={(event) => setHoldUntilAt(event.target.value)} /></div>
              </div>
              <div className="space-y-2">
                <Label htmlFor="reservation-purpose">Purpose</Label>
                <Input id="reservation-purpose" value={purpose} onChange={(event) => setPurpose(event.target.value)} />
              </div>
              <div className="space-y-2">
                <Label htmlFor="reservation-note">Note</Label>
                <Textarea id="reservation-note" value={note} onChange={(event) => setNote(event.target.value)} />
              </div>
              <Button type="submit" className="w-full gap-2" disabled={!formScopeId || !formItemId || isCreating}>
                {isCreating ? <Loader2 className="h-4 w-4 animate-spin" /> : <CalendarCheck className="h-4 w-4" />}
                Create Reservation
              </Button>
            </form>
          </CardContent>
        </Card>
      </div>

      <Dialog open={csvDialogOpen} onOpenChange={setCsvDialogOpen}>
        <DialogContent className="max-w-4xl">
          <DialogHeader>
            <DialogTitle>CSV Import Preview</DialogTitle>
            <DialogDescription>Review validation results before the backend applies this import as one job.</DialogDescription>
          </DialogHeader>
          <div className="space-y-4">
            <div className="grid grid-cols-3 gap-3 text-sm">
              <div className="rounded border p-3"><p className="text-muted-foreground">Type</p><p className="font-medium">{csvKind}</p></div>
              <div className="rounded border p-3"><p className="text-muted-foreground">Rows</p><p className="font-medium">{csvPreview?.rows.length ?? 0}</p></div>
              <div className="rounded border p-3"><p className="text-muted-foreground">Invalid</p><p className="font-medium">{csvInvalidRows.length}</p></div>
            </div>
            <div className="max-h-[50vh] overflow-auto rounded-md border">
              <Table>
                <TableHeader>
                  <TableRow>
                    <TableHead>Row</TableHead>
                    <TableHead>Status</TableHead>
                    <TableHead>Code</TableHead>
                    <TableHead>Message</TableHead>
                  </TableRow>
                </TableHeader>
                <TableBody>
                  {(csvPreview?.rows ?? []).map((row) => (
                    <TableRow key={row.rowNumber}>
                      <TableCell>{row.rowNumber}</TableCell>
                      <TableCell><Badge variant={row.status === 'invalid' ? 'destructive' : 'default'}>{row.status}</Badge></TableCell>
                      <TableCell>{row.code || '—'}</TableCell>
                      <TableCell>{row.message || 'Ready'}</TableCell>
                    </TableRow>
                  ))}
                </TableBody>
              </Table>
            </div>
          </div>
          <DialogFooter>
            <Button variant="outline" onClick={() => setCsvDialogOpen(false)}>Cancel</Button>
            <Button
              className="gap-2"
              disabled={csvInvalidRows.length > 0 || isBulkCreating || isBulkAllocating}
              onClick={() => void handleApplyCSVImport()}
            >
              {(isBulkCreating || isBulkAllocating) && <Loader2 className="h-4 w-4 animate-spin" />}
              Apply Import
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>

      <Sheet open={detailOpen} onOpenChange={setDetailOpen}>
        <SheetContent side="right" className="w-full overflow-y-auto sm:max-w-2xl">
          <SheetHeader>
            <SheetTitle>Reservation Detail</SheetTitle>
            <SheetDescription>Allocation status, reservation metadata, and lifecycle actions.</SheetDescription>
          </SheetHeader>
          {loadingDetailId ? (
            <div className="mt-8 flex items-center gap-2 text-sm text-muted-foreground"><Loader2 className="h-4 w-4 animate-spin" />Loading...</div>
          ) : detail ? (
            <div className="mt-6 space-y-6">
              <div className="grid gap-3 md:grid-cols-2">
                <div className="rounded border p-4"><p className="text-xs uppercase text-muted-foreground">Item</p><p className="mt-2 font-medium">{detail.itemNumber}</p><p className="text-sm text-muted-foreground">{detail.description}</p></div>
                <div className="rounded border p-4"><p className="text-xs uppercase text-muted-foreground">Scope</p><p className="mt-2 font-medium">{detail.device} / {detail.scope}</p></div>
                <div className="rounded border p-4"><p className="text-xs uppercase text-muted-foreground">Quantity</p><p className="mt-2 font-medium">{detail.allocatedQuantity} / {detail.quantity}</p></div>
                <div className="rounded border p-4"><p className="text-xs uppercase text-muted-foreground">Status</p><Badge className="mt-2" variant={getStatusBadgeVariant(detail.status)}>{detail.status}</Badge></div>
              </div>
              <div className="flex flex-wrap gap-2">
                {(['allocate', 'release', 'fulfill', 'cancel', 'undo'] as const).map((action) => (
                  <Button key={action} variant="outline" size="sm" onClick={() => setActionDialog({ action })}>{action}</Button>
                ))}
              </div>
              <Card>
                <CardHeader>
                  <CardTitle className="text-base">Allocations</CardTitle>
                </CardHeader>
                <CardContent>
                  <Table>
                    <TableHeader><TableRow><TableHead>Location</TableHead><TableHead className="text-right">Qty</TableHead><TableHead>Status</TableHead><TableHead>Allocated</TableHead><TableHead>Released</TableHead></TableRow></TableHeader>
                    <TableBody>
                      {detail.allocations.map((allocation) => (
                        <TableRow key={allocation.id}>
                          <TableCell>{allocation.locationCode}</TableCell>
                          <TableCell className="text-right">{allocation.quantity}</TableCell>
                          <TableCell><Badge variant={getStatusBadgeVariant(allocation.status)}>{allocation.status}</Badge></TableCell>
                          <TableCell>{allocation.allocatedAt || '—'}</TableCell>
                          <TableCell>{allocation.releasedAt || '—'}</TableCell>
                        </TableRow>
                      ))}
                      {detail.allocations.length === 0 && <TableRow><TableCell colSpan={5} className="py-6 text-center text-muted-foreground">No allocations yet.</TableCell></TableRow>}
                    </TableBody>
                  </Table>
                </CardContent>
              </Card>
            </div>
          ) : null}
        </SheetContent>
      </Sheet>

      <Dialog open={Boolean(actionDialog)} onOpenChange={(open) => !open && setActionDialog(null)}>
        <DialogContent>
          <DialogHeader>
            <DialogTitle>{actionDialog?.action} Reservation</DialogTitle>
            <DialogDescription>Confirm the reservation action. Allocation needs a location and quantity.</DialogDescription>
          </DialogHeader>
          <div className="space-y-4">
            {actionDialog?.action === 'allocate' && (
              <div className="space-y-2">
                <Label htmlFor="action-location">Location</Label>
                <Input id="action-location" value={actionLocation} onChange={(event) => setActionLocation(event.target.value)} placeholder="TOKYO-A1" />
              </div>
            )}
            <div className="space-y-2">
              <Label htmlFor="action-quantity">Quantity</Label>
              <Input id="action-quantity" type="number" min={1} value={actionQuantity} onChange={(event) => setActionQuantity(Math.max(1, Number(event.target.value) || 1))} />
            </div>
            <div className="space-y-2">
              <Label htmlFor="action-reason">Reason / Note</Label>
              <Textarea id="action-reason" value={actionReason} onChange={(event) => setActionReason(event.target.value)} />
            </div>
          </div>
          <DialogFooter>
            <Button variant="outline" onClick={() => setActionDialog(null)}>Cancel</Button>
            <Button className="gap-2" disabled={actionBusy || (actionDialog?.action === 'allocate' && !actionLocation)} onClick={() => void handleActionConfirm()}>
              {actionBusy && <Loader2 className="h-4 w-4 animate-spin" />}
              Confirm
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>
    </div>
  )
}
