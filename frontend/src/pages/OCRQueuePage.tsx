import { useEffect, useRef, useState } from 'react'
import type { ChangeEvent } from 'react'
import { useNavigate, useSearchParams } from 'react-router-dom'
import { useSWRConfig } from 'swr'
import { Loader2 } from 'lucide-react'

import { Button } from '@/components/ui/button'
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import { Badge } from '@/components/ui/badge'
import { Checkbox } from '@/components/ui/checkbox'
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from '@/components/ui/select'
import { ItemCombobox } from '@/components/ui/item-combobox'
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from '@/components/ui/table'
import { Separator } from '@/components/ui/separator'
import { MasterItemAliasDialog } from '@/components/MasterItemAliasDialog'

import { useAuthSession } from '@/hooks/useAuthSession'
import { useMasterData } from '@/hooks/useMasterData'
import { useMasterItems } from '@/hooks/useMasterItems'
import { useOCRJobDetail } from '@/hooks/useOCRJobDetail'
import { useOCRJobs } from '@/hooks/useOCRJobs'
import { useProcurementBudgetCategories } from '@/hooks/useProcurementBudgetCategories'
import {
  assistOCRLine,
  createProcurementDraftFromOCR,
  deleteOCRJob,
  retryOCRJob,
  registerOCRItem,
  updateOCRReview,
  uploadOCRJob,
} from '@/lib/mockApi'
import type {
  OCRJobDetailResponse,
  OCRLineAssistSuggestion,
  OCRLineUpdate,
  OCRRegisterItemInput,
  OCRResultLine,
  OCRReviewUpdateInput,
} from '@/types'

type RegisterDraft = OCRRegisterItemInput
type ReviewHeaderOverride = Partial<Pick<OCRReviewUpdateInput, 'supplierId' | 'quotationNumber' | 'issueDate'>>

const UNMATCHED_SUPPLIER_VALUE = '__unmatched_supplier__'
const UNASSIGNED_BUDGET_VALUE = '__unassigned_budget__'
const NEW_CATEGORY_VALUE = '__new_category__'

