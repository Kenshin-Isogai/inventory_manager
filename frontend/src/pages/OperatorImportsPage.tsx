import type { ChangeEvent } from 'react'
import { useMemo, useRef, useState } from 'react'
import { useSWRConfig } from 'swr'

import { Badge } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card'
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from '@/components/ui/dialog'
import { Label } from '@/components/ui/label'
import { Sheet, SheetContent, SheetDescription, SheetHeader, SheetTitle } from '@/components/ui/sheet'
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from '@/components/ui/table'
import { Tabs, TabsList, TabsTrigger } from '@/components/ui/tabs'
import { useAuthSession } from '@/hooks/useAuthSession'
import { useImports } from '@/hooks/useImports'
import { downloadTextFile } from '@/lib/csv'
import { fetchImportDetail, importMasterDataCSV, previewMasterDataCSV, undoImport } from '@/lib/mockApi'
import type { ImportDetailResponse, ImportPreviewResult, ImportPreviewRow, ImportType } from '@/types'
import {
  CheckCircle2,
  Eye,
  FileDown,
  FileUp,
  History,
  Loader2,
  RotateCcw,
  SearchCheck,
  TriangleAlert,
  Upload,
} from 'lucide-react'

type OperatorImportsPageProps = {
  mode?: 'upload' | 'history'
}

const importTypeMeta: Record<ImportType, { label: string; description: string; headers: string[]; optionalHeaders: string[] }> = {
  items_with_aliases: {
    label: 'Items + Aliases CSV',
    description:
      'Register canonical items and supplier/order aliases in one CSV. Alias-only rows can update existing canonical items.',
    headers: ['canonical_item_number'],
    optionalHeaders: [
      'description',
      'manufacturer',
      'category',
      'default_supplier_id',
      'supplier_id',
      'supplier_item_number',
      'units_per_order',
      'note',
    ],
  },
}

const importTemplates: Record<ImportType, string> = {
  items_with_aliases: [
    'canonical_item_number,description,manufacturer,category,default_supplier_id,supplier_id,supplier_item_number,units_per_order,note',
    'ER2,Control relay,Omron,Relay,sup-misumi,sup-misumi,ER2-P4,4,New item with pack alias',
    'ER2,,,,,sup-misumi,ER2-P8,8,Alias-only row for existing canonical item',
  ].join('\n'),
}

function getStatusBadgeVariant(status: string) {
  switch (status.toLowerCase()) {
    case 'completed':
    case 'valid':
    case 'ready':
      return 'default'
    case 'pending':
    case 'staged':
    case 'applied':
      return 'secondary'
    case 'failed':
    case 'invalid':
    case 'has_errors':
    case 'rejected':
      return 'destructive'
    default:
      return 'outline'
  }
}

function formatDateTime(value: string) {
  if (!value) {
    return '—'
  }
  const parsed = new Date(value)
  return Number.isNaN(parsed.getTime()) ? value : parsed.toLocaleString()
}

function prettifyKey(value: string) {
  return value.replace(/_/g, ' ')
}

function parseSummary(summary: string) {
  try {
    const parsed = JSON.parse(summary) as Record<string, unknown>
    if (parsed && typeof parsed === 'object' && !Array.isArray(parsed)) {
      return Object.entries(parsed).map(([key, value]) => ({
        key,
        label: prettifyKey(key),
        value: String(value),
      }))
    }
  } catch {
    // Fall through to raw summary.
  }

  return [{ key: 'summary', label: 'summary', value: summary || '—' }]
}

function summarizeRows(rows: ImportPreviewRow[]) {
  return rows.reduce(
    (summary, row) => {
      summary.total += 1
      if (row.status === 'invalid') {
        summary.invalid += 1
      } else {
        summary.valid += 1
      }
      return summary
    },
    { total: 0, valid: 0, invalid: 0 },
  )
}

