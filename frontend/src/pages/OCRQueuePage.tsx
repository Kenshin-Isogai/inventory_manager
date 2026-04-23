import { useState } from 'react'
import type { ChangeEvent } from 'react'
import { useNavigate, useSearchParams } from 'react-router-dom'
import { useSWRConfig } from 'swr'

import { SectionCard } from '../components/SectionCard'
import { useMasterData } from '../hooks/useMasterData'
import { useOCRJobDetail } from '../hooks/useOCRJobDetail'
import { useOCRJobs } from '../hooks/useOCRJobs'
import { useProcurementBudgetCategories } from '../hooks/useProcurementBudgetCategories'
import {
  assistOCRLine,
  createProcurementDraftFromOCR,
  retryOCRJob,
  registerOCRItem,
  updateOCRReview,
  uploadOCRJob,
} from '../lib/mockApi'
import type {
  OCRJobDetailResponse,
  OCRLineAssistSuggestion,
  OCRLineUpdate,
  OCRRegisterItemInput,
  OCRResultLine,
  OCRReviewUpdateInput,
} from '../types'

type RegisterDraft = OCRRegisterItemInput
type ReviewHeaderOverride = Partial<Pick<OCRReviewUpdateInput, 'supplierId' | 'quotationNumber' | 'issueDate'>>

