import type { ChangeEvent, FormEvent } from 'react'
import { useMemo, useRef, useState } from 'react'
import { useSearchParams } from 'react-router-dom'
import { useSWRConfig } from 'swr'
import { CheckCircle2, Download, FileDown, FileUp, Loader2, PackageCheck, SearchCheck, Upload } from 'lucide-react'

import { DeviceScopeFilters } from '@/components/context/DeviceScopeFilters'
import { ItemInfoPopover } from '@/components/ItemInfoPopover'
import { Badge } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card'
import { Dialog, DialogContent, DialogDescription, DialogFooter, DialogHeader, DialogTitle } from '@/components/ui/dialog'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from '@/components/ui/select'
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from '@/components/ui/table'
import { Textarea } from '@/components/ui/textarea'
import { useAuthSession } from '@/hooks/useAuthSession'
import { useDeviceScopes } from '@/hooks/useDeviceScopes'
import { useInventoryOverview } from '@/hooks/useInventoryOverview'
import { useRequirements } from '@/hooks/useRequirements'
import { applyRequirementsImport, confirmBulkReservation, exportRequirementsCSV, fetchBulkReservationPreview, previewRequirementsImport } from '@/lib/additionalApi'
import { downloadTextFile } from '@/lib/csv'
import { upsertRequirement } from '@/lib/mockApi'
import type { BulkReservationPreviewResponse, RequirementSummary, RequirementsImportPreviewResponse } from '@/types'

const REQUIREMENTS_TEMPLATE =
  'device,scope,manufacturer,item_number,description,quantity,note\nER2,powerboard,Omron,ER2,Control relay,10,Initial build demand\n'

function todayDate() {
  return new Date().toISOString().slice(0, 10)
}

function getBadgeVariant(status: string) {
  const normalized = status.toLowerCase()
  if (['valid', 'ready', 'created', 'updated'].includes(normalized)) return 'default' as const
  if (['missing_item', 'warning', 'skipped'].includes(normalized)) return 'secondary' as const
  if (['invalid', 'error', 'errored'].includes(normalized)) return 'destructive' as const
  return 'outline' as const
}