export function OCRQueuePage() {
  const navigate = useNavigate()
  const [searchParams, setSearchParams] = useSearchParams()
  const { mutate } = useSWRConfig()
  const { data: session, isLoading: sessionLoading } = useAuthSession()
  const [filterMyJobs, setFilterMyJobs] = useState(true)
  const currentUserId = session?.user?.userId ?? ''
  const sessionReady = !!session?.authenticated && session.user.status === 'active'
  const jobsReady = sessionReady && (!filterMyJobs || currentUserId !== '')
  const { data: jobs } = useOCRJobs(filterMyJobs ? currentUserId : undefined, jobsReady)
  const { data: masterData } = useMasterData(sessionReady)
  const { data: masterItems } = useMasterItems(sessionReady)
  const { data: budgetCategories } = useProcurementBudgetCategories(undefined, sessionReady)
  const uploadInputRef = useRef<HTMLInputElement>(null)
  const jobRows = jobs?.rows ?? []
  const selectedIDFromSearch = searchParams.get('jobId') || ''

  const [selectedIDOverride, setSelectedIDOverride] = useState('')

  const [pendingUploadFile, setPendingUploadFile] = useState<File | null>(null)
  const [uploadingJob, setUploadingJob] = useState(false)
  const selectedIDCandidate = selectedIDFromSearch || selectedIDOverride
  const selectedID =
    selectedIDCandidate && jobRows.some((job) => job.id === selectedIDCandidate)
      ? selectedIDCandidate
      : jobRows[0]?.id || ''
  const { data: detail } = useOCRJobDetail(selectedID, sessionReady)
  const supplierMatches = detail?.supplierMatch ?? []

  const [assistByLine, setAssistByLine] = useState<Record<string, OCRLineAssistSuggestion>>({})
  const [draftByLine, setDraftByLine] = useState<Record<string, RegisterDraft>>({})
  const [reviewHeaderByJob, setReviewHeaderByJob] = useState<Record<string, ReviewHeaderOverride>>({})
  const [reviewLinesByJob, setReviewLinesByJob] = useState<Record<string, Record<string, Partial<OCRLineUpdate>>>>({})
  const [busyLineId, setBusyLineId] = useState('')
  const [creatingDraft, setCreatingDraft] = useState(false)
  const [savingReview, setSavingReview] = useState(false)
  const [retryingJob, setRetryingJob] = useState(false)
  const [uploadError, setUploadError] = useState('')
  const [draftError, setDraftError] = useState('')
  const [reviewError, setReviewError] = useState('')
  const [deletingJobId, setDeletingJobId] = useState('')
  const [aliasLineId, setAliasLineId] = useState('')
  const [ocrElapsedSec, setOcrElapsedSec] = useState(0)

  // Elapsed-time counter while OCR is running
  useEffect(() => {
    if (!uploadingJob) return
    const t = setInterval(() => setOcrElapsedSec((s) => s + 1), 1000)
    return () => clearInterval(t)
  }, [uploadingJob])

  const currentHeader = detail ? getReviewHeader(detail, reviewHeaderByJob[selectedID]) : null
  const currentLines = detail ? (detail.lines ?? []).map((line) => getReviewLine(line, reviewLinesByJob[selectedID]?.[line.id])) : []
  const masterItemOptions = (masterItems?.rows ?? []).map((item) => ({
    itemId: item.id,
    itemNumber: item.itemNumber,
    description: item.description,
    manufacturer: item.manufacturerKey,
    category: item.categoryKey,
  }))
  const hasReviewOverrides =
    !!reviewHeaderByJob[selectedID] || Object.keys(reviewLinesByJob[selectedID] ?? {}).length > 0
  const unresolvedLineCount = currentLines.filter((line) => !line.itemId).length
  const unconfirmedLineCount = currentLines.filter((line) => !line.isUserConfirmed).length
  const canCreateDraft =
    !!detail &&
    !!currentHeader?.supplierId &&
    !!currentHeader?.quotationNumber &&
    !!currentHeader?.issueDate &&
    currentLines.length > 0 &&
    unresolvedLineCount === 0 &&
    unconfirmedLineCount === 0

  function handleFileSelection(event: ChangeEvent<HTMLInputElement>) {
    const file = event.target.files?.[0]
    if (!file) {
      setPendingUploadFile(null)
      return
    }
    setUploadError('')
    setPendingUploadFile(file)
  }

  function clearPendingUpload() {
    setPendingUploadFile(null)
    if (uploadInputRef.current) {
      uploadInputRef.current.value = ''
    }
  }

  function setSelectedID(nextID: string) {
    if (searchParams.has('jobId')) {
      const nextSearchParams = new URLSearchParams(searchParams)
      nextSearchParams.delete('jobId')
      setSearchParams(nextSearchParams, { replace: true })
    }
    setSelectedIDOverride(nextID)
  }

  async function handleStartOCR() {
    if (!pendingUploadFile) {
      return
    }
    setOcrElapsedSec(0)
    setUploadingJob(true)
    setUploadError('')
    try {
      const result = await uploadOCRJob(pendingUploadFile)
      await revalidateOCRJobs()
      const nextID = (result as { data?: { id?: string } } | undefined)?.data?.id
      if (nextID) {
        setSelectedID(nextID)
      }
      clearPendingUpload()
    } catch (error) {
      setUploadError(error instanceof Error ? error.message : 'Failed to start OCR job')
    } finally {
      setUploadingJob(false)
    }
  }

  async function handleAssist(line: OCRResultLine) {
    setBusyLineId(line.id)
    try {
      const suggestion = await assistOCRLine(selectedID, { lineId: line.id })
      setAssistByLine((current) => ({ ...current, [line.id]: suggestion }))
      setDraftByLine((current) => ({
        ...current,
        [line.id]: {
          ...defaultRegisterDraft(line, currentHeader?.supplierId ?? ''),
          canonicalItemNumber: suggestion.suggestedCanonicalNumber || line.itemNumber,
          description: line.itemDescription,
          manufacturerName: suggestion.suggestedManufacturer || line.manufacturerName,
          categoryKey: suggestion.suggestedCategoryKey || current[line.id]?.categoryKey || masterData?.categories[0]?.key || 'misc',
          categoryName:
            masterData?.categories.find((category) => category.key === suggestion.suggestedCategoryKey)?.name ||
            current[line.id]?.categoryName ||
            formatCategoryName(suggestion.suggestedCategoryKey) ||
            masterData?.categories[0]?.name ||
            formatCategoryName('misc'),
          defaultSupplierId: currentHeader?.supplierId ?? '',
          supplierAliasNumber: suggestion.suggestedAliasNumber || line.itemNumber,
          unitsPerOrder: current[line.id]?.unitsPerOrder || 1,
        },
      }))
    } finally {
      setBusyLineId('')
    }
  }

  async function handleRegister(lineId: string) {
    const line = detail?.lines.find((candidate) => candidate.id === lineId)
    const draft = draftByLine[lineId] ?? (line ? defaultRegisterDraft(line, currentHeader?.supplierId ?? '') : undefined)
    if (!draft) {
      return
    }
    const categoryKey = normalizeCategoryKey(draft.categoryKey || draft.categoryName)
    const categoryName = draft.categoryName.trim() || formatCategoryName(categoryKey)
    if (!categoryKey) {
      setDraftError('Category key is required. Use letters, numbers, and hyphens for a new category.')
      return
    }
    if (!categoryName) {
      setDraftError('Category name is required.')
      return
    }
    setBusyLineId(lineId)
    setDraftError('')
    try {
      await registerOCRItem(selectedID, {
        ...draft,
        categoryKey,
        categoryName,
      })
      await Promise.all([mutate(['ocr-job-detail', selectedID]), mutate('master-data')])
    } catch (error) {
      setDraftError(error instanceof Error ? error.message : 'Failed to register item')
    } finally {
      setBusyLineId('')
    }
  }

  async function handleSaveReview() {
    if (!detail || !currentHeader) {
      return
    }
    setSavingReview(true)
    setReviewError('')
    try {
      const payload: OCRReviewUpdateInput = {
        supplierId: currentHeader.supplierId,
        quotationNumber: currentHeader.quotationNumber,
        issueDate: currentHeader.issueDate,
        lines: currentLines.map((line) => ({
          id: line.id,
          itemId: line.itemId,
          deliveryLocation: line.deliveryLocation,
          budgetCategoryId: line.budgetCategoryId,
          accountingCategory: line.accountingCategory,
          supplierContact: line.supplierContact,
          isUserConfirmed: line.isUserConfirmed,
        })),
      }
      await updateOCRReview(selectedID, payload)
      clearReviewOverrides(selectedID)
      await Promise.all([mutate(['ocr-job-detail', selectedID]), revalidateOCRJobs()])
    } catch (error) {
      setReviewError(error instanceof Error ? error.message : 'Failed to save OCR review')
      throw error
    } finally {
      setSavingReview(false)
    }
  }

  async function handleDraftAction() {
    if (!detail) {
      return
    }
    setDraftError('')
    if (detail.procurementRequestId) {
      navigate(`/app/procurement/requests/${detail.procurementRequestId}`)
      return
    }
    setCreatingDraft(true)
    try {
      if (hasReviewOverrides) {
        await handleSaveReview()
      }
      const result = await createProcurementDraftFromOCR(selectedID)
      await Promise.all([revalidateOCRJobs(), mutate(['ocr-job-detail', selectedID]), mutate('procurement-requests')])
      navigate(`/app/procurement/requests/${result.procurementRequestId}`)
    } catch (error) {
      setDraftError(error instanceof Error ? error.message : 'Failed to create procurement draft')
    } finally {
      setCreatingDraft(false)
    }
  }

  async function handleRetry() {
    if (!detail) {
      return
    }
    setRetryingJob(true)
    setReviewError('')
    try {
      await retryOCRJob(detail.id)
      await Promise.all([revalidateOCRJobs(), mutate(['ocr-job-detail', selectedID])])
    } catch (error) {
      setReviewError(error instanceof Error ? error.message : 'Failed to retry OCR job')
    } finally {
      setRetryingJob(false)
    }
  }

  async function handleDeleteJob(jobId: string) {
    if (!window.confirm('このジョブを削除しますか？アーティファクトも削除されます。')) {
      return
    }
    setDeletingJobId(jobId)
    try {
      await deleteOCRJob(jobId)
      if (selectedID === jobId) {
        setSelectedID('')
      }
      await revalidateOCRJobs()
    } catch (error) {
      setUploadError(error instanceof Error ? error.message : 'Failed to delete OCR job')
    } finally {
      setDeletingJobId('')
    }
  }

  function updateRegisterDraft(lineId: string, patch: Partial<RegisterDraft>) {
    setDraftByLine((current) => ({
      ...current,
      [lineId]: {
        ...(current[lineId] ?? defaultRegisterDraft(detail?.lines.find((line) => line.id === lineId) ?? emptyLine(lineId), currentHeader?.supplierId ?? '')),
        ...patch,
      },
    }))
  }

  function updateReviewHeader(patch: ReviewHeaderOverride) {
    setReviewHeaderByJob((current) => ({
      ...current,
      [selectedID]: {
        ...current[selectedID],
        ...patch,
      },
    }))
  }

  function updateReviewLine(lineId: string, patch: Partial<OCRLineUpdate>) {
    setReviewLinesByJob((current) => ({
      ...current,
      [selectedID]: {
        ...(current[selectedID] ?? {}),
        [lineId]: {
          ...(current[selectedID]?.[lineId] ?? {}),
          ...patch,
        },
      },
    }))
  }

  function revalidateOCRJobs() {
    return mutate((key: unknown) => {
      if (typeof key === 'string') return key === 'ocr-jobs'
      if (Array.isArray(key)) return key[0] === 'ocr-jobs'
      return false
    })
  }

  function clearReviewOverrides(jobId: string) {
    setReviewHeaderByJob((current) => {
      const next = { ...current }
      delete next[jobId]
      return next
    })
    setReviewLinesByJob((current) => {
      const next = { ...current }
      delete next[jobId]
      return next
    })
  }

  const aliasLine = currentLines.find((line) => line.id === aliasLineId)

  if (sessionLoading || !sessionReady) {
    return (
      <div className="p-6 space-y-6">
        <div>
          <h1 className="text-3xl font-bold tracking-tight">OCR Queue</h1>
          <p className="text-muted-foreground mt-1">Upload and review quotations for OCR extraction</p>
        </div>

        <Card>
          <CardHeader>
            <CardTitle>Preparing session</CardTitle>
            <CardDescription>Waiting for your authenticated session before loading OCR data.</CardDescription>
          </CardHeader>
        </Card>
      </div>
    )
  }

  return (
    <div className="p-6 space-y-6">
      <div>
        <h1 className="text-3xl font-bold tracking-tight">OCR Queue</h1>
        <p className="text-muted-foreground mt-1">Upload and review quotations for OCR extraction</p>
      </div>

      <div className="grid grid-cols-3 gap-6">
        <Card className="col-span-2">
          <CardHeader>
            <CardTitle>Jobs</CardTitle>
            <CardDescription>Uploaded quotation files and their processing status</CardDescription>
          </CardHeader>
          <CardContent className="space-y-4">
            <div className="space-y-3">
              <div className="space-y-2">
                <Label htmlFor="file-upload">Upload quotation PDF or image</Label>
                <Input
                  id="file-upload"
                  ref={uploadInputRef}
                  type="file"
                  accept=".pdf,image/*"
                  onChange={handleFileSelection}
                />
                <p className="text-sm text-muted-foreground">Select a file first, then start OCR when you are ready.</p>
              </div>
              {pendingUploadFile && (
                <div className="rounded-lg border bg-muted/40 p-3">
                  <p className="text-sm font-medium">{pendingUploadFile.name}</p>
                  <p className="text-xs text-muted-foreground mt-1">
                    {pendingUploadFile.type || 'unknown type'} - {(pendingUploadFile.size / 1024).toFixed(1)} KB
                  </p>
                </div>
              )}
              <div className="flex gap-2">
                <Button onClick={() => void handleStartOCR()} disabled={!pendingUploadFile || uploadingJob}>
                  {uploadingJob && <Loader2 className="mr-2 h-4 w-4 animate-spin" />}
                  {uploadingJob ? 'OCR Processing...' : 'Start OCR'}
                </Button>
                <Button variant="outline" onClick={clearPendingUpload} disabled={!pendingUploadFile || uploadingJob}>
                  Clear Selection
                </Button>
              </div>
              {uploadingJob && (
                <div className="flex items-center gap-3 rounded-lg border bg-muted/40 p-3 text-sm text-muted-foreground">
                  <Loader2 className="h-5 w-5 animate-spin text-primary" />
                  <div>
                    <p className="font-medium text-foreground">LLMでOCR解析中...</p>
                    <p>ドキュメントの内容を読み取っています。通常10〜30秒ほどかかります。（{ocrElapsedSec}秒経過）</p>
                  </div>
                </div>
              )}
              {uploadError && <p className="text-sm text-destructive">{uploadError}</p>}
            </div>
            <div className="flex items-center gap-2 mb-2">
              <label className="flex items-center space-x-2">
                <Checkbox
                  checked={filterMyJobs}
                  onCheckedChange={(checked) => setFilterMyJobs(checked === true)}
                />
                <span className="text-sm">自分のジョブのみ表示</span>
              </label>
            </div>
            <div className="border rounded-lg overflow-hidden">
              <Table>
                <TableHeader>
                  <TableRow>
                    <TableHead>File</TableHead>
                    <TableHead>Status</TableHead>
                    <TableHead>Provider</TableHead>
                    <TableHead>Retries</TableHead>
                    <TableHead>Updated</TableHead>
                    <TableHead></TableHead>
                  </TableRow>
                </TableHeader>
                <TableBody>
                  {jobRows.length ? jobRows.map((job) => (
                      <TableRow
                        key={job.id}
                        onClick={() => setSelectedID(job.id)}
                        className={`cursor-pointer ${selectedID === job.id ? 'bg-muted' : ''}`}
                      >
                      <TableCell className="font-medium">{job.fileName}</TableCell>
                      <TableCell>
                        <Badge variant="outline">{job.status}</Badge>
                      </TableCell>
                      <TableCell className="text-sm text-muted-foreground">{job.provider}</TableCell>
                      <TableCell className="text-sm">{job.retryCount}</TableCell>
                      <TableCell className="text-sm text-muted-foreground">{job.updatedAt}</TableCell>
                      <TableCell>
                        <Button
                          variant="ghost"
                          size="sm"
                          onClick={(e) => { e.stopPropagation(); void handleDeleteJob(job.id) }}
                          disabled={deletingJobId === job.id}
                          className="text-destructive hover:text-destructive"
                        >
                          {deletingJobId === job.id ? '...' : '削除'}
                        </Button>
                      </TableCell>
                    </TableRow>
                  )) : (
                    <TableRow>
                      <TableCell colSpan={6} className="text-sm text-muted-foreground">
                        No OCR jobs yet. Select a quotation and start OCR to create the first job.
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
            <CardTitle className="text-base">Status</CardTitle>
          </CardHeader>
          <CardContent className="space-y-3">
            <div>
              <p className="text-sm text-muted-foreground">Unresolved Lines</p>
              <p className="text-2xl font-bold text-orange-600">{unresolvedLineCount}</p>
            </div>
            <Separator />
            <div>
              <p className="text-sm text-muted-foreground">Unconfirmed Lines</p>
              <p className="text-2xl font-bold text-blue-600">{unconfirmedLineCount}</p>
            </div>
          </CardContent>
        </Card>
      </div>

      {detail && (
        <Card>
          <CardHeader>
            <CardTitle>OCR Review</CardTitle>
            <CardDescription>Review and resolve OCR extraction results</CardDescription>
          </CardHeader>
          <CardContent className="space-y-6">
            <div className="grid grid-cols-2 gap-4 p-4 bg-muted rounded-lg">
              <div>
                <p className="text-sm text-muted-foreground">Supplier (OCR)</p>
                <p className="font-medium">{detail.supplierName || '-'}</p>
              </div>
              <div>
                <p className="text-sm text-muted-foreground">Supplier (Matched)</p>
                <p className="font-medium">
                  {supplierMatches[0]
                    ? `${supplierMatches[0].name} (${Math.round(supplierMatches[0].score * 100)}%)`
                    : detail.supplierId || '-'}
                </p>
              </div>
              <div>
                <p className="text-sm text-muted-foreground">Linked Request</p>
                <p className="font-medium">{detail.procurementBatchNumber || detail.procurementRequestId || '-'}</p>
              </div>
              <div>
                <p className="text-sm text-muted-foreground">Retry Count</p>
                <p className="font-medium">{detail.retryCount ?? 0}</p>
              </div>
            </div>

            <Separator />

            <div className="space-y-4">
              <h3 className="font-semibold">Header Information</h3>
              <div className="grid grid-cols-2 gap-4">
                <div className="space-y-2">
                  <Label htmlFor="supplier">Supplier</Label>
                  <Select
                    value={currentHeader?.supplierId || UNMATCHED_SUPPLIER_VALUE}
                    onValueChange={(value) => updateReviewHeader({ supplierId: value === UNMATCHED_SUPPLIER_VALUE ? '' : value })}
                  >
                    <SelectTrigger id="supplier">
                      <SelectValue placeholder="Select supplier" />
                    </SelectTrigger>
                    <SelectContent>
                      <SelectItem value={UNMATCHED_SUPPLIER_VALUE}>Unmatched</SelectItem>
                      {dedupeSuppliers(detail, masterData).map((supplier) => (
                        <SelectItem key={supplier.id} value={supplier.id}>
                          {supplier.name}
                        </SelectItem>
                      ))}
                    </SelectContent>
                  </Select>
                </div>
                <div className="space-y-2">
                  <Label htmlFor="quotation">Quotation Number</Label>
                  <Input
                    id="quotation"
                    value={currentHeader?.quotationNumber ?? ''}
                    onChange={(event) => updateReviewHeader({ quotationNumber: event.target.value })}
                  />
                </div>
                <div className="space-y-2">
                  <Label htmlFor="issue-date">Issue Date</Label>
                  <Input
                    id="issue-date"
                    value={currentHeader?.issueDate ?? ''}
                    onChange={(event) => updateReviewHeader({ issueDate: event.target.value })}
                  />
                </div>
                <div className="space-y-2">
                  <Label htmlFor="error">OCR Error</Label>
                  <Input id="error" value={detail.errorMessage ?? ''} disabled />
                </div>
              </div>
            </div>

            <Separator />

            {/* Lines summary table */}
            <div className="space-y-4">
              <h3 className="font-semibold">Lines ({currentLines.length})</h3>
              <div className="border rounded-lg overflow-hidden">
                <Table>
                  <TableHeader>
                    <TableRow>
                      <TableHead>Manufacturer</TableHead>
                      <TableHead>Item No.</TableHead>
                      <TableHead>Matched Item</TableHead>
                      <TableHead>Description</TableHead>
                      <TableHead>Qty</TableHead>
                      <TableHead>Lead Time</TableHead>
                    </TableRow>
                  </TableHeader>
                  <TableBody>
                    {currentLines.map((line) => (
                      <TableRow key={line.id}>
                        <TableCell className="text-sm">{line.manufacturerName}</TableCell>
                        <TableCell className="font-mono text-sm">{line.itemNumber}</TableCell>
                        <TableCell>
                          {line.matchCandidates[0]
                            ? `${line.matchCandidates[0].canonicalItemNumber} (${Math.round(line.matchCandidates[0].score * 100)}%)`
                            : line.itemId || '-'}
                        </TableCell>
                        <TableCell className="text-sm">{line.itemDescription}</TableCell>
                        <TableCell className="text-sm">{line.quantity}</TableCell>
                        <TableCell className="text-sm">{line.leadTimeDays} days</TableCell>
                      </TableRow>
                    ))}
                  </TableBody>
                </Table>
              </div>
            </div>

            <Separator />

            {/* ── Section: Line Review (grouped) ── */}
            <div className="space-y-4">
              <h3 className="font-semibold">Line Review</h3>
              <p className="text-sm text-muted-foreground">
                Confirm or adjust the resolved item and procurement details for each line.
              </p>
              <div className="rounded-lg border border-blue-200 bg-blue-50/40 dark:border-blue-900 dark:bg-blue-950/20 p-4 space-y-4">
                {currentLines.map((line) => {
                  const assist = assistByLine[line.id]
                  return (
                    <Card key={`${line.id}-review`} className="border shadow-sm">
                      <CardHeader className="pb-3">
                        <div className="flex items-center justify-between">
                          <div>
                            <CardTitle className="text-base">{line.itemNumber || line.id}</CardTitle>
                            <CardDescription>
                              {line.matchCandidates.slice(0, 2).map((c) => `${c.canonicalItemNumber} (${Math.round(c.score * 100)}%)`).join(' \u2022 ') ||
                                'No deterministic candidates'}
                            </CardDescription>
                          </div>
                          <div className="flex items-center gap-2">
                            {assist && (
                              <span className="text-xs text-muted-foreground">
                                {Math.round(assist.confidence * 100)}%
                              </span>
                            )}
                            <Button
                              variant="outline"
                              size="sm"
                              onClick={() => handleAssist(line)}
                              disabled={busyLineId === line.id}
                            >
                              {busyLineId === line.id && <Loader2 className="mr-2 h-3 w-3 animate-spin" />}
                              {busyLineId === line.id ? 'LLM 解析中...' : 'LLM Assist'}
                            </Button>
                            <Button
                              variant="outline"
                              size="sm"
                              onClick={() => setAliasLineId(line.id)}
                              disabled={!line.itemId || !currentHeader?.supplierId}
                            >
                              Add Alias
                            </Button>
                          </div>
                        </div>
                        {assist && (
                          <p className="text-xs text-muted-foreground mt-1">{assist.rationale}</p>
                        )}
                      </CardHeader>
                      <CardContent className="pt-0">
                        <div className="grid grid-cols-3 gap-4">
                          <div className="space-y-2">
                            <Label htmlFor={`item-${line.id}`}>Resolved Item</Label>
                            <ItemCombobox
                              items={mergeLineItemOptions(line, masterItemOptions)}
                              value={line.itemId}
                              onValueChange={(value) => updateReviewLine(line.id, { itemId: value })}
                              onCreateNew={() => {
                                updateRegisterDraft(line.id, defaultRegisterDraft(line, currentHeader?.supplierId ?? ''))
                              }}
                              placeholder="Search existing item or leave unresolved"
                            />
                            {line.itemId && currentHeader?.supplierId && (
                              <Button type="button" variant="ghost" size="sm" className="px-0" onClick={() => setAliasLineId(line.id)}>
                                Register OCR number as supplier alias
                              </Button>
                            )}
                          </div>
                          <div className="space-y-2">
                            <Label htmlFor={`location-${line.id}`}>Delivery Location</Label>
                            <Input
                              id={`location-${line.id}`}
                              value={line.deliveryLocation}
                              onChange={(event) => updateReviewLine(line.id, { deliveryLocation: event.target.value })}
                            />
                          </div>
                          <div className="space-y-2">
                            <Label htmlFor={`budget-${line.id}`}>Budget Category</Label>
                            <Select
                              value={line.budgetCategoryId || UNASSIGNED_BUDGET_VALUE}
                              onValueChange={(value) =>
                                updateReviewLine(line.id, { budgetCategoryId: value === UNASSIGNED_BUDGET_VALUE ? '' : value })
                              }
                            >
                              <SelectTrigger id={`budget-${line.id}`}>
                                <SelectValue placeholder="Unassigned" />
                              </SelectTrigger>
                              <SelectContent>
                                <SelectItem value={UNASSIGNED_BUDGET_VALUE}>Unassigned</SelectItem>
                                {budgetCategories?.map((category) => (
                                  <SelectItem key={category.id} value={category.id}>
                                    {category.name}
                                  </SelectItem>
                                ))}
                              </SelectContent>
                            </Select>
                          </div>
                          <div className="space-y-2">
                            <Label htmlFor={`accounting-${line.id}`}>Accounting Category</Label>
                            <Input
                              id={`accounting-${line.id}`}
                              value={line.accountingCategory}
                              onChange={(event) => updateReviewLine(line.id, { accountingCategory: event.target.value })}
                            />
                          </div>
                          <div className="space-y-2">
                            <Label htmlFor={`contact-${line.id}`}>Supplier Contact</Label>
                            <Input
                              id={`contact-${line.id}`}
                              value={line.supplierContact}
                              onChange={(event) => updateReviewLine(line.id, { supplierContact: event.target.value })}
                            />
                          </div>
                          <div className="flex items-end pb-1">
                            <label className="flex items-center space-x-2">
                              <Checkbox
                                checked={line.isUserConfirmed}
                                onCheckedChange={(checked) =>
                                  updateReviewLine(line.id, { isUserConfirmed: checked === true })
                                }
                              />
                              <span className="text-sm">User Confirmed</span>
                            </label>
                          </div>
                        </div>
                      </CardContent>
                    </Card>
                  )
                })}
              </div>
            </div>

            {/* ── Section: Register New Items (grouped, only if unresolved lines exist) ── */}
            {currentLines.some((line) => !line.itemId) && (
              <>
                <Separator />
                <div className="space-y-4">
                  <h3 className="font-semibold">
                    Register New Items ({currentLines.filter((line) => !line.itemId).length})
                  </h3>
                  <p className="text-sm text-muted-foreground">
                    The following lines have no matching item in the master catalog. Register them as new items to proceed.
                  </p>
                  <div className="rounded-lg border border-emerald-200 bg-emerald-50/40 dark:border-emerald-900 dark:bg-emerald-950/20 p-4 space-y-4">
                    {currentLines.filter((line) => !line.itemId).map((line) => {
                      const registerDraft = draftByLine[line.id] ?? defaultRegisterDraft(line, currentHeader?.supplierId ?? '')
                      const categoryOptions = masterData?.categories ?? []
                      const selectedCategory = categoryOptions.find((category) => category.key === registerDraft.categoryKey)
                      const isCustomCategory = !selectedCategory
                      return (
                        <Card key={`${line.id}-register`} className="border shadow-sm">
                          <CardHeader className="pb-3">
                            <CardTitle className="text-base">{line.itemNumber || line.id}</CardTitle>
                            <CardDescription>{line.itemDescription}</CardDescription>
                          </CardHeader>
                          <CardContent className="space-y-4 pt-0">
                            <div className="grid grid-cols-3 gap-4">
                              <div className="space-y-2">
                                <Label htmlFor={`canon-${line.id}`}>Canonical Item Number</Label>
                                <Input
                                  id={`canon-${line.id}`}
                                  value={registerDraft.canonicalItemNumber}
                                  onChange={(event) =>
                                    updateRegisterDraft(line.id, { canonicalItemNumber: event.target.value })
                                  }
                                />
                              </div>
                              <div className="space-y-2">
                                <Label htmlFor={`desc-${line.id}`}>Description</Label>
                                <Input
                                  id={`desc-${line.id}`}
                                  value={registerDraft.description}
                                  onChange={(event) => updateRegisterDraft(line.id, { description: event.target.value })}
                                />
                              </div>
                              <div className="space-y-2">
                                <Label htmlFor={`mfg-${line.id}`}>Manufacturer</Label>
                                <Input
                                  id={`mfg-${line.id}`}
                                  value={registerDraft.manufacturerName}
                                  onChange={(event) =>
                                    updateRegisterDraft(line.id, { manufacturerName: event.target.value })
                                  }
                                />
                              </div>
                              <div className="space-y-2">
                               <Label htmlFor={`cat-${line.id}`}>Category</Label>
                                <Select
                                  value={selectedCategory?.key || NEW_CATEGORY_VALUE}
                                  onValueChange={(value) =>
                                    value === NEW_CATEGORY_VALUE
                                      ? updateRegisterDraft(line.id, {
                                          categoryKey: selectedCategory ? '' : registerDraft.categoryKey,
                                          categoryName: selectedCategory ? '' : registerDraft.categoryName,
                                        })
                                      : updateRegisterDraft(line.id, {
                                          categoryKey: value,
                                          categoryName: categoryOptions.find((c) => c.key === value)?.name || value,
                                        })
                                  }
                                >
                                  <SelectTrigger id={`cat-${line.id}`}>
                                    <SelectValue placeholder="Select category" />
                                  </SelectTrigger>
                                  <SelectContent>
                                    <SelectItem value={NEW_CATEGORY_VALUE}>Enter new category</SelectItem>
                                    {categoryOptions.map((category) => (
                                      <SelectItem key={category.key} value={category.key}>
                                        {category.name}
                                      </SelectItem>
                                    ))}
                                  </SelectContent>
                                </Select>
                              </div>
                              {isCustomCategory && (
                                <>
                                  <div className="space-y-2">
                                    <Label htmlFor={`cat-name-${line.id}`}>New Category Name</Label>
                                    <Input
                                      id={`cat-name-${line.id}`}
                                      value={registerDraft.categoryName}
                                      onChange={(event) =>
                                        updateRegisterDraft(line.id, {
                                          categoryName: event.target.value,
                                        })
                                      }
                                      placeholder="e.g. Cable Harness"
                                    />
                                  </div>
                                  <div className="space-y-2">
                                    <Label htmlFor={`cat-key-${line.id}`}>Category Key</Label>
                                    <Input
                                      id={`cat-key-${line.id}`}
                                      value={registerDraft.categoryKey}
                                      onChange={(event) =>
                                        updateRegisterDraft(line.id, {
                                          categoryKey: normalizeCategoryKey(event.target.value),
                                        })
                                      }
                                      placeholder={normalizeCategoryKey(registerDraft.categoryName) || 'e.g. cable-harness'}
                                    />
                                    <p className="text-xs text-muted-foreground">
                                      New categories are saved with this key. Use lowercase letters, numbers, and hyphens.
                                    </p>
                                  </div>
                                </>
                              )}
                              <div className="space-y-2">
                                <Label htmlFor={`alias-${line.id}`}>Supplier Alias</Label>
                                <Input
                                  id={`alias-${line.id}`}
                                  value={registerDraft.supplierAliasNumber}
                                  onChange={(event) =>
                                    updateRegisterDraft(line.id, { supplierAliasNumber: event.target.value })
                                  }
                                />
                              </div>
                              <div className="space-y-2">
                                <Label htmlFor={`upo-${line.id}`}>Units per Order</Label>
                                <Input
                                  id={`upo-${line.id}`}
                                  type="number"
                                  min={1}
                                  value={registerDraft.unitsPerOrder}
                                  onChange={(event) =>
                                    updateRegisterDraft(line.id, { unitsPerOrder: Number(event.target.value) || 1 })
                                  }
                                />
                              </div>
                            </div>
                            <Button
                              onClick={() => handleRegister(line.id)}
                              disabled={busyLineId === line.id}
                              className="w-full"
                            >
                              {busyLineId === line.id && <Loader2 className="mr-2 h-4 w-4 animate-spin" />}
                              {busyLineId === line.id ? 'Registering...' : 'Register as New Item'}
                            </Button>
                          </CardContent>
                        </Card>
                      )
                    })}
                  </div>
                </div>
              </>
            )}

            {reviewError && <p className="text-sm text-destructive">{reviewError}</p>}
            {draftError && <p className="text-sm text-destructive">{draftError}</p>}
          </CardContent>
        </Card>
      )}

      {/* ── Sticky Action Bar (visible when OCR detail is loaded) ── */}
      {detail && (
        <div className="sticky bottom-0 z-10 -mx-6 px-6 py-3 bg-background/95 backdrop-blur supports-[backdrop-filter]:bg-background/80 border-t shadow-[0_-2px_10px_rgba(0,0,0,0.06)]">
          <div className="flex items-center justify-between">
            <div className="flex items-center gap-3 text-sm text-muted-foreground">
              {unresolvedLineCount > 0 && (
                <span className="text-orange-600 font-medium">{unresolvedLineCount} unresolved</span>
              )}
              {unconfirmedLineCount > 0 && (
                <span className="text-blue-600 font-medium">{unconfirmedLineCount} unconfirmed</span>
              )}
              {unresolvedLineCount === 0 && unconfirmedLineCount === 0 && (
                <span className="text-green-600 font-medium">All lines resolved & confirmed</span>
              )}
            </div>
            <div className="flex gap-2">
              <Button
                variant="outline"
                onClick={handleRetry}
                disabled={retryingJob || detail.status === 'processing'}
              >
                {retryingJob && <Loader2 className="mr-2 h-4 w-4 animate-spin" />}
                {retryingJob ? 'Retrying...' : 'Retry OCR'}
              </Button>
              <Button
                variant="outline"
                onClick={() => void handleSaveReview()}
                disabled={!hasReviewOverrides || savingReview}
              >
                {savingReview && <Loader2 className="mr-2 h-4 w-4 animate-spin" />}
                {savingReview ? 'Saving...' : 'Save Review'}
              </Button>
              <Button
                onClick={handleDraftAction}
                disabled={creatingDraft || (!detail.procurementRequestId && !canCreateDraft)}
              >
                {creatingDraft && <Loader2 className="mr-2 h-4 w-4 animate-spin" />}
                {creatingDraft ? 'Creating...' : detail.procurementRequestId ? 'Open Procurement Draft' : 'Continue to Request (Draft)'}
              </Button>
            </div>
          </div>
        </div>
      )}
      <MasterItemAliasDialog
        open={!!aliasLine}
        mode="alias-only"
        masterData={masterData}
        masterItems={masterItems?.rows ?? []}
        initialItemId={aliasLine?.itemId ?? ''}
        initialSupplierId={currentHeader?.supplierId ?? ''}
        initialSupplierAliasNumber={aliasLine?.itemNumber ?? ''}
        onOpenChange={(open) => {
          if (!open) setAliasLineId('')
        }}
        onCompleted={() => {
          if (aliasLine) {
            updateReviewLine(aliasLine.id, { itemId: aliasLine.itemId, isUserConfirmed: true })
          }
          void Promise.all([mutate('master-data'), mutate('master-items'), mutate(['ocr-job-detail', selectedID])])
        }}
      />
    </div>
  )
}

function getReviewHeader(detail: OCRJobDetailResponse, override?: ReviewHeaderOverride) {
  return {
    supplierId: override?.supplierId ?? detail.supplierId,
    quotationNumber: override?.quotationNumber ?? detail.quotationNumber,
    issueDate: override?.issueDate ?? detail.issueDate,
  }
}

function getReviewLine(line: OCRResultLine, override?: Partial<OCRLineUpdate>): OCRResultLine {
  return {
    ...line,
    matchCandidates: line.matchCandidates ?? [],
    itemId: override?.itemId ?? line.itemId,
    deliveryLocation: override?.deliveryLocation ?? line.deliveryLocation,
    budgetCategoryId: override?.budgetCategoryId ?? line.budgetCategoryId,
    accountingCategory: override?.accountingCategory ?? line.accountingCategory,
    supplierContact: override?.supplierContact ?? line.supplierContact,
    isUserConfirmed: override?.isUserConfirmed ?? line.isUserConfirmed,
  }
}

function defaultRegisterDraft(line: OCRResultLine, supplierId: string): RegisterDraft {
  return {
    lineId: line.id,
    canonicalItemNumber: line.itemNumber,
    description: line.itemDescription,
    manufacturerName: line.manufacturerName,
    categoryKey: 'misc',
    categoryName: formatCategoryName('misc'),
    defaultSupplierId: supplierId,
    supplierAliasNumber: line.itemNumber,
    unitsPerOrder: 1,
  }
}

function normalizeCategoryKey(value: string) {
  return value
    .trim()
    .toLowerCase()
    .replace(/_/g, '-')
    .replace(/\s+/g, '-')
    .replace(/[^a-z0-9-]+/g, '')
    .replace(/-+/g, '-')
    .replace(/^-|-$/g, '')
}

function formatCategoryName(value: string) {
  return normalizeCategoryKey(value)
    .split('-')
    .filter(Boolean)
    .map((segment) => segment.charAt(0).toUpperCase() + segment.slice(1))
    .join(' ')
}

function dedupeSuppliers(detail?: OCRJobDetailResponse, masterData?: { suppliers: { id: string; name: string }[] }) {
  const map = new Map<string, string>()
  for (const supplier of masterData?.suppliers ?? []) {
    map.set(supplier.id, supplier.name)
  }
  for (const supplier of detail?.supplierMatch ?? []) {
    map.set(supplier.id, supplier.name)
  }
  if (detail?.supplierId && !map.has(detail.supplierId)) {
    map.set(detail.supplierId, detail.supplierName || detail.supplierId)
  }
  return Array.from(map.entries()).map(([id, name]) => ({ id, name }))
}

function mergeLineItemOptions(
  line: OCRResultLine,
  masterItems: { itemId: string; itemNumber: string; description: string; manufacturer?: string; category?: string }[],
) {
  const byId = new Map(masterItems.map((item) => [item.itemId, item]))
  for (const candidate of line.matchCandidates) {
    byId.set(candidate.itemId, {
      itemId: candidate.itemId,
      itemNumber: candidate.canonicalItemNumber,
      description: candidate.description,
      manufacturer: candidate.manufacturerName,
      category: `candidate ${Math.round(candidate.score * 100)}%`,
    })
  }
  if (line.itemId && !byId.has(line.itemId)) {
    byId.set(line.itemId, {
      itemId: line.itemId,
      itemNumber: line.itemNumber || line.itemId,
      description: line.itemDescription,
      manufacturer: line.manufacturerName,
      category: 'current',
    })
  }
  return Array.from(byId.values()).sort((left, right) => left.itemNumber.localeCompare(right.itemNumber))
}

function emptyLine(id: string): OCRResultLine {
  return {
    id,
    itemId: '',
    manufacturerName: '',
    itemNumber: '',
    itemDescription: '',
    quantity: 1,
    leadTimeDays: 0,
    deliveryLocation: '',
    budgetCategoryId: '',
    accountingCategory: '',
    supplierContact: '',
    isUserConfirmed: false,
    matchCandidates: [],
  }
}
