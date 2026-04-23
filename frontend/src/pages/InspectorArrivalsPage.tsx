import { useState } from 'react'
import { useSWRConfig } from 'swr'
import { useProcurementRequests } from '@/hooks/useProcurementRequests'
import { sendMockProcurementWebhook } from '@/lib/mockApi'
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card'
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from '@/components/ui/table'
import { Badge } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
  DialogTrigger,
} from '@/components/ui/dialog'
import { AlertCircle, CheckCircle } from 'lucide-react'

export function InspectorArrivalsPage() {
  const { data } = useProcurementRequests()
  const { mutate } = useSWRConfig()
  const [processingIds, setProcessingIds] = useState<Set<string>>(new Set())
  const [openDialogs, setOpenDialogs] = useState<Set<string>>(new Set())

  const arrivalRows =
    data?.rows.filter((row) => ['submitted', 'ordered', 'partially_received'].includes(row.normalizedStatus)) ?? []

  const handleMarkReceived = async (row: (typeof arrivalRows)[0]) => {
    setProcessingIds((prev) => new Set(prev).add(row.id))
    try {
      await sendMockProcurementWebhook({
        eventType: 'procurement.status_changed',
        requestId: row.id,
        normalizedStatus: 'received',
        rawStatus: 'external_receipt_completed',
      })
      await Promise.all([
        mutate('procurement-requests'),
        mutate(['procurement-request-detail', row.id]),
        mutate('procurement-webhook-events'),
      ])
      setOpenDialogs((prev) => {
        const next = new Set(prev)
        next.delete(row.id)
        return next
      })
    } finally {
      setProcessingIds((prev) => {
        const next = new Set(prev)
        next.delete(row.id)
        return next
      })
    }
  }

  const getStatusColor = (status: string) => {
    switch (status) {
      case 'submitted':
        return 'default'
      case 'ordered':
        return 'secondary'
      case 'partially_received':
        return 'outline'
      default:
        return 'default'
    }
  }

  const getStatusLabel = (status: string) => {
    return status.replace(/_/g, ' ').charAt(0).toUpperCase() + status.slice(1).replace(/_/g, ' ')
  }

  return (
    <div className="space-y-6 p-6">
      <div>
        <h1 className="text-3xl font-bold tracking-tight">Arrivals & Inspection</h1>
        <p className="text-muted-foreground mt-2">
          Review inbound procurement requests and confirm receipt of items.
        </p>
      </div>

      <Card>
        <CardHeader>
          <CardTitle>Pending Arrival Confirmations</CardTitle>
          <CardDescription>
            Requests awaiting receiving inspection confirmation. Only visible to receiving inspectors.
          </CardDescription>
        </CardHeader>
        <CardContent>
          {arrivalRows.length === 0 ? (
            <div className="text-center py-12">
              <CheckCircle className="h-12 w-12 text-muted-foreground mx-auto mb-4 opacity-50" />
              <p className="text-muted-foreground">No pending arrivals at this time.</p>
            </div>
          ) : (
            <div className="overflow-x-auto">
              <Table>
                <TableHeader>
                  <TableRow>
                    <TableHead>Batch Number</TableHead>
                    <TableHead>Supplier</TableHead>
                    <TableHead>Status</TableHead>
                    <TableHead className="text-right">Items</TableHead>
                    <TableHead className="w-[150px] text-right">Action</TableHead>
                  </TableRow>
                </TableHeader>
                <TableBody>
                  {arrivalRows.map((row) => (
                    <TableRow key={row.id}>
                      <TableCell className="font-medium">{row.batchNumber}</TableCell>
                      <TableCell>{row.supplierName}</TableCell>
                      <TableCell>
                        <Badge variant={getStatusColor(row.normalizedStatus)}>
                          {getStatusLabel(row.normalizedStatus)}
                        </Badge>
                      </TableCell>
                      <TableCell className="text-right">{row.requestedItems}</TableCell>
                      <TableCell className="text-right">
                        <Dialog
                          open={openDialogs.has(row.id)}
                          onOpenChange={(open) => {
                            setOpenDialogs((prev) => {
                              const next = new Set(prev)
                              if (open) {
                                next.add(row.id)
                              } else {
                                next.delete(row.id)
                              }
                              return next
                            })
                          }}
                        >
                          <DialogTrigger asChild>
                            <Button
                              size="sm"
                              variant="default"
                              disabled={processingIds.has(row.id)}
                            >
                              {processingIds.has(row.id) ? 'Processing...' : 'Mark Received'}
                            </Button>
                          </DialogTrigger>
                          <DialogContent>
                            <DialogHeader>
                              <DialogTitle>Confirm Receipt</DialogTitle>
                              <DialogDescription>
                                Are you ready to mark this procurement request as received? This action will update the inventory system.
                              </DialogDescription>
                            </DialogHeader>
                            <div className="space-y-3 py-4">
                              <div className="rounded-lg bg-muted p-4">
                                <p className="text-sm">
                                  <span className="text-muted-foreground">Batch: </span>
                                  <span className="font-semibold">{row.batchNumber}</span>
                                </p>
                                <p className="text-sm mt-1">
                                  <span className="text-muted-foreground">Supplier: </span>
                                  <span className="font-semibold">{row.supplierName}</span>
                                </p>
                                <p className="text-sm mt-1">
                                  <span className="text-muted-foreground">Items: </span>
                                  <span className="font-semibold">{row.requestedItems}</span>
                                </p>
                              </div>
                              <div className="flex gap-2 text-sm bg-amber-50 border border-amber-200 rounded-lg p-3">
                                <AlertCircle className="h-4 w-4 text-amber-600 flex-shrink-0 mt-0.5" />
                                <p className="text-amber-800">
                                  Ensure all items have been physically verified before confirming receipt.
                                </p>
                              </div>
                            </div>
                            <DialogFooter>
                              <Button
                                type="button"
                                variant="outline"
                                onClick={() => {
                                  setOpenDialogs((prev) => {
                                    const next = new Set(prev)
                                    next.delete(row.id)
                                    return next
                                  })
                                }}
                              >
                                Cancel
                              </Button>
                              <Button
                                type="button"
                                onClick={() => handleMarkReceived(row)}
                                disabled={processingIds.has(row.id)}
                              >
                                {processingIds.has(row.id) ? 'Processing...' : 'Confirm Receipt'}
                              </Button>
                            </DialogFooter>
                          </DialogContent>
                        </Dialog>
                      </TableCell>
                    </TableRow>
                  ))}
                </TableBody>
              </Table>
            </div>
          )}
        </CardContent>
      </Card>
    </div>
  )
}
