import type { ChangeEvent, FormEvent } from 'react'
import { useMemo, useRef, useState } from 'react'
import { Link, useLocation, useSearchParams } from 'react-router-dom'
import { useSWRConfig } from 'swr'
import { AlertTriangle, CheckCircle2, Download, ExternalLink, FileDown, FileUp, Loader2, PackageCheck, Plus, SearchCheck, Trash2, Upload } from 'lucide-react'

import { DeviceScopeFilters } from '@/components/context/DeviceScopeFilters'
import { ItemInfoPopover } from '@/components/ItemInfoPopover'
import { NewItemDialog } from '@/components/NewItemDialog'
import { Badge } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card'
import { Dialog, DialogContent, DialogDescription, DialogFooter, DialogHeader, DialogTitle } from '@/components/ui/dialog'
import { ItemCombobox, type ItemComboboxOption } from '@/components/ui/item-combobox'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from '@/components/ui/select'
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from '@/components/ui/table'
import { useAuthSession } from '@/hooks/useAuthSession'
import { useDeviceScopes } from '@/hooks/useDeviceScopes'
import { useInventoryOverview } from '@/hooks/useInventoryOverview'
import { useRequirements } from '@/hooks/useRequirements'
import { resolveActorId } from '@/lib/auth'
import { applyRequirementsImport, confirmBulkReservation, exportRequirementsCSV, fetchBulkReservationPreview, previewRequirementsImport } from '@/lib/additionalApi'
import { downloadTextFile } from '@/lib/csv'
import { batchUpsertRequirements, upsertRequirement } from '@/lib/mockApi'
import type { BulkReservationPreviewResponse, MasterItemRecord, RequirementSummary, RequirementsImportPreviewResponse } from '@/types'

const REQUIREMENTS_TEMPLATE =
  'device,scope,manufacturer,item_number,description,quantity,note\nER2,powerboard,Omron,ER2,Control relay,10,Initial build demand\n'

type RequirementFormRow = {
  key: string
  itemId: string
  quantity: number
  note: string
}

let rowKeyCounter = 0
function nextRowKey() {
  return `row-${++rowKeyCounter}`
}

function createEmptyRow(): RequirementFormRow {
  return { key: nextRowKey(), itemId: '', quantity: 1, note: '' }
}

