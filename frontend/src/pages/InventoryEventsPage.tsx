import type { ChangeEvent, FormEvent } from 'react'
import { useMemo, useRef, useState } from 'react'
import { useSWRConfig } from 'swr'
import { ArrowRightLeft, FileDown, FileUp, Loader2, PackageCheck, SlidersHorizontal } from 'lucide-react'

import { Badge } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card'
import { Dialog, DialogContent, DialogDescription, DialogFooter, DialogHeader, DialogTitle } from '@/components/ui/dialog'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from '@/components/ui/select'
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from '@/components/ui/table'
import { Tabs, TabsContent, TabsList, TabsTrigger } from '@/components/ui/tabs'
import { useAuthSession } from '@/hooks/useAuthSession'
import { useDeviceScopes } from '@/hooks/useDeviceScopes'
import { useInventoryOverview } from '@/hooks/useInventoryOverview'
import { resolveActorId } from '@/lib/auth'
import { applyInventoryOperationImport, previewInventoryOperationImport } from '@/lib/additionalApi'
import { downloadTextFile } from '@/lib/csv'
import { adjustInventory, moveInventory, receiveInventory } from '@/lib/mockApi'
import type { ImportPreviewResult } from '@/types'

const templates = {
  adjust: 'item_id,location_code,quantity_delta,device_scope_id,note\nitem-er2,TOKYO-A1,-2,ds-er2-powerboard,Stock correction\n',
  receive: 'item_id,location_code,quantity,device_scope_id,source_type,source_id,note\nitem-er2,TOKYO-A1,10,ds-er2-powerboard,manual,PO-001,Receipt\n',
  move: 'item_id,from_location_code,to_location_code,quantity,device_scope_id,source_type,source_id,note\nitem-er2,TOKYO-A1,TOKYO-B1,3,ds-er2-powerboard,manual,,Relocation\n',
}

type Operation = 'adjust' | 'receive' | 'move'