function deriveIssue(row: ImportPreviewRow, importType: ImportType) {
  const matchedColumn = row.message.match(/^([a-z0-9_]+) is required$/i)?.[1]
  const fallbackColumn = importTypeMeta[importType].headers.find((header) => (row.raw[header] ?? '').trim() === '')
  const column = matchedColumn ?? fallbackColumn ?? ''
  const currentValue = column ? row.raw[column] ?? '' : ''
  const fix = column
    ? `${prettifyKey(column)} を補完してから再度プレビューしてください。`
    : '行データを修正してから再度プレビューしてください。'

  return { column, currentValue, fix }
}

function RowResultsTable({
  rows,
  importType,
}: {
  rows: ImportPreviewRow[]
  importType: ImportType
}) {
  return (
    <div className="overflow-x-auto">
      <Table>
        <TableHeader>
          <TableRow>
            <TableHead className="w-20">Row</TableHead>
            <TableHead className="w-28">Status</TableHead>
            <TableHead className="w-44">Code</TableHead>
            <TableHead>Message</TableHead>
            <TableHead>Values</TableHead>
          </TableRow>
        </TableHeader>
        <TableBody>
          {rows.map((row) => (
            <TableRow key={`${row.rowNumber}-${row.code}`}>
              <TableCell className="text-sm font-medium">{row.rowNumber}</TableCell>
              <TableCell>
                <Badge variant={getStatusBadgeVariant(row.status)} className="text-xs">
                  {row.status}
                </Badge>
              </TableCell>
              <TableCell className="text-xs text-muted-foreground">{row.code || '—'}</TableCell>
              <TableCell className="text-sm">{row.message || '—'}</TableCell>
              <TableCell className="text-xs text-muted-foreground">
                {Object.keys(row.raw).length === 0
                  ? '—'
                  : Object.entries(row.raw)
                      .map(([key, value]) => `${prettifyKey(key)}=${value || '∅'}`)
                      .join(' / ')}
              </TableCell>
            </TableRow>
          ))}
          {rows.length === 0 && (
            <TableRow>
              <TableCell colSpan={5} className="py-6 text-center text-sm text-muted-foreground">
                No rows to display for {importTypeMeta[importType].label}.
              </TableCell>
            </TableRow>
          )}
        </TableBody>
      </Table>
    </div>
  )
}

