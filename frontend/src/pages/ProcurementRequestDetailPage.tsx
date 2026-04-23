import { useState } from 'react'
import { useParams, useNavigate } from 'react-router-dom'
import { useSWRConfig } from 'swr'

import { Button } from '@/components/ui/button'
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card'
import { Badge } from '@/components/ui/badge'
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from '@/components/ui/table'

import { useProcurementRequestDetail } from '@/hooks/useProcurementRequestDetail'
import { useProcurementProjects } from '@/hooks/useProcurementProjects'
import {
  submitProcurementRequest,
  reconcileProcurementRequest,
  sendMockProcurementWebhook,
} from '@/lib/mockApi'

export function ProcurementRequestDetailPage() {
  const { id } = useParams<{ id: string }>()
  const navigate = useNavigate()
  const { mutate } = useSWRConfig()
  const { data: detail } = useProcurementRequestDetail(id || '')
  const { data: projects } = useProcurementProjects()

  const [submitting, setSubmitting] = useState(false)
  const [reconciling, setReconciling] = useState(false)
  const [sendingWebhook, setSendingWebhook] = useState(false)
  const [error, setError] = useState('')
  const [message, setMessage] = useState('')

  if (!id || !detail) {
    return (
      <div className="p-6">
        <p>Loading...</p>
      </div>
    )
  }

  async function handleDispatch() {
    if (!detail) return
    setError('')
    setMessage('')
    setSubmitting(true)
    try {
      await submitProcurementRequest(detail.id)
      await Promise.all([mutate('procurement-requests'), mutate(['procurement-request-detail', detail.id])])
      setMessage('Request submitted successfully')
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to submit procurement request')
    } finally {
      setSubmitting(false)
    }
  }

  async function handleReconcile() {
    if (!detail) return
    setError('')
    setMessage('')
    setReconciling(true)
    try {
      const result = await reconcileProcurementRequest(detail.id)
      await Promise.all([mutate('procurement-requests'), mutate(['procurement-request-detail', detail.id])])
      setMessage(`Reconciled to ${result.normalizedStatus} at ${result.lastReconciledAt}`)
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to reconcile procurement request')
    } finally {
      setReconciling(false)
    }
  }

  async function handleSendWebhook(eventType: 'procurement.status_changed' | 'master.projects_changed' | 'master.budget_categories_changed') {
    if (!detail) return
    setError('')
    setMessage('')
    setSendingWebhook(true)
    try {
      const result = await sendMockProcurementWebhook({
        eventType,
        requestId: detail.id,
        projectKey: projects?.[0]?.key ?? '',
      })
      await Promise.all([
        mutate('procurement-requests'),
        mutate(['procurement-request-detail', detail.id]),
        mutate('procurement-sync-runs'),
        mutate('procurement-webhook-events'),
      ])
      setMessage(`Processed ${result.eventType} at ${result.syncedAt}`)
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to process webhook')
    } finally {
      setSendingWebhook(false)
    }
  }

  const getStatusVariant = (status: string): 'default' | 'secondary' | 'destructive' | 'outline' => {
    switch (status) {
      case 'draft':
        return 'secondary'
      case 'submitted':
        return 'default'
      case 'ordered':
        return 'default'
      case 'partially_received':
        return 'outline'
      case 'received':
        return 'default'
      default:
        return 'default'
    }
  }

  return (
    <div className="p-6 space-y-6">
      <div className="flex items-center justify-between">
        <div>
          <Button variant="ghost" onClick={() => navigate(-1)} className="mb-4">
            ← Back
          </Button>
          <h1 className="text-3xl font-bold tracking-tight">{detail.title}</h1>
          <p className="text-muted-foreground mt-1">Batch {detail.batchNumber}</p>
        </div>
        <Badge variant={getStatusVariant(detail.normalizedStatus)} className="text-base px-3 py-1">
          {detail.normalizedStatus}
        </Badge>
      </div>

      <div className="grid grid-cols-2 gap-4">
        <Card>
          <CardHeader className="pb-3">
            <CardTitle className="text-sm font-medium">Quotation Number</CardTitle>
          </CardHeader>
          <CardContent>
            <p className="text-2xl font-bold">{detail.quotationNumber}</p>
          </CardContent>
        </Card>
        <Card>
          <CardHeader className="pb-3">
            <CardTitle className="text-sm font-medium">Supplier</CardTitle>
          </CardHeader>
          <CardContent>
            <p className="text-2xl font-bold">{detail.supplierName || '-'}</p>
          </CardContent>
        </Card>
        <Card>
          <CardHeader className="pb-3">
            <CardTitle className="text-sm font-medium">Project</CardTitle>
          </CardHeader>
          <CardContent>
            <p className="text-2xl font-bold">{detail.projectName || '-'}</p>
          </CardContent>
        </Card>
        <Card>
          <CardHeader className="pb-3">
            <CardTitle className="text-sm font-medium">Dispatch Status</CardTitle>
          </CardHeader>
          <CardContent>
            <p className="text-2xl font-bold">{detail.dispatchStatus}</p>
          </CardContent>
        </Card>
      </div>

      <div className="flex gap-2">
        <Button
          onClick={handleDispatch}
          disabled={submitting || detail.dispatchStatus === 'submitted'}
        >
          {submitting ? 'Submitting...' : detail.dispatchStatus === 'submitted' ? 'Submitted' : 'Submit to External Flow'}
        </Button>
        <Button
          variant="outline"
          onClick={handleReconcile}
          disabled={reconciling || !detail.externalRequestReference}
        >
          {reconciling ? 'Reconciling...' : 'Reconcile'}
        </Button>
      </div>

      <Card>
        <CardHeader>
          <CardTitle>Details</CardTitle>
        </CardHeader>
        <CardContent className="space-y-4">
          <div className="grid grid-cols-2 gap-4">
            <div>
              <p className="text-sm text-muted-foreground">Normalized Status</p>
              <p className="font-medium">{detail.normalizedStatus}</p>
            </div>
            <div>
              <p className="text-sm text-muted-foreground">Raw Status</p>
              <p className="font-medium">{detail.rawStatus}</p>
            </div>
            <div>
              <p className="text-sm text-muted-foreground">Quantity Progression</p>
              <p className="font-medium">{detail.quantityProgression}</p>
            </div>
            <div>
              <p className="text-sm text-muted-foreground">External Reference</p>
              <p className="font-medium">{detail.externalRequestReference || 'Not submitted'}</p>
            </div>
            <div>
              <p className="text-sm text-muted-foreground">Dispatch Attempts</p>
              <p className="font-medium">{detail.dispatchAttempts}</p>
            </div>
            <div>
              <p className="text-sm text-muted-foreground">Last Reconciled</p>
              <p className="font-medium">{detail.lastReconciledAt || 'Not yet reconciled'}</p>
            </div>
            <div>
              <p className="text-sm text-muted-foreground">Sync Source</p>
              <p className="font-medium">{detail.syncSource || 'Not yet synced'}</p>
            </div>
            <div>
              <p className="text-sm text-muted-foreground">Artifact Cleanup</p>
              <p className="font-medium">{detail.artifactDeleteStatus}</p>
            </div>
          </div>
        </CardContent>
      </Card>

      <Card>
        <CardHeader>
          <CardTitle>Lines</CardTitle>
          <CardDescription>{detail.lines.length} items in this request</CardDescription>
        </CardHeader>
        <CardContent>
          <div className="border rounded-lg overflow-hidden">
            <Table>
              <TableHeader>
                <TableRow>
                  <TableHead>Item</TableHead>
                  <TableHead>Description</TableHead>
                  <TableHead>Qty</TableHead>
                  <TableHead>Delivery Location</TableHead>
                  <TableHead>Lead Time</TableHead>
                </TableRow>
              </TableHeader>
              <TableBody>
                {detail.lines.map((line) => (
                  <TableRow key={line.id}>
                    <TableCell className="font-mono text-sm">{line.itemNumber}</TableCell>
                    <TableCell>{line.description}</TableCell>
                    <TableCell>{line.requestedQuantity}</TableCell>
                    <TableCell className="text-sm text-muted-foreground">{line.deliveryLocation}</TableCell>
                    <TableCell className="text-sm text-muted-foreground">{line.leadTimeDays} days</TableCell>
                  </TableRow>
                ))}
              </TableBody>
            </Table>
          </div>
        </CardContent>
      </Card>

      <Card>
        <CardHeader>
          <CardTitle>Status History</CardTitle>
        </CardHeader>
        <CardContent>
          <div className="space-y-2">
            {detail.statusHistory.map((entry) => (
              <div key={entry.id} className="flex justify-between text-sm py-2 border-b last:border-b-0">
                <div>
                  <Badge variant="secondary" className="mr-2">{entry.normalizedStatus}</Badge>
                  <span className="text-muted-foreground">({entry.rawStatus})</span>
                </div>
                <span className="text-muted-foreground">{entry.observedAt}</span>
              </div>
            ))}
          </div>
        </CardContent>
      </Card>

      <Card>
        <CardHeader>
          <CardTitle>Dispatch History</CardTitle>
        </CardHeader>
        <CardContent>
          <div className="space-y-2">
            {detail.dispatchHistory.map((entry) => (
              <div key={entry.id} className="flex justify-between text-sm py-2 border-b last:border-b-0">
                <div>
                  <Badge variant="secondary" className="mr-2">{entry.dispatchStatus}</Badge>
                  <span className="text-muted-foreground">{entry.externalRequestReference || 'No external ref'}</span>
                  {entry.errorMessage && <span className="text-destructive ml-2">{entry.errorMessage}</span>}
                </div>
                <span className="text-muted-foreground">{entry.observedAt}</span>
              </div>
            ))}
          </div>
        </CardContent>
      </Card>

      <Card>
        <CardHeader>
          <CardTitle>Development Tools</CardTitle>
          <CardDescription>Webhook simulation for testing</CardDescription>
        </CardHeader>
        <CardContent className="space-y-2">
          <Button
            variant="outline"
            onClick={() => handleSendWebhook('procurement.status_changed')}
            disabled={sendingWebhook || !detail.externalRequestReference}
            className="w-full"
          >
            {sendingWebhook ? 'Processing...' : 'Simulate Status Webhook'}
          </Button>
          <Button
            variant="outline"
            onClick={() => handleSendWebhook('master.projects_changed')}
            disabled={sendingWebhook}
            className="w-full"
          >
            Simulate Project Webhook
          </Button>
          <Button
            variant="outline"
            onClick={() => handleSendWebhook('master.budget_categories_changed')}
            disabled={sendingWebhook}
            className="w-full"
          >
            Simulate Budget Webhook
          </Button>
        </CardContent>
      </Card>

      {error && (
        <Card className="border-destructive">
          <CardContent className="pt-6">
            <p className="text-destructive">{error}</p>
          </CardContent>
        </Card>
      )}

      {message && (
        <Card className="border-green-500">
          <CardContent className="pt-6">
            <p className="text-green-600">{message}</p>
          </CardContent>
        </Card>
      )}
    </div>
  )
}