function csvEscape(value: string | number) {
  const text = String(value)
  return /[",\r\n]/.test(text) ? `"${text.replace(/"/g, '""')}"` : text
}

function buildMissingItemsCSV(preview: RequirementsImportPreviewResponse) {
  const itemNumbers = Array.from(
    new Set(
      preview.rows
        .filter((row) => !row.itemRegistered && row.itemNumber.trim())
        .map((row) => row.itemNumber.trim()),
    ),
  )
  return [
    'canonical_item_number,description,manufacturer,category',
    ...itemNumbers.map((itemNumber) => [itemNumber, '', '', ''].map(csvEscape).join(',')),
  ].join('\n')
}

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
  const location = useLocation()
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
  const [formRows, setFormRows] = useState<RequirementFormRow[]>([createEmptyRow()])
  const [duplicateDialogOpen, setDuplicateDialogOpen] = useState(false)
  const [duplicateItems, setDuplicateItems] = useState<string[]>([])
  const [createdItemOptions, setCreatedItemOptions] = useState<ItemComboboxOption[]>([])
  const [newItemDialogOpen, setNewItemDialogOpen] = useState(false)
  const [newItemInitialNumber, setNewItemInitialNumber] = useState('')
  const [newItemTargetRowKey, setNewItemTargetRowKey] = useState<string | null>(null)
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
    const byId = new Map<string, ItemComboboxOption>()
    for (const row of inventory?.balances ?? []) {
      byId.set(row.itemId, {
        itemId: row.itemId,
        itemNumber: row.itemNumber,
        description: row.description,
        manufacturer: row.manufacturer,
        category: row.category,
      })
    }
    for (const row of createdItemOptions) {
      byId.set(row.itemId, row)
    }
    return Array.from(byId.values()).sort((left, right) => left.itemNumber.localeCompare(right.itemNumber))
  }, [createdItemOptions, inventory?.balances])
  const activeScopes = scopes.filter((row) => row.status !== 'inactive')
  const selectedScope = activeScopes.find((row) => row.id === formScopeId)
  const actorId = resolveActorId(session)
  const returnTo = `${location.pathname}${location.search}`
  const itemsImportHref = `/app/operator/items/import?returnTo=${encodeURIComponent(returnTo)}`

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
    setFormRows([{ key: nextRowKey(), itemId: row.itemId, quantity: row.quantity, note: row.note }])
  }

  function resetForm() {
    setSelectedRequirement(null)
    setFormScopeId('')
    setFormRows([createEmptyRow()])
  }

  function updateFormRow(key: string, field: keyof RequirementFormRow, value: string | number) {
    setFormRows((prev) => prev.map((row) => (row.key === key ? { ...row, [field]: value } : row)))
  }

  function removeFormRow(key: string) {
    setFormRows((prev) => {
      const next = prev.filter((row) => row.key !== key)
      return next.length === 0 ? [createEmptyRow()] : next
    })
  }

  function addFormRow() {
    setFormRows((prev) => [...prev, createEmptyRow()])
  }

  function openNewItemDialog(rowKey: string, query: string) {
    setNewItemTargetRowKey(rowKey)
    setNewItemInitialNumber(query)
    setNewItemDialogOpen(true)
  }

  async function handleNewItemCreated(item: MasterItemRecord) {
    const option: ItemComboboxOption = {
      itemId: item.id,
      itemNumber: item.itemNumber,
      description: item.description,
      manufacturer: item.manufacturerKey,
      category: item.categoryKey,
    }
    setCreatedItemOptions((current) => {
      const next = current.filter((candidate) => candidate.itemId !== option.itemId)
      return [...next, option]
    })
    if (newItemTargetRowKey) {
      updateFormRow(newItemTargetRowKey, 'itemId', item.id)
    }
    await Promise.all([mutate('inventory-overview'), mutate('master-data')])
    setFeedback({ tone: 'success', text: `${item.itemNumber} を登録し、Requirement フォームに選択しました。` })
  }

  function handleDownloadMissingItemsCSV() {
    if (!importPreview) return
    downloadTextFile(`missing-items-${importPreview.fileName || 'requirements'}.csv`, buildMissingItemsCSV(importPreview))
  }

  // Check for duplicates against existing requirements
  function findDuplicateItems(rows: RequirementFormRow[]): string[] {
    const existingReqs = requirements?.rows ?? []
    const scopeRecord = activeScopes.find((s) => s.id === formScopeId)
    if (!scopeRecord) return []
    const duplicates: string[] = []
    for (const row of rows) {
      if (!row.itemId) continue
      const existing = existingReqs.find(
        (req) => req.device === scopeRecord.deviceKey && req.scope === (scopeRecord.scopeName || scopeRecord.scopeKey) && req.itemId === row.itemId
      )
      if (existing) {
        const item = items.find((i) => i.itemId === row.itemId)
        duplicates.push(item ? `${item.itemNumber} / ${item.description}` : row.itemId)
      }
    }
    return duplicates
  }

  async function handleRequirementSubmit(event: FormEvent<HTMLFormElement>) {
    event.preventDefault()

    // Single-edit mode (editing an existing requirement)
    if (selectedRequirement) {
      setIsSaving(true)
      setFeedback(null)
      try {
        await upsertRequirement({
          id: selectedRequirement.id,
          deviceScopeId: formScopeId,
          itemId: formRows[0]?.itemId ?? '',
          quantity: formRows[0]?.quantity ?? 1,
          note: formRows[0]?.note ?? '',
        })
        await Promise.all([mutate(['requirements', device, scope]), mutate(['scope-overview', device])])
        setFeedback({ tone: 'success', text: 'Requirement updated.' })
        resetForm()
      } catch (caught) {
        setFeedback({ tone: 'error', text: caught instanceof Error ? caught.message : 'Failed to save requirement.' })
      } finally {
        setIsSaving(false)
      }
      return
    }

    // Batch mode - filter valid rows
    const validRows = formRows.filter((row) => row.itemId && row.quantity > 0)
    if (validRows.length === 0) return

    // Check for duplicates
    const dupes = findDuplicateItems(validRows)
    if (dupes.length > 0 && !duplicateDialogOpen) {
      setDuplicateItems(dupes)
      setDuplicateDialogOpen(true)
      return
    }

    await submitBatch(validRows)
  }

  async function submitBatch(validRows: RequirementFormRow[]) {
    setDuplicateDialogOpen(false)
    setIsSaving(true)
    setFeedback(null)
    try {
      const result = await batchUpsertRequirements({
        deviceScopeId: formScopeId,
        rows: validRows.map((row) => ({ itemId: row.itemId, quantity: row.quantity, note: row.note })),
      })
      await Promise.all([mutate(['requirements', device, scope]), mutate(['scope-overview', device])])
      const parts: string[] = []
      if (result.created > 0) parts.push(`${result.created} created`)
      if (result.updated > 0) parts.push(`${result.updated} updated`)
      if (result.errored > 0) parts.push(`${result.errored} errored`)
      setFeedback({ tone: result.errored > 0 ? 'error' : 'success', text: `Batch save: ${parts.join(', ')}.` })
      resetForm()
    } catch (caught) {
      setFeedback({ tone: 'error', text: caught instanceof Error ? caught.message : 'Failed to save requirements.' })
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
      const summary = {
        invalid: result.rows.filter((row) => row.status !== 'valid').length,
      }
      setImportPreview(result)
      setFeedback({
        tone: summary.invalid > 0 ? 'error' : 'success',
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

      <div className="grid gap-6 xl:grid-cols-[minmax(0,1fr)_480px]">
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
              <CardTitle>{selectedRequirement ? 'Edit Requirement' : 'New Requirements'}</CardTitle>
              <CardDescription>{selectedRequirement ? 'Update an existing requirement.' : 'Add item demand for a scope. You can add multiple items at once.'}</CardDescription>
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
                  <div className="flex items-center justify-between">
                    <Label>Items</Label>
                    {!selectedRequirement && (
                      <Button type="button" variant="ghost" size="sm" className="h-7 gap-1 text-xs" onClick={addFormRow}>
                        <Plus className="h-3.5 w-3.5" />
                        Add Row
                      </Button>
                    )}
                  </div>
                  <div className="space-y-2">
                    {formRows.map((formRow) => {
                      const isItemUsedElsewhere = formRows.some((other) => other.key !== formRow.key && other.itemId !== '' && other.itemId === formRow.itemId)
                      return (
                        <div key={formRow.key} className="flex items-start gap-2">
                          <div className="min-w-0 flex-1">
                            <ItemCombobox
                              items={items}
                              value={formRow.itemId}
                              onValueChange={(value) => updateFormRow(formRow.key, 'itemId', value)}
                              onCreateNew={(query) => openNewItemDialog(formRow.key, query)}
                              placeholder="Search or select item"
                              triggerClassName={`h-9 text-xs ${isItemUsedElsewhere ? 'border-yellow-400' : ''}`}
                            />
                          </div>
                          <Input
                            type="number"
                            min={1}
                            className="h-9 w-16 text-xs"
                            placeholder="Qty"
                            value={formRow.quantity}
                            onChange={(e) => updateFormRow(formRow.key, 'quantity', Math.max(1, Number(e.target.value) || 1))}
                          />
                          <Input
                            className="h-9 w-24 text-xs"
                            placeholder="Note"
                            value={formRow.note}
                            onChange={(e) => updateFormRow(formRow.key, 'note', e.target.value)}
                          />
                          {!selectedRequirement && (
                            <Button
                              type="button"
                              variant="ghost"
                              size="sm"
                              className="h-9 w-9 shrink-0 p-0 text-muted-foreground hover:text-destructive"
                              onClick={() => removeFormRow(formRow.key)}
                            >
                              <Trash2 className="h-3.5 w-3.5" />
                            </Button>
                          )}
                        </div>
                      )
                    })}
                  </div>
                  {formRows.some((r) => formRows.filter((o) => o.itemId === r.itemId && r.itemId !== '').length > 1) && (
                    <p className="flex items-center gap-1 text-xs text-yellow-600">
                      <AlertTriangle className="h-3.5 w-3.5" />
                      Duplicate items detected within this form.
                    </p>
                  )}
                </div>

                <div className="flex gap-2">
                  <Button
                    type="submit"
                    className="flex-1"
                    disabled={!formScopeId || formRows.every((r) => !r.itemId) || isSaving}
                  >
                    {isSaving ? 'Saving...' : selectedRequirement ? 'Update' : `Save All (${formRows.filter((r) => r.itemId).length})`}
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
                  {importSummary.missingItems > 0 && (
                    <div className="flex flex-wrap items-center gap-2 rounded-md border border-amber-200 bg-amber-50 p-3">
                      <Button type="button" variant="outline" size="sm" className="gap-2 bg-white" onClick={handleDownloadMissingItemsCSV}>
                        <FileDown className="h-4 w-4" />
                        不足アイテム CSV をダウンロード
                      </Button>
                      <Button asChild type="button" variant="link" size="sm" className="gap-1 px-0 text-amber-900">
                        <Link to={itemsImportHref}>
                          Items Import ページで登録
                          <ExternalLink className="h-3.5 w-3.5" />
                        </Link>
                      </Button>
                      <p className="basis-full text-xs text-amber-900">
                        CSV は canonical_item_number と空の description, manufacturer, category を含みます。登録後はこのページへ戻り、同じ Requirements CSV を再プレビューしてください。
                      </p>
                    </div>
                  )}
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

      <Dialog open={duplicateDialogOpen} onOpenChange={setDuplicateDialogOpen}>
        <DialogContent>
          <DialogHeader>
            <DialogTitle className="flex items-center gap-2">
              <AlertTriangle className="h-5 w-5 text-yellow-500" />
              Duplicate Requirements Detected
            </DialogTitle>
            <DialogDescription>
              The following items already have requirements for this scope. Saving will update their quantity and note.
            </DialogDescription>
          </DialogHeader>
          <div className="max-h-48 space-y-1 overflow-auto text-sm">
            {duplicateItems.map((name) => (
              <div key={name} className="rounded border px-3 py-2">{name}</div>
            ))}
          </div>
          <DialogFooter>
            <Button variant="outline" onClick={() => setDuplicateDialogOpen(false)}>Cancel</Button>
            <Button onClick={() => void submitBatch(formRows.filter((r) => r.itemId && r.quantity > 0))}>
              Save Anyway
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>
      <NewItemDialog
        open={newItemDialogOpen}
        initialItemNumber={newItemInitialNumber}
        onOpenChange={setNewItemDialogOpen}
        onCreated={(item) => void handleNewItemCreated(item)}
      />
    </div>
  )
}