export function InventoryEventsPage() {
  const { data } = useInventoryOverview()
  const { data: deviceScopes } = useDeviceScopes()
  const { data: session } = useAuthSession()
  const { mutate } = useSWRConfig()
  const fileInputRef = useRef<HTMLInputElement>(null)
  const [operation, setOperation] = useState<Operation>('adjust')

  const [itemId, setItemId] = useState('item-er2')
  const [locationCode, setLocationCode] = useState('TOKYO-A1')
  const [toLocationCode, setToLocationCode] = useState('TOKYO-B1')
  const [quantityDelta, setQuantityDelta] = useState(-1)
  const [quantity, setQuantity] = useState(1)
  const [deviceScopeId, setDeviceScopeId] = useState('')
  const [sourceType, setSourceType] = useState('manual')
  const [sourceId, setSourceId] = useState('')
  const [note, setNote] = useState('Inventory operation')
  const [message, setMessage] = useState<{ tone: 'success' | 'error'; text: string } | null>(null)
  const [isSubmitting, setIsSubmitting] = useState(false)
  const [isBulkRunning, setIsBulkRunning] = useState(false)
  const [csvDialogOpen, setCsvDialogOpen] = useState(false)
  const [csvPreview, setCsvPreview] = useState<ImportPreviewResult | null>(null)
  const [csvFile, setCsvFile] = useState<File | null>(null)
  const [recentResults, setRecentResults] = useState<{ operation: string; count: number; at: string }[]>([])

  const activeDeviceScopes = deviceScopes?.rows.filter((row) => row.status !== 'inactive') ?? []
  const selectedDeviceScopeId = deviceScopeId || activeDeviceScopes[0]?.id || ''
  const actorId = resolveActorId(session)
  const items = useMemo(() => {
    const byId = new Map((data?.balances ?? []).map((row) => [row.itemId, row]))
    return Array.from(byId.values()).sort((left, right) => left.itemNumber.localeCompare(right.itemNumber))
  }, [data?.balances])
  const selectedItem = data?.balances.find((balance) => balance.itemId === itemId && balance.locationCode === locationCode)
  const csvInvalidRows = (csvPreview?.rows ?? []).filter((row) => row.status === 'invalid')

  async function afterOperation(operationName: string, count: number) {
    await mutate('inventory-overview')
    setRecentResults((current) => [{ operation: operationName, count, at: new Date().toLocaleString() }, ...current].slice(0, 8))
  }

  async function handleAdjustSubmit(event: FormEvent<HTMLFormElement>) {
    event.preventDefault()
    setIsSubmitting(true)
    setMessage(null)
    try {
      await adjustInventory({ itemId, locationCode, quantityDelta, deviceScopeId: selectedDeviceScopeId, note })
      await afterOperation('adjust', 1)
      setMessage({ tone: 'success', text: `Recorded adjustment ${quantityDelta} at ${locationCode}.` })
    } catch (caught) {
      setMessage({ tone: 'error', text: caught instanceof Error ? caught.message : 'Adjustment failed.' })
    } finally {
      setIsSubmitting(false)
    }
  }

  async function handleReceiveSubmit(event: FormEvent<HTMLFormElement>) {
    event.preventDefault()
    setIsSubmitting(true)
    setMessage(null)
    try {
      await receiveInventory({ itemId, locationCode, quantity, deviceScopeId: selectedDeviceScopeId, actorId, sourceType, sourceId, note })
      await afterOperation('receive', 1)
      setMessage({ tone: 'success', text: `Received ${quantity} units at ${locationCode}.` })
    } catch (caught) {
      setMessage({ tone: 'error', text: caught instanceof Error ? caught.message : 'Receive failed.' })
    } finally {
      setIsSubmitting(false)
    }
  }

  async function handleMoveSubmit(event: FormEvent<HTMLFormElement>) {
    event.preventDefault()
    setIsSubmitting(true)
    setMessage(null)
    try {
      await moveInventory({ itemId, fromLocationCode: locationCode, toLocationCode, quantity, deviceScopeId: selectedDeviceScopeId, actorId, sourceType, sourceId, note })
      await afterOperation('move', 1)
      setMessage({ tone: 'success', text: `Moved ${quantity} units from ${locationCode} to ${toLocationCode}.` })
    } catch (caught) {
      setMessage({ tone: 'error', text: caught instanceof Error ? caught.message : 'Move failed.' })
    } finally {
      setIsSubmitting(false)
    }
  }

  async function handleBulkFile(event: ChangeEvent<HTMLInputElement>) {
    const file = event.target.files?.[0]
    event.target.value = ''
    if (!file) return
    setIsBulkRunning(true)
    setMessage(null)
    try {
      const preview = await previewInventoryOperationImport(operation, file)
      setCsvFile(file)
      setCsvPreview(preview)
      setCsvDialogOpen(true)
    } catch (caught) {
      setMessage({ tone: 'error', text: caught instanceof Error ? caught.message : `CSV preview for ${operation} failed.` })
    } finally {
      setIsBulkRunning(false)
    }
  }

  async function handleApplyCSVImport() {
    if (!csvFile || !csvPreview || csvInvalidRows.length > 0) return
    setIsBulkRunning(true)
    setMessage(null)
    try {
      const result = await applyInventoryOperationImport(operation, csvFile, actorId)
      await afterOperation(operation, result.created)
      setMessage({ tone: 'success', text: `Applied ${result.created} ${operation} row(s) as import job ${result.jobId}.` })
      setCsvDialogOpen(false)
      setCsvFile(null)
      setCsvPreview(null)
    } catch (caught) {
      setMessage({ tone: 'error', text: caught instanceof Error ? caught.message : `CSV apply for ${operation} failed.` })
    } finally {
      setIsBulkRunning(false)
    }
  }

  function renderCommonFields() {
    return (
      <>
        <div className="space-y-2">
          <Label htmlFor={`${operation}-item`}>Item</Label>
          <Select value={itemId} onValueChange={setItemId}>
            <SelectTrigger id={`${operation}-item`}><SelectValue placeholder="Select item" /></SelectTrigger>
            <SelectContent>
              {items.map((row) => (
                <SelectItem key={row.itemId} value={row.itemId}>{row.itemNumber} / {row.description}</SelectItem>
              ))}
            </SelectContent>
          </Select>
        </div>
        <div className="space-y-2">
          <Label htmlFor={`${operation}-scope`}>Device Scope</Label>
          <Select value={selectedDeviceScopeId} onValueChange={setDeviceScopeId}>
            <SelectTrigger id={`${operation}-scope`}><SelectValue placeholder="Select scope" /></SelectTrigger>
            <SelectContent>
              {activeDeviceScopes.map((row) => (
                <SelectItem key={row.id} value={row.id}>{row.deviceKey} / {row.scopeName || row.scopeKey}</SelectItem>
              ))}
            </SelectContent>
          </Select>
        </div>
      </>
    )
  }

  return (
    <div className="space-y-6 p-6">
      <div className="flex flex-wrap items-start justify-between gap-4">
        <div>
          <h1 className="text-3xl font-bold tracking-tight">Inventory Events</h1>
          <p className="mt-2 text-muted-foreground">Record receive, move, and adjust operations for inventory items.</p>
        </div>
        <div className="flex flex-wrap gap-2">
          <Button variant="outline" size="sm" className="gap-2" onClick={() => downloadTextFile(`${operation}-template.csv`, templates[operation])}>
            <FileDown className="h-4 w-4" />
            Template
          </Button>
          <input ref={fileInputRef} type="file" accept=".csv,text/csv" className="hidden" onChange={(event) => void handleBulkFile(event)} />
          <Button variant="outline" size="sm" className="gap-2" disabled={isBulkRunning} onClick={() => fileInputRef.current?.click()}>
            {isBulkRunning ? <Loader2 className="h-4 w-4 animate-spin" /> : <FileUp className="h-4 w-4" />}
            Upload CSV
          </Button>
        </div>
      </div>

      {message && (
        <div className={`rounded-lg border px-4 py-3 text-sm ${message.tone === 'success' ? 'border-green-200 bg-green-50 text-green-800' : 'border-red-200 bg-red-50 text-red-800'}`}>
          {message.text}
        </div>
      )}

      <Tabs value={operation} onValueChange={(value) => setOperation(value as Operation)} className="space-y-4">
        <TabsList>
          <TabsTrigger value="adjust">Adjust</TabsTrigger>
          <TabsTrigger value="receive">Receive</TabsTrigger>
          <TabsTrigger value="move">Move</TabsTrigger>
        </TabsList>

        <TabsContent value="adjust">
          <Card>
            <CardHeader>
              <CardTitle className="flex items-center gap-2"><SlidersHorizontal className="h-5 w-5" />Adjust Inventory</CardTitle>
              <CardDescription>Record positive or negative corrections for a location.</CardDescription>
            </CardHeader>
            <CardContent>
              <form onSubmit={handleAdjustSubmit} className="space-y-5">
                {renderCommonFields()}
                {selectedItem && (
                  <div className="rounded-lg bg-muted p-4 text-sm">
                    Current at {locationCode}: on hand {selectedItem.onHandQuantity}, available {selectedItem.availableQuantity}
                  </div>
                )}
                <div className="grid gap-4 sm:grid-cols-2">
                  <div className="space-y-2"><Label htmlFor="adjust-location">Location</Label><Input id="adjust-location" value={locationCode} onChange={(event) => setLocationCode(event.target.value)} /></div>
                  <div className="space-y-2"><Label htmlFor="adjust-qty">Quantity Delta</Label><Input id="adjust-qty" type="number" value={quantityDelta} onChange={(event) => setQuantityDelta(Number(event.target.value) || 0)} /></div>
                </div>
                <div className="space-y-2"><Label htmlFor="adjust-note">Note</Label><Input id="adjust-note" value={note} onChange={(event) => setNote(event.target.value)} /></div>
                <Button type="submit" className="w-full" disabled={isSubmitting || !itemId || !locationCode || !selectedDeviceScopeId}>Record Adjustment</Button>
              </form>
            </CardContent>
          </Card>
        </TabsContent>

        <TabsContent value="receive">
          <Card>
            <CardHeader>
              <CardTitle className="flex items-center gap-2"><PackageCheck className="h-5 w-5" />Receive Inventory</CardTitle>
              <CardDescription>Add received inventory to a location.</CardDescription>
            </CardHeader>
            <CardContent>
              <form onSubmit={handleReceiveSubmit} className="space-y-5">
                {renderCommonFields()}
                <div className="grid gap-4 sm:grid-cols-2">
                  <div className="space-y-2"><Label htmlFor="receive-location">Location</Label><Input id="receive-location" value={locationCode} onChange={(event) => setLocationCode(event.target.value)} /></div>
                  <div className="space-y-2"><Label htmlFor="receive-qty">Quantity</Label><Input id="receive-qty" type="number" min={1} value={quantity} onChange={(event) => setQuantity(Math.max(1, Number(event.target.value) || 1))} /></div>
                </div>
                <div className="grid gap-4 sm:grid-cols-2">
                  <div className="space-y-2"><Label htmlFor="receive-source-type">Source Type</Label><Input id="receive-source-type" value={sourceType} onChange={(event) => setSourceType(event.target.value)} /></div>
                  <div className="space-y-2"><Label htmlFor="receive-source-id">Source ID</Label><Input id="receive-source-id" value={sourceId} onChange={(event) => setSourceId(event.target.value)} /></div>
                </div>
                <div className="space-y-2"><Label htmlFor="receive-note">Note</Label><Input id="receive-note" value={note} onChange={(event) => setNote(event.target.value)} /></div>
                <Button type="submit" className="w-full" disabled={isSubmitting || !itemId || !locationCode || !selectedDeviceScopeId}>Record Receipt</Button>
              </form>
            </CardContent>
          </Card>
        </TabsContent>

        <TabsContent value="move">
          <Card>
            <CardHeader>
              <CardTitle className="flex items-center gap-2"><ArrowRightLeft className="h-5 w-5" />Move Inventory</CardTitle>
              <CardDescription>Transfer inventory between two locations.</CardDescription>
            </CardHeader>
            <CardContent>
              <form onSubmit={handleMoveSubmit} className="space-y-5">
                {renderCommonFields()}
                <div className="grid gap-4 sm:grid-cols-3">
                  <div className="space-y-2"><Label htmlFor="from-location">From</Label><Input id="from-location" value={locationCode} onChange={(event) => setLocationCode(event.target.value)} /></div>
                  <div className="space-y-2"><Label htmlFor="to-location">To</Label><Input id="to-location" value={toLocationCode} onChange={(event) => setToLocationCode(event.target.value)} /></div>
                  <div className="space-y-2"><Label htmlFor="move-qty">Quantity</Label><Input id="move-qty" type="number" min={1} value={quantity} onChange={(event) => setQuantity(Math.max(1, Number(event.target.value) || 1))} /></div>
                </div>
                <div className="grid gap-4 sm:grid-cols-2">
                  <div className="space-y-2"><Label htmlFor="move-source-type">Source Type</Label><Input id="move-source-type" value={sourceType} onChange={(event) => setSourceType(event.target.value)} /></div>
                  <div className="space-y-2"><Label htmlFor="move-source-id">Source ID</Label><Input id="move-source-id" value={sourceId} onChange={(event) => setSourceId(event.target.value)} /></div>
                </div>
                <div className="space-y-2"><Label htmlFor="move-note">Note</Label><Input id="move-note" value={note} onChange={(event) => setNote(event.target.value)} /></div>
                <Button type="submit" className="w-full" disabled={isSubmitting || !itemId || !locationCode || !toLocationCode || !selectedDeviceScopeId}>Record Move</Button>
              </form>
            </CardContent>
          </Card>
        </TabsContent>
      </Tabs>

      <Dialog open={csvDialogOpen} onOpenChange={setCsvDialogOpen}>
        <DialogContent className="max-w-4xl">
          <DialogHeader>
            <DialogTitle>CSV Import Preview</DialogTitle>
            <DialogDescription>Review validation results before the backend applies this inventory import as one job.</DialogDescription>
          </DialogHeader>
          <div className="space-y-4">
            <div className="grid grid-cols-3 gap-3 text-sm">
              <div className="rounded border p-3"><p className="text-muted-foreground">Operation</p><p className="font-medium">{operation}</p></div>
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
            <Button className="gap-2" disabled={csvInvalidRows.length > 0 || isBulkRunning} onClick={() => void handleApplyCSVImport()}>
              {isBulkRunning && <Loader2 className="h-4 w-4 animate-spin" />}
              Apply Import
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>

      <Card>
        <CardHeader>
          <CardTitle>Recent Event Results</CardTitle>
          <CardDescription>Operations submitted from this screen.</CardDescription>
        </CardHeader>
        <CardContent>
          <Table>
            <TableHeader><TableRow><TableHead>Operation</TableHead><TableHead className="text-right">Rows</TableHead><TableHead>Submitted</TableHead></TableRow></TableHeader>
            <TableBody>
              {recentResults.map((row, index) => (
                <TableRow key={`${row.operation}-${row.at}-${index}`}>
                  <TableCell><Badge variant="outline">{row.operation}</Badge></TableCell>
                  <TableCell className="text-right">{row.count}</TableCell>
                  <TableCell>{row.at}</TableCell>
                </TableRow>
              ))}
              {recentResults.length === 0 && <TableRow><TableCell colSpan={3} className="py-8 text-center text-muted-foreground">No operations submitted yet.</TableCell></TableRow>}
            </TableBody>
          </Table>
        </CardContent>
      </Card>
    </div>
  )
}
