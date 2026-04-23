import { useState } from 'react'
import type { FormEvent } from 'react'
import { useSearchParams } from 'react-router-dom'
import { useSWRConfig } from 'swr'

import { SectionCard } from '../components/SectionCard'
import { useProcurementBudgetCategories } from '../hooks/useProcurementBudgetCategories'
import { useProcurementProjects } from '../hooks/useProcurementProjects'
import { useProcurementRequestDetail } from '../hooks/useProcurementRequestDetail'
import { useProcurementRequests } from '../hooks/useProcurementRequests'
import { useProcurementSyncRuns } from '../hooks/useProcurementSyncRuns'
import { useProcurementWebhookEvents } from '../hooks/useProcurementWebhookEvents'
import {
  createProcurementRequest,
  reconcileProcurementRequest,
  refreshProcurementBudgetCategories,
  refreshProcurementProjects,
  sendMockProcurementWebhook,
  submitProcurementRequest,
} from '../lib/mockApi'

export function ProcurementPage() {
  const [searchParams, setSearchParams] = useSearchParams()
  const { mutate } = useSWRConfig()
  const { data: requests } = useProcurementRequests()
  const { data: projects } = useProcurementProjects()
  const { data: syncRuns } = useProcurementSyncRuns()
  const { data: webhookEvents } = useProcurementWebhookEvents()
  const requestedId = searchParams.get('requestId') ?? ''
  const selectedId = requests?.rows.some((row) => row.id === requestedId) ? requestedId : requests?.rows[0]?.id || requestedId || 'batch-001'
  const { data: detail } = useProcurementRequestDetail(selectedId)

  const [title, setTitle] = useState('New shortage follow-up')
  const [projectId, setProjectId] = useState('proj-er2-upgrade')
  const [budgetCategoryId, setBudgetCategoryId] = useState('budget-er2-material')
  const { data: budgetCategories } = useProcurementBudgetCategories(projectId)
  const selectedBudgetCategoryId =
    budgetCategories?.some((budget) => budget.id === budgetCategoryId) ? budgetCategoryId : budgetCategories?.[0]?.id || budgetCategoryId
  const [submitting, setSubmitting] = useState(false)
  const [reconciling, setReconciling] = useState(false)
  const [refreshingProjects, setRefreshingProjects] = useState(false)
  const [refreshingBudgets, setRefreshingBudgets] = useState(false)
  const [submitError, setSubmitError] = useState('')
  const [syncMessage, setSyncMessage] = useState('')
  const [sendingWebhook, setSendingWebhook] = useState(false)

  async function handleSubmit(event: FormEvent<HTMLFormElement>) {
    event.preventDefault()
    const result = await createProcurementRequest({
      title,
      projectId,
      budgetCategoryId: selectedBudgetCategoryId,
      supplierId: 'sup-misumi',
      quotationId: 'quote-001',
      sourceType: 'manual',
      createdBy: 'local-user',
      lines: [
        {
          itemId: 'item-er2',
          quotationLineId: 'quote-line-001',
          requestedQuantity: 4,
          deliveryLocation: 'Tokyo Assembly',
          accountingCategory: 'parts',
          note: 'Created from Procurement page form',
        },
      ],
    })
    await mutate('procurement-requests')
    if ((result as { id?: string } | undefined)?.id) {
      selectRequest((result as { id: string }).id)
    }
  }

  async function handleDispatch() {
    if (!detail) {
      return
    }
    setSubmitError('')
    setSyncMessage('')
    setSubmitting(true)
    try {
      await submitProcurementRequest(detail.id)
      await Promise.all([mutate('procurement-requests'), mutate(['procurement-request-detail', detail.id])])
    } catch (error) {
      setSubmitError(error instanceof Error ? error.message : 'Failed to submit procurement request')
    } finally {
      setSubmitting(false)
    }
  }

  async function handleReconcile() {
    if (!detail) {
      return
    }
    setSubmitError('')
    setSyncMessage('')
    setReconciling(true)
    try {
      const result = await reconcileProcurementRequest(detail.id)
      await Promise.all([mutate('procurement-requests'), mutate(['procurement-request-detail', detail.id])])
      setSyncMessage(`Reconciled to ${result.normalizedStatus} at ${result.lastReconciledAt}.`)
    } catch (error) {
      setSubmitError(error instanceof Error ? error.message : 'Failed to reconcile procurement request')
    } finally {
      setReconciling(false)
    }
  }

  async function handleRefreshProjects() {
    setSubmitError('')
    setSyncMessage('')
    setRefreshingProjects(true)
    try {
      const result = await refreshProcurementProjects()
      await mutate('procurement-projects')
      setSyncMessage(`Refreshed ${result.rowCount} project rows at ${result.syncedAt}.`)
    } catch (error) {
      setSubmitError(error instanceof Error ? error.message : 'Failed to refresh project cache')
    } finally {
      setRefreshingProjects(false)
    }
  }

  async function handleRefreshBudgets() {
    setSubmitError('')
    setSyncMessage('')
    setRefreshingBudgets(true)
    try {
      const result = await refreshProcurementBudgetCategories(projectId)
      await mutate(['procurement-budget-categories', projectId])
      setSyncMessage(`Refreshed ${result.rowCount} budget categories at ${result.syncedAt}.`)
    } catch (error) {
      setSubmitError(error instanceof Error ? error.message : 'Failed to refresh budget categories')
    } finally {
      setRefreshingBudgets(false)
    }
  }

  async function handleSendWebhook(eventType: 'procurement.status_changed' | 'master.projects_changed' | 'master.budget_categories_changed') {
    setSubmitError('')
    setSyncMessage('')
    setSendingWebhook(true)
    try {
      const result = await sendMockProcurementWebhook({
        eventType,
        requestId: selectedId,
        projectKey: projects?.find((project) => project.id === projectId)?.key ?? projects?.[0]?.key ?? '',
      })
      await Promise.all([
        mutate('procurement-requests'),
        mutate(['procurement-request-detail', selectedId]),
        mutate('procurement-sync-runs'),
        mutate('procurement-webhook-events'),
        mutate('procurement-projects'),
        mutate(['procurement-budget-categories', projectId]),
      ])
      setSyncMessage(`Processed ${result.eventType} at ${result.syncedAt}.`)
    } catch (error) {
      setSubmitError(error instanceof Error ? error.message : 'Failed to process webhook')
    } finally {
      setSendingWebhook(false)
    }
  }

  function selectRequest(id: string, replace = false) {
    const next = new URLSearchParams(searchParams)
    next.set('requestId', id)
    setSearchParams(next, { replace })
  }

  return (
    <div className="page-grid">
      <SectionCard title="Procurement Requests" subtitle="Local request list, normalized status, and project context.">
        <table className="data-table">
          <thead>
            <tr>
              <th>Batch</th>
              <th>Title</th>
              <th>Project</th>
              <th>Supplier</th>
              <th>Status</th>
              <th>Source</th>
              <th>Dispatch</th>
            </tr>
          </thead>
          <tbody>
            {requests?.rows.map((row) => (
              <tr key={row.id} onClick={() => selectRequest(row.id)} className={selectedId === row.id ? 'selected-row' : ''}>
                <td>{row.batchNumber}</td>
                <td>{row.title}</td>
                <td>{row.projectName}</td>
                <td>{row.supplierName}</td>
                <td>{row.normalizedStatus}</td>
                <td>{row.sourceType}</td>
                <td>{row.dispatchStatus}</td>
              </tr>
            ))}
          </tbody>
        </table>
      </SectionCard>

      <div className="two-column">
        <SectionCard title="Request Detail" subtitle="Internal tracking stays available before any external adapter is connected.">
          <dl className="definition-list">
            <div>
              <dt>Batch</dt>
              <dd>{detail?.batchNumber}</dd>
            </div>
            <div>
              <dt>Quotation</dt>
              <dd>{detail?.quotationNumber}</dd>
            </div>
            <div>
              <dt>Normalized status</dt>
              <dd>{detail?.normalizedStatus}</dd>
            </div>
            <div>
              <dt>Raw status</dt>
              <dd>{detail?.rawStatus}</dd>
            </div>
            <div>
              <dt>Progression</dt>
              <dd>{detail?.quantityProgression}</dd>
            </div>
            <div>
              <dt>External ref</dt>
              <dd>{detail?.externalRequestReference || 'not submitted'}</dd>
            </div>
            <div>
              <dt>Dispatch</dt>
              <dd>{detail?.dispatchStatus}</dd>
            </div>
            <div>
              <dt>Dispatch attempts</dt>
              <dd>{detail?.dispatchAttempts}</dd>
            </div>
            <div>
              <dt>Last reconciled</dt>
              <dd>{detail?.lastReconciledAt || 'not yet reconciled'}</dd>
            </div>
            <div>
              <dt>Sync source</dt>
              <dd>{detail?.syncSource || 'not yet synced'}</dd>
            </div>
            <div>
              <dt>Artifact cleanup</dt>
              <dd>{detail?.artifactDeleteStatus}</dd>
            </div>
          </dl>
          <div className="two-column">
            <button
              type="button"
              className="primary-button"
              disabled={submitting || detail?.dispatchStatus === 'submitted'}
              onClick={handleDispatch}
            >
              {submitting ? 'Submitting...' : detail?.dispatchStatus === 'submitted' ? 'Submitted' : 'Submit to External Flow'}
            </button>
            <button
              type="button"
              className="primary-button"
              disabled={reconciling || !detail?.externalRequestReference}
              onClick={handleReconcile}
            >
              {reconciling ? 'Reconciling...' : 'Reconcile Projection'}
            </button>
          </div>
          <div className="two-column">
            <span>
              {detail?.dispatchStatus === 'submitted'
                ? `Submitted as ${detail.externalRequestReference || 'local reference pending'}.`
                : 'Uses the local dispatch adapter, persists outbox/history, and deletes the quotation artifact after successful submission.'}
            </span>
            <span>{detail?.syncError || 'Projection stays local-first and can be refreshed without direct frontend-to-external calls.'}</span>
          </div>
          <div className="button-row">
            <button
              type="button"
              className="secondary-button"
              disabled={sendingWebhook || !detail?.externalRequestReference}
              onClick={() => void handleSendWebhook('procurement.status_changed')}
            >
              {sendingWebhook ? 'Processing webhook...' : 'Simulate Status Webhook'}
            </button>
            <button
              type="button"
              className="secondary-button"
              disabled={sendingWebhook}
              onClick={() => void handleSendWebhook('master.projects_changed')}
            >
              Simulate Project Webhook
            </button>
            <button
              type="button"
              className="secondary-button"
              disabled={sendingWebhook}
              onClick={() => void handleSendWebhook('master.budget_categories_changed')}
            >
              Simulate Budget Webhook
            </button>
          </div>
          {submitError ? <p className="muted-copy">{submitError}</p> : null}
          {syncMessage ? <p className="muted-copy">{syncMessage}</p> : null}
          <h3 className="subheading">Lines</h3>
          <table className="data-table">
            <thead>
              <tr>
                <th>Item</th>
                <th>Description</th>
                <th>Qty</th>
                <th>Delivery</th>
                <th>Lead time</th>
              </tr>
            </thead>
            <tbody>
              {detail?.lines.map((line) => (
                <tr key={line.id}>
                  <td>{line.itemNumber}</td>
                  <td>{line.description}</td>
                  <td>{line.requestedQuantity}</td>
                  <td>{line.deliveryLocation}</td>
                  <td>{line.leadTimeDays} days</td>
                </tr>
              ))}
            </tbody>
          </table>
          <h3 className="subheading">Dispatch History</h3>
          <ul className="list">
            {detail?.dispatchHistory.map((entry) => (
              <li key={entry.id}>
                {entry.dispatchStatus} / {entry.externalRequestReference || 'no external ref'} / {entry.observedAt}
                {entry.errorMessage ? ` / ${entry.errorMessage}` : ''}
              </li>
            ))}
          </ul>
          <h3 className="subheading">Status History</h3>
          <ul className="list">
            {detail?.statusHistory.map((entry) => (
              <li key={entry.id}>
                {entry.normalizedStatus} / {entry.rawStatus} / {entry.observedAt}
              </li>
            ))}
          </ul>
        </SectionCard>

        <SectionCard title="Create Request" subtitle="Phase 2 local creation flow with project and budget selection.">
          <form className="stack-form" onSubmit={handleSubmit}>
            <label>
              <span>Title</span>
              <input value={title} onChange={(event) => setTitle(event.target.value)} />
            </label>
            <label>
              <span>Project</span>
                <select
                  value={projectId}
                  onChange={(event) => {
                    setProjectId(event.target.value)
                    setBudgetCategoryId('')
                  }}
                >
                {projects?.map((project) => (
                  <option key={project.id} value={project.id}>
                    {project.name}
                  </option>
                ))}
              </select>
            </label>
            <label>
              <span>Budget category</span>
                <select value={selectedBudgetCategoryId} onChange={(event) => setBudgetCategoryId(event.target.value)}>
                {budgetCategories?.map((budget) => (
                  <option key={budget.id} value={budget.id}>
                    {budget.name}
                  </option>
                ))}
              </select>
            </label>
            <p className="muted-copy">
              Lead time is treated as duration from order date. PDF artifact and structured data stay paired in the internal model.
            </p>
            <div className="two-column">
              <button type="button" className="primary-button" disabled={refreshingProjects} onClick={handleRefreshProjects}>
                {refreshingProjects ? 'Refreshing projects...' : 'Refresh Project Cache'}
              </button>
              <button type="button" className="primary-button" disabled={refreshingBudgets} onClick={handleRefreshBudgets}>
                {refreshingBudgets ? 'Refreshing budgets...' : 'Refresh Budget Categories'}
              </button>
            </div>
            <p className="muted-copy">
              Project cache last synced:{' '}
              {projects?.find((project) => project.id === projectId)?.syncedAt || projects?.[0]?.syncedAt || 'unknown'}
            </p>
            <button type="submit" className="primary-button">
              Create Local Draft
            </button>
          </form>
        </SectionCard>
      </div>

      <SectionCard title="Submission Model" subtitle="Known normalization decisions from the implementation plan.">
        <ul className="list">
          <li>Quotation PDF and structured payload move together.</li>
          <li>Lead time is treated as duration from order date.</li>
          <li>Canonical item and supplier alias stay separate.</li>
        </ul>
      </SectionCard>

      <div className="two-column">
        <SectionCard title="Webhook History" subtitle="Latest received events, processed locally before any direct frontend-to-external coupling.">
          <table className="data-table">
            <thead>
              <tr>
                <th>Event</th>
                <th>External Ref</th>
                <th>Status</th>
                <th>Received</th>
              </tr>
            </thead>
            <tbody>
              {webhookEvents?.map((event) => (
                <tr key={event.id}>
                  <td>{event.eventType}</td>
                  <td>{event.externalRequestReference || event.projectKey || '-'}</td>
                  <td>{event.processingError || 'processed'}</td>
                  <td>{event.receivedAt}</td>
                </tr>
              ))}
            </tbody>
          </table>
        </SectionCard>

        <SectionCard title="Sync Runs" subtitle="Manual refresh and webhook-driven master sync runs are persisted separately from current projections.">
          <table className="data-table">
            <thead>
              <tr>
                <th>Type</th>
                <th>Project</th>
                <th>Rows</th>
                <th>Triggered By</th>
                <th>Finished</th>
              </tr>
            </thead>
            <tbody>
              {syncRuns?.map((run) => (
                <tr key={run.id}>
                  <td>{run.syncType}</td>
                  <td>{run.projectKey || '-'}</td>
                  <td>{run.rowCount}</td>
                  <td>{run.triggeredBy}</td>
                  <td>{run.finishedAt}</td>
                </tr>
              ))}
            </tbody>
          </table>
        </SectionCard>
      </div>
    </div>
  )
}