export function OperatorDashboardPage() {
  const [searchParams, setSearchParams] = useSearchParams()
  const device = searchParams.get('device') ?? ''
  const scope = searchParams.get('scope') ?? ''
  const system = searchParams.get('system') ?? ''
  const { mutate } = useSWRConfig()
  const { data: session } = useAuthSession()
  const { data: requirements } = useRequirements(device, scope)
  const { data: scopeData } = useDeviceScopes()
  const { data: inventory } = useInventoryOverview()
  const fileInputRef = useRef<HTMLInputElement>(null)

  const [selectedRequirement, setSelectedRequirement] = useState<RequirementSummary | null>(null)
  const [formScopeId, setFormScopeId] = useState('')
  const [formItemId, setFormItemId] = useState('')
  const [formQuantity, setFormQuantity] = useState(1)
  const [formNote, setFormNote] = useState('')
  const [search, setSearch] = useState('')
  const [feedback, setFeedback] = useState<{ tone: 'success' | 'error'; text: string } | null>(null)
  const [isSaving, setIsSaving] = useState(false)
  const [selectedFile, setSelectedFile] = useState<File | null>(null)
  const [importPreview, setImportPreview] = useState<RequirementsImportPreviewResponse | null>(null)
  const [isPreviewing, setIsPreviewing] = useState(false)
  const [isApplyingImport, setIsApplyingImport] = useState(false)
  const [bulkPreview, setBulkPreview] = useState<BulkReservationPreviewResponse | null>(null)
  const [bulkDialogOpen, setBulkDialogOpen] = useState(false)
  const [isBulkLoading, setIsBulkLoading] = useState(false)
  const [isBulkConfirming, setIsBulkConfirming] = useState(false)

  const scopes = scopeData?.rows ?? []
  const items = useMemo(() => {
    const byId = new Map((inventory?.balances ?? []).map((row) => [row.itemId, row]))
    return Array.from(byId.values()).sort((left, right) => left.itemNumber.localeCompare(right.itemNumber))
  }, [inventory?.balances])
  const activeScopes = scopes.filter((row) => row.status !== 'inactive')
  const selectedScope = activeScopes.find((row) => row.id === formScopeId)
  const actorId = session?.user?.userId || 'local-user'

  const filteredRows = (requirements?.rows ?? []).filter((row) => {
    if (!search) return true
    const q = search.toLowerCase()
    return (
      row.device.toLowerCase().includes(q) ||
      row.scope.toLowerCase().includes(q) ||
      row.itemNumber.toLowerCase().includes(q) ||
      row.description.toLowerCase().includes(q) ||
      row.note.toLowerCase().includes(q)
    )
  })

  const importSummary = useMemo(() => {
    const rows = importPreview?.rows ?? []
    return {
      total: rows.length,
      valid: rows.filter((row) => row.status === 'valid').length,
      missingItems: rows.filter((row) => !row.itemRegistered).length,
      invalid: rows.filter((row) => row.status !== 'valid').length,
    }
  }, [importPreview])

  function updateContext(key: 'device' | 'scope' | 'system', value: string) {
    const next = new URLSearchParams(searchParams)
    if (value.trim() === '') next.delete(key)
    else next.set(key, value)
    if (key !== 'scope') next.delete('scope')
    setSearchParams(next, { replace: true })
  }

  function startEdit(row: RequirementSummary) {
    const matchedScope = scopes.find((candidate) => candidate.deviceKey === row.device && candidate.scopeKey === row.scope)
    setSelectedRequirement(row)
    setFormScopeId(matchedScope?.id ?? '')
    setFormItemId(row.itemId)
    setFormQuantity(row.quantity)
    setFormNote(row.note)
  }

  function resetForm() {
    setSelectedRequirement(null)
    setFormScopeId('')
    setFormItemId('')
    setFormQuantity(1)
    setFormNote('')
  }

  async function handleRequirementSubmit(event: FormEvent<HTMLFormElement>) {
    event.preventDefault()
    setIsSaving(true)
    setFeedback(null)
    try {
      await upsertRequirement({
        id: selectedRequirement?.id,
        deviceScopeId: formScopeId,
        itemId: formItemId,
        quantity: formQuantity,
        note: formNote,
      })
      await Promise.all([mutate(['requirements', device, scope]), mutate(['scope-overview', device])])
      setFeedback({ tone: 'success', text: selectedRequirement ? 'Requirement updated.' : 'Requirement created.' })
      resetForm()
    } catch (caught) {
      setFeedback({ tone: 'error', text: caught instanceof Error ? caught.message : 'Failed to save requirement.' })
    } finally {
      setIsSaving(false)
    }
  }

  function handleFileChange(event: ChangeEvent<HTMLInputElement>) {
    const file = event.target.files?.[0] ?? null
    setSelectedFile(file)
    setImportPreview(null)
    setFeedback(null)
    event.target.value = ''
  }

  async function handlePreviewImport() {
    if (!selectedFile) return
    setIsPreviewing(true)
    setFeedback(null)
    try {
      const result = await previewRequirementsImport(selectedFile)
      setImportPreview(result)
      setFeedback({
        tone: importSummary.invalid > 0 ? 'error' : 'success',
        text: `Preview generated for ${result.fileName}.`,
      })
    } catch (caught) {
      setFeedback({ tone: 'error', text: caught instanceof Error ? caught.message : 'Failed to preview requirements CSV.' })
    } finally {
      setIsPreviewing(false)
    }
  }

  async function handleApplyImport() {
    if (!selectedFile || !importPreview || importSummary.invalid > 0) return
    setIsApplyingImport(true)
    setFeedback(null)
    try {
      const result = await applyRequirementsImport(selectedFile)
      await Promise.all([mutate(['requirements', device, scope]), mutate(['scope-overview', device])])
      setFeedback({
        tone: result.errored > 0 ? 'error' : 'success',
        text: `Requirements import finished. created=${result.created}, updated=${result.updated}, skipped=${result.skipped}, errored=${result.errored}.`,
      })
      setSelectedFile(null)
      setImportPreview(null)
    } catch (caught) {
      setFeedback({ tone: 'error', text: caught instanceof Error ? caught.message : 'Failed to apply requirements CSV.' })
    } finally {
      setIsApplyingImport(false)
    }
  }

  async function handleBulkPreview() {
    const targetScopeId = selectedScope?.id || scopes.find((row) => row.deviceKey === device && row.scopeKey === scope)?.id || ''
    if (!targetScopeId) {
      setFeedback({ tone: 'error', text: 'Select a single scope before generating bulk reservation preview.' })
      return
    }
    setIsBulkLoading(true)
    setFeedback(null)
    try {
      const result = await fetchBulkReservationPreview(targetScopeId)
      setBulkPreview(result)
      setBulkDialogOpen(true)
    } catch (caught) {
      setFeedback({ tone: 'error', text: caught instanceof Error ? caught.message : 'Failed to generate bulk reservation preview.' })
    } finally {
      setIsBulkLoading(false)
    }
  }

  async function handleBulkConfirm() {
    if (!bulkPreview) return
    setIsBulkConfirming(true)
    setFeedback(null)
    try {
      const result = await confirmBulkReservation({
        scopeId: bulkPreview.scopeId,
        actorId,
        rows: bulkPreview.rows
          .filter((row) => row.allocFromStock > 0 || row.allocFromOrders > 0)
          .map((row) => ({
            itemId: row.itemId,
            stockAllocations: row.allocFromStockLocs,
            orderAllocations: row.allocFromOrderLocs,
            purpose: 'Requirement bulk reservation',
            priority: 'normal',
            neededByAt: todayDate(),
          })),
      })
      await Promise.all([mutate(['reservations', device, scope]), mutate(['requirements', device, scope]), mutate(['scope-overview', device])])
      setBulkDialogOpen(false)
      setFeedback({ tone: 'success', text: `Created ${result.created} reservations.` })
    } catch (caught) {
      setFeedback({ tone: 'error', text: caught instanceof Error ? caught.message : 'Failed to confirm bulk reservation.' })
    } finally {
      setIsBulkConfirming(false)
    }
  }

  return (
    <div className="space-y-6 p-6">
      <div className="flex flex-wrap items-start justify-between gap-4">
        <div className="space-y-2">
          <h1 className="text-3xl font-bold tracking-tight">Requirements</h1>
          <p className="text-muted-foreground">Manage scope-level item demand and convert demand into reservations.</p>
        </div>
        <div className="flex flex-wrap gap-2">
          <Button variant="outline" size="sm" className="gap-2" onClick={() => downloadTextFile('requirements-template.csv', REQUIREMENTS_TEMPLATE)}>
            <FileDown className="h-4 w-4" />
            Template
          </Button>
          <Button variant="outline" size="sm" className="gap-2" onClick={() => void exportRequirementsCSV(device || undefined, scope || undefined)}>
            <Download className="h-4 w-4" />
            Export CSV
          </Button>
          <Button size="sm" className="gap-2" disabled={isBulkLoading} onClick={() => void handleBulkPreview()}>
            {isBulkLoading ? <Loader2 className="h-4 w-4 animate-spin" /> : <PackageCheck className="h-4 w-4" />}
            Bulk Reserve
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

      <div className="grid gap-6 xl:grid-cols-[minmax(0,1fr)_380px]">
        <Card>
          <CardHeader>
            <div className="flex flex-wrap items-center justify-between gap-3">
              <div>
                <CardTitle>Requirement List ({filteredRows.length})</CardTitle>
                <CardDescription>Click a row to edit the requirement.</CardDescription>
              </div>
              <Input className="w-full sm:w-72" placeholder="Filter requirements..." value={search} onChange={(event) => setSearch(event.target.value)} />
            </div>
          </CardHeader>
          <CardContent>
            <div className="overflow-x-auto">
              <Table>
                <TableHeader>
                  <TableRow>
                    <TableHead>Device</TableHead>
                    <TableHead>Scope</TableHead>
                    <TableHead>Item</TableHead>
                    <TableHead>Description</TableHead>
                    <TableHead className="text-right">Qty</TableHead>
                    <TableHead>Note</TableHead>
                  </TableRow>
                </TableHeader>
                <TableBody>
                  {filteredRows.map((row) => (
                    <TableRow key={row.id} className="cursor-pointer hover:bg-muted/50" onClick={() => startEdit(row)}>
                      <TableCell className="font-mono text-sm">{row.device}</TableCell>
                      <TableCell className="font-mono text-sm">{row.scope}</TableCell>
                      <TableCell>
                        <ItemInfoPopover itemNumber={row.itemNumber} description={row.description} />
                      </TableCell>
                      <TableCell className="max-w-[16rem] truncate text-sm text-muted-foreground">{row.description}</TableCell>
                      <TableCell className="text-right tabular-nums">{row.quantity}</TableCell>
                      <TableCell className="max-w-[18rem] truncate text-sm text-muted-foreground">{row.note || '—'}</TableCell>
                    </TableRow>
                  ))}
                  {filteredRows.length === 0 && (
                    <TableRow>
                      <TableCell colSpan={6} className="py-8 text-center text-muted-foreground">
                        No requirements found.
                      </TableCell>
                    </TableRow>
                  )}
                </TableBody>
              </Table>
            </div>
          </CardContent>
        </Card>

        <div className="space-y-6">
          <Card>
            <CardHeader>
              <CardTitle>{selectedRequirement ? 'Edit Requirement' : 'New Requirement'}</CardTitle>
              <CardDescription>Set item demand for a device scope.</CardDescription>
            </CardHeader>
            <CardContent>
              <form onSubmit={handleRequirementSubmit} className="space-y-4">
                <div className="space-y-2">
                  <Label htmlFor="requirement-scope">Scope</Label>
                  <Select value={formScopeId} onValueChange={setFormScopeId}>
                    <SelectTrigger id="requirement-scope">
                      <SelectValue placeholder="Select scope" />
                    </SelectTrigger>
                    <SelectContent>
                      {activeScopes.map((row) => (
                        <SelectItem key={row.id} value={row.id}>
                          {row.deviceKey} / {row.scopeName || row.scopeKey}
                        </SelectItem>
                      ))}
                    </SelectContent>
                  </Select>
                </div>
                <div className="space-y-2">
                  <Label htmlFor="requirement-item">Item</Label>
                  <Select value={formItemId} onValueChange={setFormItemId}>
                    <SelectTrigger id="requirement-item">
                      <SelectValue placeholder="Select item" />
                    </SelectTrigger>
                    <SelectContent>
                      {items.map((row) => (
                        <SelectItem key={row.itemId} value={row.itemId}>
                          {row.itemNumber} / {row.description}
                        </SelectItem>
                      ))}
                    </SelectContent>
                  </Select>
                </div>
                <div className="space-y-2">
                  <Label htmlFor="requirement-quantity">Quantity</Label>
                  <Input id="requirement-quantity" type="number" min={1} value={formQuantity} onChange={(event) => setFormQuantity(Math.max(1, Number(event.target.value) || 1))} />
                </div>
                <div className="space-y-2">
                  <Label htmlFor="requirement-note">Note</Label>
                  <Textarea id="requirement-note" value={formNote} onChange={(event) => setFormNote(event.target.value)} />
                </div>
                <div className="flex gap-2">
                  <Button type="submit" className="flex-1" disabled={!formScopeId || !formItemId || isSaving}>
                    {isSaving ? 'Saving...' : selectedRequirement ? 'Update' : 'Create'}
                  </Button>
                  <Button type="button" variant="outline" onClick={resetForm}>
                    Clear
                  </Button>
                </div>
              </form>
            </CardContent>
          </Card>

          <Card>
            <CardHeader>
              <CardTitle className="flex items-center gap-2">
                <Upload className="h-5 w-5" />
                Requirements CSV
              </CardTitle>
              <CardDescription>Preview and apply requirement rows in bulk.</CardDescription>
            </CardHeader>
            <CardContent className="space-y-4">
              <input ref={fileInputRef} type="file" accept=".csv,text/csv" className="hidden" onChange={handleFileChange} />
              <div className="flex flex-wrap gap-2">
                <Button variant="outline" className="gap-2" onClick={() => fileInputRef.current?.click()}>
                  <FileUp className="h-4 w-4" />
                  Choose CSV
                </Button>
                <Button className="gap-2" disabled={!selectedFile || isPreviewing} onClick={() => void handlePreviewImport()}>
                  {isPreviewing ? <Loader2 className="h-4 w-4 animate-spin" /> : <SearchCheck className="h-4 w-4" />}
                  Preview
                </Button>
                <Button variant="outline" className="gap-2" onClick={() => downloadTextFile('requirements-template.csv', REQUIREMENTS_TEMPLATE)}>
                  <FileDown className="h-4 w-4" />
                  Template
                </Button>
              </div>
              <p className="text-sm text-muted-foreground">{selectedFile ? selectedFile.name : 'No CSV selected.'}</p>
              {importPreview && (
                <div className="space-y-3">
                  <div className="grid grid-cols-4 gap-2 text-sm">
                    <div className="rounded border p-3"><p className="text-muted-foreground">Rows</p><p className="font-semibold">{importSummary.total}</p></div>
                    <div className="rounded border p-3"><p className="text-muted-foreground">Valid</p><p className="font-semibold">{importSummary.valid}</p></div>
                    <div className="rounded border p-3"><p className="text-muted-foreground">Missing items</p><p className="font-semibold">{importSummary.missingItems}</p></div>
                    <div className="rounded border p-3"><p className="text-muted-foreground">Invalid</p><p className="font-semibold">{importSummary.invalid}</p></div>
                  </div>
                  <div className="max-h-64 overflow-auto rounded-md border">
                    <Table>
                      <TableHeader>
                        <TableRow>
                          <TableHead>Row</TableHead>
                          <TableHead>Status</TableHead>
                          <TableHead>Scope</TableHead>
                          <TableHead>Item</TableHead>
                          <TableHead className="text-right">Qty</TableHead>
                          <TableHead>Message</TableHead>
                        </TableRow>
                      </TableHeader>
                      <TableBody>
                        {importPreview.rows.map((row) => (
                          <TableRow key={row.rowNumber}>
                            <TableCell>{row.rowNumber}</TableCell>
                            <TableCell><Badge variant={getBadgeVariant(row.status)}>{row.status}</Badge></TableCell>
                            <TableCell>{row.deviceKey} / {row.scopeKey}</TableCell>
                            <TableCell>{row.itemNumber}</TableCell>
                            <TableCell className="text-right">{row.quantity}</TableCell>
                            <TableCell>{row.message || (row.itemRegistered ? 'Ready' : 'Item needs registration')}</TableCell>
                          </TableRow>
                        ))}
                      </TableBody>
                    </Table>
                  </div>
                  <Button className="gap-2" disabled={isApplyingImport || importSummary.invalid > 0} onClick={() => void handleApplyImport()}>
                    {isApplyingImport ? <Loader2 className="h-4 w-4 animate-spin" /> : <CheckCircle2 className="h-4 w-4" />}
                    Apply Requirements
                  </Button>
                </div>
              )}
            </CardContent>
          </Card>
        </div>
      </div>

      <Dialog open={bulkDialogOpen} onOpenChange={setBulkDialogOpen}>
        <DialogContent className="max-w-5xl">
          <DialogHeader>
            <DialogTitle>Bulk Reservation Preview</DialogTitle>
            <DialogDescription>Review stock and incoming-order allocations before creating reservations.</DialogDescription>
          </DialogHeader>
          <div className="max-h-[60vh] overflow-auto">
            <Table>
              <TableHeader>
                <TableRow>
                  <TableHead>Item</TableHead>
                  <TableHead className="text-right">Required</TableHead>
                  <TableHead className="text-right">From stock</TableHead>
                  <TableHead className="text-right">From orders</TableHead>
                  <TableHead className="text-right">Unallocated</TableHead>
                  <TableHead>Sources</TableHead>
                </TableRow>
              </TableHeader>
              <TableBody>
                {(bulkPreview?.rows ?? []).map((row) => (
                  <TableRow key={row.itemId}>
                    <TableCell>
                      <div className="font-medium">{row.itemNumber}</div>
                      <div className="text-xs text-muted-foreground">{row.description}</div>
                    </TableCell>
                    <TableCell className="text-right">{row.requiredQuantity}</TableCell>
                    <TableCell className="text-right">{row.allocFromStock}</TableCell>
                    <TableCell className="text-right">{row.allocFromOrders}</TableCell>
                    <TableCell className="text-right">{row.unallocated > 0 ? <Badge variant="destructive">{row.unallocated}</Badge> : 0}</TableCell>
                    <TableCell className="text-xs text-muted-foreground">
                      {[...row.allocFromStockLocs.map((loc) => `${loc.locationCode}:${loc.quantity}`), ...row.allocFromOrderLocs.map((order) => `${order.purchaseOrderNumber}:${order.quantity}@${order.expectedArrival}`)].join(' / ') || '—'}
                    </TableCell>
                  </TableRow>
                ))}
              </TableBody>
            </Table>
          </div>
          <DialogFooter>
            <Button variant="outline" onClick={() => setBulkDialogOpen(false)}>Cancel</Button>
            <Button className="gap-2" disabled={isBulkConfirming || !bulkPreview?.rows.length} onClick={() => void handleBulkConfirm()}>
              {isBulkConfirming ? <Loader2 className="h-4 w-4 animate-spin" /> : <PackageCheck className="h-4 w-4" />}
              Confirm Reservations
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>
    </div>
  )
}