export function OperatorImportsPage({ mode = 'upload' }: OperatorImportsPageProps) {
  const { data } = useImports()
  const { data: session } = useAuthSession()
  const { mutate } = useSWRConfig()
  const fileInputRef = useRef<HTMLInputElement>(null)

  const [importType, setImportType] = useState<ImportType>('items_with_aliases')
  const [selectedFile, setSelectedFile] = useState<File | null>(null)
  const [previewResult, setPreviewResult] = useState<ImportPreviewResult | null>(null)
  const [previewFilter, setPreviewFilter] = useState<'all' | 'errors' | 'valid'>('all')
  const [latestAppliedDetail, setLatestAppliedDetail] = useState<ImportDetailResponse | null>(null)
  const [detailCache, setDetailCache] = useState<Record<string, ImportDetailResponse>>({})
  const [selectedDetailId, setSelectedDetailId] = useState<string | null>(null)
  const [detailSheetOpen, setDetailSheetOpen] = useState(false)
  const [undoDialogOpen, setUndoDialogOpen] = useState(false)
  const [undoTargetId, setUndoTargetId] = useState<string | null>(null)
  const [feedback, setFeedback] = useState<{ tone: 'success' | 'error'; text: string } | null>(null)
  const [isPreviewing, setIsPreviewing] = useState(false)
  const [isApplying, setIsApplying] = useState(false)
  const [loadingDetailId, setLoadingDetailId] = useState<string | null>(null)
  const [undoingId, setUndoingId] = useState<string | null>(null)

  const historyRows = data?.rows ?? []
  const selectedDetail = selectedDetailId ? detailCache[selectedDetailId] ?? null : null
  const undoTarget = undoTargetId ? detailCache[undoTargetId] ?? null : null
  const actorId = session?.user?.userId?.trim() || session?.user?.email?.trim() || 'operator-imports-page'

  const previewSummary = useMemo(() => summarizeRows(previewResult?.rows ?? []), [previewResult])
  const filteredPreviewRows = useMemo(() => {
    const rows = previewResult?.rows ?? []
    switch (previewFilter) {
      case 'errors':
        return rows.filter((row) => row.status === 'invalid')
      case 'valid':
        return rows.filter((row) => row.status !== 'invalid')
      default:
        return rows
    }
  }, [previewFilter, previewResult])
  const invalidPreviewRows = useMemo(
    () => (previewResult?.rows ?? []).filter((row) => row.status === 'invalid'),
    [previewResult],
  )

  function resetFileInput() {
    if (fileInputRef.current) {
      fileInputRef.current.value = ''
    }
  }

  function resetUploadState(clearFile = false) {
    setPreviewResult(null)
    setPreviewFilter('all')
    setLatestAppliedDetail(null)
    if (clearFile) {
      setSelectedFile(null)
      resetFileInput()
    }
  }

  function handleImportTypeChange(nextImportType: ImportType) {
    setImportType(nextImportType)
    setFeedback(null)
    resetUploadState(true)
  }

  function handleFileSelected(event: ChangeEvent<HTMLInputElement>) {
    const file = event.target.files?.[0]
    if (!file) {
      return
    }

    setSelectedFile(file)
    setFeedback(null)
    setLatestAppliedDetail(null)
    setPreviewResult(null)
    setPreviewFilter('all')
    event.target.value = ''
  }

  async function handlePreviewRequest() {
    if (!selectedFile) {
      setFeedback({ tone: 'error', text: 'PreviewするCSVを先に選択してください。' })
      return
    }

    setIsPreviewing(true)
    setFeedback(null)
    setLatestAppliedDetail(null)
    try {
      const result = await previewMasterDataCSV(importType, selectedFile)
      setPreviewResult(result)
      setFeedback({
        tone: result.status === 'has_errors' ? 'error' : 'success',
        text:
          result.status === 'has_errors'
            ? `${result.fileName} の検証で ${summarizeRows(result.rows).invalid} 件の修正点が見つかりました。`
            : `${result.fileName} のプレビューを作成しました。適用前に内容を確認してください。`,
      })
    } catch (caught) {
      setPreviewResult(null)
      setFeedback({
        tone: 'error',
        text: caught instanceof Error ? caught.message : 'Preview generation failed.',
      })
    } finally {
      setIsPreviewing(false)
    }
  }

  async function handleApplyRequest() {
    if (!selectedFile || !previewResult) {
      setFeedback({ tone: 'error', text: 'Applyする前にプレビューを作成してください。' })
      return
    }
    if (previewSummary.invalid > 0) {
      setFeedback({ tone: 'error', text: 'Validation error が残っているため、このCSVは適用できません。' })
      return
    }

    setIsApplying(true)
    setFeedback(null)
    try {
      const job = await importMasterDataCSV(importType, selectedFile)
      await Promise.all([mutate('imports'), mutate('master-data')])
      const detail = await fetchImportDetail(job.id)
      setDetailCache((current) => ({ ...current, [detail.id]: detail }))
      setLatestAppliedDetail(detail)
      setPreviewResult(null)
      setPreviewFilter('all')
      setSelectedFile(null)
      resetFileInput()
      setFeedback({ tone: 'success', text: `${detail.fileName} を適用しました。履歴と結果詳細を確認できます。` })
    } catch (caught) {
      setFeedback({
        tone: 'error',
        text: caught instanceof Error ? caught.message : 'Import apply failed.',
      })
    } finally {
      setIsApplying(false)
    }
  }

  async function handleOpenDetail(importId: string) {
    setDetailSheetOpen(true)
    setSelectedDetailId(importId)
    setFeedback(null)

    if (detailCache[importId]) {
      return
    }

    setLoadingDetailId(importId)
    try {
      const detail = await fetchImportDetail(importId)
      setDetailCache((current) => ({ ...current, [importId]: detail }))
    } catch (caught) {
      setFeedback({
        tone: 'error',
        text: caught instanceof Error ? caught.message : 'Failed to load import detail.',
      })
      setDetailSheetOpen(false)
      setSelectedDetailId(null)
    } finally {
      setLoadingDetailId(null)
    }
  }

  async function handleConfirmUndo() {
    if (!undoTargetId) {
      return
    }

    setUndoingId(undoTargetId)
    setFeedback(null)
    try {
      const detail = await undoImport(undoTargetId, actorId)
      setDetailCache((current) => ({ ...current, [detail.id]: detail }))
      setLatestAppliedDetail((current) => (current?.id === detail.id ? detail : current))
      await mutate('imports')
      setUndoDialogOpen(false)
      setFeedback({ tone: 'success', text: `${detail.fileName} の取込を取り消しました。` })
    } catch (caught) {
      setFeedback({
        tone: 'error',
        text: caught instanceof Error ? caught.message : 'Failed to undo import.',
      })
    } finally {
      setUndoingId(null)
    }
  }

  return (
    <div className="space-y-6 p-6">
      <div className="space-y-2">
        <h1 className="text-3xl font-bold tracking-tight">Imports</h1>
        <p className="text-muted-foreground">
          Preview, validate, apply, and audit CSV imports for item and alias master data.
        </p>
      </div>

      {feedback && (
        <div
          className={`rounded-lg border px-4 py-3 text-sm ${
            feedback.tone === 'success'
              ? 'border-green-200 bg-green-50 text-green-800'
              : 'border-red-200 bg-red-50 text-red-800'
          }`}
        >
          {feedback.text}
        </div>
      )}

      {mode === 'upload' && (
        <>
          <Card className="border-dashed border-2 border-primary/30">
            <CardHeader>
              <CardTitle className="flex items-center gap-2">
                <Upload className="h-5 w-5" />
                Upload CSV
              </CardTitle>
              <CardDescription>
                Choose the master-data import type, preview the file, and apply only when validation passes.
              </CardDescription>
            </CardHeader>
            <CardContent className="space-y-6">
              <div className="space-y-3">
                <Label>Import format</Label>
                <Tabs value={importType} onValueChange={(value) => handleImportTypeChange(value as ImportType)}>
                  <TabsList className="grid w-full grid-cols-1">
                    <TabsTrigger value="items_with_aliases">Items + Aliases CSV</TabsTrigger>
                  </TabsList>
                </Tabs>
                <div className="rounded-lg border bg-muted/30 p-4 text-sm">
                  <p className="font-medium">{importTypeMeta.items_with_aliases.description}</p>
                  <p className="mt-2 text-muted-foreground">
                    Required column: {importTypeMeta.items_with_aliases.headers.join(', ')}
                  </p>
                  <p className="mt-1 text-muted-foreground">
                    Optional columns: {importTypeMeta.items_with_aliases.optionalHeaders.join(', ')}
                  </p>
                </div>
              </div>

              <div className="space-y-3">
                <Label htmlFor="csv-upload">CSV file</Label>
                <input
                  ref={fileInputRef}
                  id="csv-upload"
                  type="file"
                  accept=".csv,text/csv"
                  onChange={handleFileSelected}
                  className="hidden"
                />
                <div className="flex flex-wrap items-center gap-3">
                  <Button variant="outline" className="gap-2" onClick={() => fileInputRef.current?.click()}>
                    <FileUp className="h-4 w-4" />
                    Choose File
                  </Button>
                  <Button
                    className="gap-2"
                    onClick={() => void handlePreviewRequest()}
                    disabled={!selectedFile || isPreviewing}
                  >
                    {isPreviewing ? <Loader2 className="h-4 w-4 animate-spin" /> : <SearchCheck className="h-4 w-4" />}
                    {isPreviewing ? 'Generating Preview...' : 'Run Preview'}
                  </Button>
                  <Button
                    variant="ghost"
                    onClick={() => {
                      setFeedback(null)
                      resetUploadState(true)
                    }}
                    disabled={!selectedFile && !previewResult && !latestAppliedDetail}
                  >
                    Clear
                  </Button>
                  <Button
                    variant="outline"
                    className="gap-2"
                    onClick={() => downloadTextFile(`${importType}-template.csv`, importTemplates[importType])}
                  >
                    <FileDown className="h-4 w-4" />
                    Template
                  </Button>
                </div>
                <p className="text-sm text-muted-foreground">
                  {selectedFile
                    ? `Selected file: ${selectedFile.name}`
                    : `Use ${importTypeMeta[importType].label} with columns aligned to the specification.`}
                </p>
              </div>
            </CardContent>
          </Card>

          {previewResult && (
            <>
              <Card>
                <CardHeader>
                  <CardTitle className="flex items-center gap-2">
                    <SearchCheck className="h-5 w-5" />
                    Validation Result
                  </CardTitle>
                  <CardDescription>
                    Preview rows before applying {previewResult.fileName}. Fix any invalid rows first.
                  </CardDescription>
                </CardHeader>
                <CardContent className="space-y-4">
                  <div className="grid gap-4 md:grid-cols-4">
                    <div className="rounded-lg border p-4">
                      <p className="text-xs uppercase tracking-wide text-muted-foreground">Status</p>
                      <div className="mt-2">
                        <Badge variant={getStatusBadgeVariant(previewResult.status)}>{previewResult.status}</Badge>
                      </div>
                    </div>
                    <div className="rounded-lg border p-4">
                      <p className="text-xs uppercase tracking-wide text-muted-foreground">Rows</p>
                      <p className="mt-2 text-2xl font-semibold">{previewSummary.total}</p>
                    </div>
                    <div className="rounded-lg border p-4">
                      <p className="text-xs uppercase tracking-wide text-muted-foreground">Valid</p>
                      <p className="mt-2 text-2xl font-semibold text-green-700">{previewSummary.valid}</p>
                    </div>
                    <div className="rounded-lg border p-4">
                      <p className="text-xs uppercase tracking-wide text-muted-foreground">Invalid</p>
                      <p className="mt-2 text-2xl font-semibold text-red-700">{previewSummary.invalid}</p>
                    </div>
                  </div>

                  {invalidPreviewRows.length > 0 && (
                    <div className="space-y-3 rounded-lg border border-red-200 bg-red-50 p-4">
                      <div className="flex items-center gap-2 text-red-800">
                        <TriangleAlert className="h-4 w-4" />
                        <p className="text-sm font-medium">Rows requiring correction</p>
                      </div>
                      <div className="overflow-x-auto">
                        <Table>
                          <TableHeader>
                            <TableRow>
                              <TableHead className="w-20">Row</TableHead>
                              <TableHead className="w-44">Column</TableHead>
                              <TableHead>Current value</TableHead>
                              <TableHead>Validation message</TableHead>
                              <TableHead>Fix guidance</TableHead>
                            </TableRow>
                          </TableHeader>
                          <TableBody>
                            {invalidPreviewRows.map((row) => {
                              const issue = deriveIssue(row, previewResult.importType)
                              return (
                                <TableRow key={`invalid-${row.rowNumber}`}>
                                  <TableCell className="font-medium">{row.rowNumber}</TableCell>
                                  <TableCell>{issue.column ? prettifyKey(issue.column) : '—'}</TableCell>
                                  <TableCell>{issue.currentValue || '—'}</TableCell>
                                  <TableCell>{row.message || '—'}</TableCell>
                                  <TableCell>{issue.fix}</TableCell>
                                </TableRow>
                              )
                            })}
                          </TableBody>
                        </Table>
                      </div>
                    </div>
                  )}

                  <div className="flex flex-wrap items-center gap-3">
                    <Button
                      onClick={() => void handleApplyRequest()}
                      disabled={isApplying || previewSummary.invalid > 0 || previewSummary.total === 0}
                      className="gap-2"
                    >
                      {isApplying ? <Loader2 className="h-4 w-4 animate-spin" /> : <CheckCircle2 className="h-4 w-4" />}
                      {isApplying ? 'Applying...' : 'Apply Import'}
                    </Button>
                    <p className="text-sm text-muted-foreground">
                      Apply is enabled only when every required row is valid.
                    </p>
                  </div>
                </CardContent>
              </Card>

              <Card>
                <CardHeader>
                  <CardTitle>Preview Rows</CardTitle>
                  <CardDescription>Inspect the staged row data before applying the import.</CardDescription>
                </CardHeader>
                <CardContent className="space-y-4">
                  <Tabs value={previewFilter} onValueChange={(value) => setPreviewFilter(value as 'all' | 'errors' | 'valid')}>
                    <TabsList>
                      <TabsTrigger value="all">All Rows</TabsTrigger>
                      <TabsTrigger value="errors">Errors</TabsTrigger>
                      <TabsTrigger value="valid">Valid</TabsTrigger>
                    </TabsList>
                  </Tabs>
                  <RowResultsTable rows={filteredPreviewRows} importType={previewResult.importType} />
                </CardContent>
              </Card>
            </>
          )}

          {latestAppliedDetail && (
            <Card>
              <CardHeader>
                <CardTitle className="flex items-center gap-2">
                  <CheckCircle2 className="h-5 w-5" />
                  Apply Result
                </CardTitle>
                <CardDescription>
                  Review the persisted result for {latestAppliedDetail.fileName} and continue from the import history if needed.
                </CardDescription>
              </CardHeader>
              <CardContent className="space-y-4">
                <div className="grid gap-4 md:grid-cols-3">
                  <div className="rounded-lg border p-4">
                    <p className="text-xs uppercase tracking-wide text-muted-foreground">Job ID</p>
                    <p className="mt-2 text-sm font-medium">{latestAppliedDetail.id}</p>
                  </div>
                  <div className="rounded-lg border p-4">
                    <p className="text-xs uppercase tracking-wide text-muted-foreground">Status</p>
                    <div className="mt-2 flex gap-2">
                      <Badge variant={getStatusBadgeVariant(latestAppliedDetail.status)}>{latestAppliedDetail.status}</Badge>
                      <Badge variant={getStatusBadgeVariant(latestAppliedDetail.lifecycleState)}>
                        {latestAppliedDetail.lifecycleState}
                      </Badge>
                    </div>
                  </div>
                  <div className="rounded-lg border p-4">
                    <p className="text-xs uppercase tracking-wide text-muted-foreground">Created</p>
                    <p className="mt-2 text-sm font-medium">{formatDateTime(latestAppliedDetail.createdAt)}</p>
                  </div>
                </div>
                <div className="grid gap-3 md:grid-cols-3">
                  {parseSummary(latestAppliedDetail.summary).map((entry) => (
                    <div key={entry.key} className="rounded-lg border bg-muted/30 p-4">
                      <p className="text-xs uppercase tracking-wide text-muted-foreground">{entry.label}</p>
                      <p className="mt-2 text-lg font-semibold">{entry.value}</p>
                    </div>
                  ))}
                </div>
                <Button variant="outline" className="gap-2" onClick={() => void handleOpenDetail(latestAppliedDetail.id)}>
                  <Eye className="h-4 w-4" />
                  Open Result Detail
                </Button>
              </CardContent>
            </Card>
          )}
        </>
      )}

      <Card>
        <CardHeader>
          <CardTitle className="flex items-center gap-2">
            <History className="h-5 w-5" />
            Import History
          </CardTitle>
          <CardDescription>Review recent CSV imports and drill into row-level details.</CardDescription>
        </CardHeader>
        <CardContent>
          <div className="overflow-x-auto">
            <Table>
              <TableHeader>
                <TableRow>
                  <TableHead className="w-20">Type</TableHead>
                  <TableHead className="w-24">Status</TableHead>
                  <TableHead>File</TableHead>
                  <TableHead className="max-w-sm">Summary</TableHead>
                  <TableHead className="w-44">Created</TableHead>
                  <TableHead className="w-32 text-right">Actions</TableHead>
                </TableRow>
              </TableHeader>
              <TableBody>
                {historyRows.map((row) => {
                  const cachedDetail = detailCache[row.id]
                  return (
                    <TableRow key={row.id}>
                      <TableCell className="text-sm font-medium">{row.importType}</TableCell>
                      <TableCell>
                        <div className="flex flex-wrap gap-2">
                          <Badge variant={getStatusBadgeVariant(row.status)} className="text-xs">
                            {row.status}
                          </Badge>
                          {cachedDetail?.lifecycleState && cachedDetail.lifecycleState !== 'applied' && (
                            <Badge variant={getStatusBadgeVariant(cachedDetail.lifecycleState)} className="text-xs">
                              {cachedDetail.lifecycleState}
                            </Badge>
                          )}
                        </div>
                      </TableCell>
                      <TableCell className="text-sm">{row.fileName}</TableCell>
                      <TableCell className="text-sm text-muted-foreground">
                        {parseSummary(row.summary)
                          .map((entry) => `${entry.label}: ${entry.value}`)
                          .join(' / ')}
                      </TableCell>
                      <TableCell className="text-sm text-muted-foreground">{formatDateTime(row.createdAt)}</TableCell>
                      <TableCell className="text-right">
                        <Button variant="outline" size="sm" onClick={() => void handleOpenDetail(row.id)} className="gap-2">
                          <Eye className="h-4 w-4" />
                          Details
                        </Button>
                      </TableCell>
                    </TableRow>
                  )
                })}
                {historyRows.length === 0 && (
                  <TableRow>
                    <TableCell colSpan={6} className="py-8 text-center text-muted-foreground">
                      No import history yet.
                    </TableCell>
                  </TableRow>
                )}
              </TableBody>
            </Table>
          </div>
        </CardContent>
      </Card>

      <Sheet
        open={detailSheetOpen}
        onOpenChange={(open) => {
          setDetailSheetOpen(open)
          if (!open) {
            setSelectedDetailId(null)
          }
        }}
      >
        <SheetContent side="right" className="w-full overflow-y-auto sm:max-w-2xl">
          <SheetHeader>
            <SheetTitle>Import Detail</SheetTitle>
            <SheetDescription>
              Inspect row outcomes, effects, and undo availability for the selected import job.
            </SheetDescription>
          </SheetHeader>

          {loadingDetailId && (!selectedDetail || selectedDetail.id !== loadingDetailId) ? (
            <div className="flex items-center gap-2 py-8 text-sm text-muted-foreground">
              <Loader2 className="h-4 w-4 animate-spin" />
              Loading import detail...
            </div>
          ) : selectedDetail ? (
            <div className="mt-6 space-y-6">
              <div className="grid gap-4 md:grid-cols-2">
                <div className="rounded-lg border p-4">
                  <p className="text-xs uppercase tracking-wide text-muted-foreground">File</p>
                  <p className="mt-2 text-sm font-medium">{selectedDetail.fileName}</p>
                </div>
                <div className="rounded-lg border p-4">
                  <p className="text-xs uppercase tracking-wide text-muted-foreground">Type</p>
                  <p className="mt-2 text-sm font-medium">{selectedDetail.importType}</p>
                </div>
                <div className="rounded-lg border p-4">
                  <p className="text-xs uppercase tracking-wide text-muted-foreground">Status</p>
                  <div className="mt-2 flex flex-wrap gap-2">
                    <Badge variant={getStatusBadgeVariant(selectedDetail.status)}>{selectedDetail.status}</Badge>
                    <Badge variant={getStatusBadgeVariant(selectedDetail.lifecycleState)}>{selectedDetail.lifecycleState}</Badge>
                  </div>
                </div>
                <div className="rounded-lg border p-4">
                  <p className="text-xs uppercase tracking-wide text-muted-foreground">Created</p>
                  <p className="mt-2 text-sm font-medium">{formatDateTime(selectedDetail.createdAt)}</p>
                </div>
                <div className="rounded-lg border p-4">
                  <p className="text-xs uppercase tracking-wide text-muted-foreground">Undone at</p>
                  <p className="mt-2 text-sm font-medium">{formatDateTime(selectedDetail.undoneAt)}</p>
                </div>
                <div className="rounded-lg border p-4">
                  <p className="text-xs uppercase tracking-wide text-muted-foreground">Job ID</p>
                  <p className="mt-2 text-sm font-medium">{selectedDetail.id}</p>
                </div>
              </div>

              <div className="grid gap-3 md:grid-cols-3">
                {parseSummary(selectedDetail.summary).map((entry) => (
                  <div key={entry.key} className="rounded-lg border bg-muted/30 p-4">
                    <p className="text-xs uppercase tracking-wide text-muted-foreground">{entry.label}</p>
                    <p className="mt-2 text-lg font-semibold">{entry.value}</p>
                  </div>
                ))}
              </div>

              <div className="flex flex-wrap items-center gap-3">
                <Button
                  variant="outline"
                  className="gap-2"
                  disabled={selectedDetail.lifecycleState === 'undone' || undoingId === selectedDetail.id}
                  onClick={() => {
                    setUndoTargetId(selectedDetail.id)
                    setUndoDialogOpen(true)
                  }}
                >
                  {undoingId === selectedDetail.id ? (
                    <Loader2 className="h-4 w-4 animate-spin" />
                  ) : (
                    <RotateCcw className="h-4 w-4" />
                  )}
                  {selectedDetail.lifecycleState === 'undone' ? 'Already Undone' : 'Undo Import'}
                </Button>
                <p className="text-sm text-muted-foreground">
                  Undo is available only while the import lifecycle has not already been reversed.
                </p>
              </div>

              <Card>
                <CardHeader>
                  <CardTitle className="text-base">Import Effects</CardTitle>
                  <CardDescription>Entities touched by this import job.</CardDescription>
                </CardHeader>
                <CardContent>
                  <div className="overflow-x-auto">
                    <Table>
                      <TableHeader>
                        <TableRow>
                          <TableHead>Effect</TableHead>
                          <TableHead>Entity Type</TableHead>
                          <TableHead>Entity ID</TableHead>
                        </TableRow>
                      </TableHeader>
                      <TableBody>
                        {selectedDetail.effects.map((effect) => (
                          <TableRow key={effect.id}>
                            <TableCell>{effect.effectType}</TableCell>
                            <TableCell>{effect.targetEntityType}</TableCell>
                            <TableCell className="font-mono text-xs">{effect.targetEntityId}</TableCell>
                          </TableRow>
                        ))}
                        {selectedDetail.effects.length === 0 && (
                          <TableRow>
                            <TableCell colSpan={3} className="py-6 text-center text-sm text-muted-foreground">
                              No persisted effects were recorded for this import.
                            </TableCell>
                          </TableRow>
                        )}
                      </TableBody>
                    </Table>
                  </div>
                </CardContent>
              </Card>

              <Card>
                <CardHeader>
                  <CardTitle className="text-base">Row Outcomes</CardTitle>
                  <CardDescription>Detailed row-by-row validation and application results.</CardDescription>
                </CardHeader>
                <CardContent>
                  <RowResultsTable rows={selectedDetail.rows} importType={selectedDetail.importType} />
                </CardContent>
              </Card>
            </div>
          ) : (
            <div className="py-8 text-sm text-muted-foreground">Select an import row to inspect its details.</div>
          )}
        </SheetContent>
      </Sheet>

      <Dialog open={undoDialogOpen} onOpenChange={setUndoDialogOpen}>
        <DialogContent>
          <DialogHeader>
            <DialogTitle>Undo Import</DialogTitle>
            <DialogDescription>
              This reverses the recorded import effects for the selected job when the backend allows it.
            </DialogDescription>
          </DialogHeader>
          <div className="space-y-2 rounded-lg bg-muted/40 p-4 text-sm">
            <p>
              <span className="font-medium">File:</span> {undoTarget?.fileName ?? '—'}
            </p>
            <p>
              <span className="font-medium">Status:</span> {undoTarget?.status ?? '—'}
            </p>
            <p>
              <span className="font-medium">Lifecycle:</span> {undoTarget?.lifecycleState ?? '—'}
            </p>
          </div>
          <DialogFooter>
            <Button variant="outline" onClick={() => setUndoDialogOpen(false)}>
              Cancel
            </Button>
            <Button onClick={() => void handleConfirmUndo()} disabled={!undoTarget || undoingId === undoTarget.id} className="gap-2">
              {undoingId ? <Loader2 className="h-4 w-4 animate-spin" /> : <RotateCcw className="h-4 w-4" />}
              Confirm Undo
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>
    </div>
  )
}