export function OCRQueuePage() {
  const navigate = useNavigate()
  const [searchParams] = useSearchParams()
  const { mutate } = useSWRConfig()
  const { data: jobs } = useOCRJobs()
  const { data: masterData } = useMasterData()
  const { data: budgetCategories } = useProcurementBudgetCategories()

  const [selectedIDOverride, setSelectedIDOverride] = useState('')
  const selectedID =
    jobs?.rows.some((job) => job.id === selectedIDOverride) ? selectedIDOverride : jobs?.rows[0]?.id || selectedIDOverride || 'ocr-001'
  const { data: detail } = useOCRJobDetail(selectedID)

  const [assistByLine, setAssistByLine] = useState<Record<string, OCRLineAssistSuggestion>>({})
  const [draftByLine, setDraftByLine] = useState<Record<string, RegisterDraft>>({})
  const [reviewHeaderByJob, setReviewHeaderByJob] = useState<Record<string, ReviewHeaderOverride>>({})
  const [reviewLinesByJob, setReviewLinesByJob] = useState<Record<string, Record<string, Partial<OCRLineUpdate>>>>({})
  const [busyLineId, setBusyLineId] = useState('')
  const [creatingDraft, setCreatingDraft] = useState(false)
  const [savingReview, setSavingReview] = useState(false)
  const [retryingJob, setRetryingJob] = useState(false)
  const [draftError, setDraftError] = useState('')
  const [reviewError, setReviewError] = useState('')

  const currentHeader = detail ? getReviewHeader(detail, reviewHeaderByJob[selectedID]) : null
  const currentLines = detail ? detail.lines.map((line) => getReviewLine(line, reviewLinesByJob[selectedID]?.[line.id])) : []
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

  async function handleFileUpload(event: ChangeEvent<HTMLInputElement>) {
    const file = event.target.files?.[0]
    if (!file) {
      return
    }
    await uploadOCRJob(file)
    await mutate('ocr-jobs')
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
            masterData?.categories[0]?.name ||
            'misc',
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
    setBusyLineId(lineId)
    try {
      await registerOCRItem(selectedID, draft)
      await Promise.all([mutate(['ocr-job-detail', selectedID]), mutate('master-data')])
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
      await Promise.all([mutate(['ocr-job-detail', selectedID]), mutate('ocr-jobs')])
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
      navigateToProcurement(detail.procurementRequestId)
      return
    }
    setCreatingDraft(true)
    try {
      if (hasReviewOverrides) {
        await handleSaveReview()
      }
      const result = await createProcurementDraftFromOCR(selectedID)
      await Promise.all([mutate('ocr-jobs'), mutate(['ocr-job-detail', selectedID]), mutate('procurement-requests')])
      navigateToProcurement(result.procurementRequestId)
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
      await Promise.all([mutate('ocr-jobs'), mutate(['ocr-job-detail', selectedID])])
    } catch (error) {
      setReviewError(error instanceof Error ? error.message : 'Failed to retry OCR job')
    } finally {
      setRetryingJob(false)
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

  function navigateToProcurement(requestId: string) {
    const next = new URLSearchParams(searchParams)
    next.set('requestId', requestId)
    navigate({ pathname: '/procurement/requests', search: next.toString() })
  }

  return (
    <div className="page-grid">
      <SectionCard title="OCR Queue" subtitle="Quotation uploads are stored as artifacts and converted into draft OCR review jobs.">
        <div className="stack-form">
          <label>
            <span>Upload quotation PDF or image</span>
            <input type="file" accept=".pdf,image/*" onChange={handleFileUpload} />
          </label>
        </div>
        <table className="data-table">
          <thead>
            <tr>
              <th>File</th>
              <th>Status</th>
              <th>Provider</th>
              <th>Retries</th>
              <th>Updated</th>
            </tr>
          </thead>
          <tbody>
            {jobs?.rows.map((job) => (
              <tr key={job.id} onClick={() => setSelectedIDOverride(job.id)} className={selectedID === job.id ? 'selected-row' : ''}>
                <td>{job.fileName}</td>
                <td>{job.status}</td>
                <td>{job.provider}</td>
                <td>{job.retryCount}</td>
                <td>{job.updatedAt}</td>
              </tr>
            ))}
          </tbody>
        </table>
      </SectionCard>

      <SectionCard title="OCR Review" subtitle="Draft extraction stays editable before any procurement request is created.">
        <dl className="definition-list">
          <div>
            <dt>Supplier (OCR)</dt>
            <dd>{detail?.supplierName || '-'}</dd>
          </div>
          <div>
            <dt>Supplier (matched)</dt>
            <dd>
              {detail?.supplierMatch?.[0]
                ? `${detail.supplierMatch[0].name} (${Math.round(detail.supplierMatch[0].score * 100)}%)`
                : detail?.supplierId || '-'}
            </dd>
          </div>
          <div>
            <dt>Linked request</dt>
            <dd>{detail?.procurementBatchNumber || detail?.procurementRequestId || '-'}</dd>
          </div>
          <div>
            <dt>Retry count</dt>
            <dd>{detail?.retryCount ?? 0}</dd>
          </div>
          <div>
            <dt>Artifact</dt>
            <dd>{detail?.artifactPath}</dd>
          </div>
          <div>
            <dt>Raw payload</dt>
            <dd>{detail?.rawPayload}</dd>
          </div>
        </dl>

        <div className="stack-form">
          <div className="two-column">
            <label>
              <span>Supplier</span>
              <select value={currentHeader?.supplierId ?? ''} onChange={(event) => updateReviewHeader({ supplierId: event.target.value })}>
                <option value="">Unmatched</option>
                {dedupeSuppliers(detail, masterData).map((supplier) => (
                  <option key={supplier.id} value={supplier.id}>
                    {supplier.name}
                  </option>
                ))}
              </select>
            </label>
            <label>
              <span>Quotation number</span>
              <input
                value={currentHeader?.quotationNumber ?? ''}
                onChange={(event) => updateReviewHeader({ quotationNumber: event.target.value })}
              />
            </label>
          </div>
          <div className="two-column">
            <label>
              <span>Issue date</span>
              <input value={currentHeader?.issueDate ?? ''} onChange={(event) => updateReviewHeader({ issueDate: event.target.value })} />
            </label>
            <label>
              <span>OCR error</span>
              <input value={detail?.errorMessage ?? ''} disabled />
            </label>
          </div>
        </div>

        <div className="two-column">
          <button type="button" className="primary-button" disabled={!hasReviewOverrides || savingReview} onClick={() => void handleSaveReview()}>
            {savingReview ? 'Saving...' : 'Save Review'}
          </button>
          <button type="button" className="primary-button" disabled={retryingJob || !detail || detail.status === 'processing'} onClick={handleRetry}>
            {retryingJob ? 'Retrying...' : 'Retry OCR'}
          </button>
        </div>

        <div className="two-column">
          <button
            type="button"
            className="primary-button"
            disabled={creatingDraft || (!detail?.procurementRequestId && !canCreateDraft)}
            onClick={handleDraftAction}
          >
            {creatingDraft ? 'Creating...' : detail?.procurementRequestId ? 'Open Procurement Draft' : 'Create Procurement Draft'}
          </button>
          <span>
            {detail?.procurementRequestId
              ? 'This OCR job is already linked to a procurement draft.'
              : canCreateDraft
                ? 'All lines are resolved and user-confirmed, so the draft can be created.'
                : 'Resolve every line, save review values, and mark all lines as user-confirmed before creating a draft.'}
          </span>
        </div>

        {reviewError ? <p className="muted-copy">{reviewError}</p> : null}
        {draftError ? <p className="muted-copy">{draftError}</p> : null}

        <table className="data-table">
          <thead>
            <tr>
              <th>Manufacturer</th>
              <th>Item No.</th>
              <th>Matched item</th>
              <th>Description</th>
              <th>Qty</th>
              <th>Lead time</th>
            </tr>
          </thead>
          <tbody>
            {currentLines.map((line) => (
              <tr key={line.id}>
                <td>{line.manufacturerName}</td>
                <td>{line.itemNumber}</td>
                <td>
                  {line.matchCandidates[0]
                    ? `${line.matchCandidates[0].canonicalItemNumber} (${Math.round(line.matchCandidates[0].score * 100)}%)`
                    : line.itemId || '-'}
                </td>
                <td>{line.itemDescription}</td>
                <td>{line.quantity}</td>
                <td>{line.leadTimeDays} days</td>
              </tr>
            ))}
          </tbody>
        </table>

        <div className="stack-form">
          {currentLines.map((line) => {
            const assist = assistByLine[line.id]
            const registerDraft = draftByLine[line.id] ?? defaultRegisterDraft(line, currentHeader?.supplierId ?? '')
            const categoryOptions = masterData?.categories ?? []
            return (
              <div key={`${line.id}-actions`}>
                <strong>{line.itemNumber || line.id}</strong>
                <div>
                  {line.matchCandidates.slice(0, 3).map((candidate) => `${candidate.canonicalItemNumber} / ${candidate.matchReason}`).join(' | ') ||
                    'No deterministic candidates yet.'}
                </div>
                <div className="two-column">
                  <button type="button" className="primary-button" disabled={busyLineId === line.id} onClick={() => handleAssist(line)}>
                    {busyLineId === line.id ? 'Working...' : 'LLM Assist'}
                  </button>
                  <span>{assist ? `${Math.round(assist.confidence * 100)}% / ${assist.rationale}` : 'Use Vertex AI only for unresolved lines.'}</span>
                </div>
                <div className="two-column">
                  <label>
                    <span>Resolved item</span>
                    <select value={line.itemId} onChange={(event) => updateReviewLine(line.id, { itemId: event.target.value })}>
                      <option value="">Unresolved</option>
                      {buildItemOptions(line).map((option) => (
                        <option key={option.value || `empty-${line.id}`} value={option.value}>
                          {option.label}
                        </option>
                      ))}
                    </select>
                  </label>
                  <label>
                    <span>User confirmed</span>
                    <input
                      type="checkbox"
                      checked={line.isUserConfirmed}
                      onChange={(event) => updateReviewLine(line.id, { isUserConfirmed: event.target.checked })}
                    />
                  </label>
                </div>
                <div className="two-column">
                  <label>
                    <span>Delivery location</span>
                    <input value={line.deliveryLocation} onChange={(event) => updateReviewLine(line.id, { deliveryLocation: event.target.value })} />
                  </label>
                  <label>
                    <span>Budget category</span>
                    <select value={line.budgetCategoryId} onChange={(event) => updateReviewLine(line.id, { budgetCategoryId: event.target.value })}>
                      <option value="">Unassigned</option>
                      {budgetCategories?.map((category) => (
                        <option key={category.id} value={category.id}>
                          {category.name}
                        </option>
                      ))}
                    </select>
                  </label>
                </div>
                <div className="two-column">
                  <label>
                    <span>Accounting category</span>
                    <input value={line.accountingCategory} onChange={(event) => updateReviewLine(line.id, { accountingCategory: event.target.value })} />
                  </label>
                  <label>
                    <span>Supplier contact</span>
                    <input value={line.supplierContact} onChange={(event) => updateReviewLine(line.id, { supplierContact: event.target.value })} />
                  </label>
                </div>
                {!line.itemId ? (
                  <div className="stack-form">
                    <label>
                      <span>Canonical item number</span>
                      <input value={registerDraft.canonicalItemNumber} onChange={(event) => updateRegisterDraft(line.id, { canonicalItemNumber: event.target.value })} />
                    </label>
                    <label>
                      <span>Description</span>
                      <input value={registerDraft.description} onChange={(event) => updateRegisterDraft(line.id, { description: event.target.value })} />
                    </label>
                    <label>
                      <span>Manufacturer</span>
                      <input value={registerDraft.manufacturerName} onChange={(event) => updateRegisterDraft(line.id, { manufacturerName: event.target.value })} />
                    </label>
                    <label>
                      <span>Category</span>
                      <select
                        value={registerDraft.categoryKey}
                        onChange={(event) =>
                          updateRegisterDraft(line.id, {
                            categoryKey: event.target.value,
                            categoryName: categoryOptions.find((category) => category.key === event.target.value)?.name || event.target.value,
                          })
                        }
                      >
                        {categoryOptions.map((category) => (
                          <option key={category.key} value={category.key}>
                            {category.name}
                          </option>
                        ))}
                      </select>
                    </label>
                    <label>
                      <span>Supplier alias</span>
                      <input value={registerDraft.supplierAliasNumber} onChange={(event) => updateRegisterDraft(line.id, { supplierAliasNumber: event.target.value })} />
                    </label>
                    <label>
                      <span>Units per order</span>
                      <input
                        type="number"
                        min={1}
                        value={registerDraft.unitsPerOrder}
                        onChange={(event) => updateRegisterDraft(line.id, { unitsPerOrder: Number(event.target.value) || 1 })}
                      />
                    </label>
                    <button type="button" className="primary-button" disabled={busyLineId === line.id} onClick={() => handleRegister(line.id)}>
                      Register as New Item
                    </button>
                  </div>
                ) : (
                  <div>Resolved to {line.itemId}</div>
                )}
              </div>
            )
          })}
        </div>
      </SectionCard>
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
    categoryName: 'misc',
    defaultSupplierId: supplierId,
    supplierAliasNumber: line.itemNumber,
    unitsPerOrder: 1,
  }
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

function buildItemOptions(line: OCRResultLine) {
  const seen = new Set<string>()
  const options = []
  if (line.itemId) {
    options.push({ value: line.itemId, label: `Current / ${line.itemId}` })
    seen.add(line.itemId)
  }
  for (const candidate of line.matchCandidates) {
    if (seen.has(candidate.itemId)) {
      continue
    }
    seen.add(candidate.itemId)
    options.push({
      value: candidate.itemId,
      label: `${candidate.canonicalItemNumber} (${Math.round(candidate.score * 100)}%)`,
    })
  }
  return options
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
