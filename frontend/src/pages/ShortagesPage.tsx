import { useSearchParams } from 'react-router-dom'
import { useState } from 'react'

import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card'
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from '@/components/ui/table'
import { Badge } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'
import { useImports } from '@/hooks/useImports'
import { useShortages } from '@/hooks/useShortages'
import { exportShortagesCSV } from '@/lib/mockApi'
import { Download } from 'lucide-react'

function getStatusBadgeVariant(status: string) {
  switch (status.toLowerCase()) {
    case 'completed':
      return 'default'
    case 'pending':
      return 'secondary'
    case 'failed':
      return 'destructive'
    default:
      return 'outline'
  }
}

export function ShortagesPage() {
  const [searchParams] = useSearchParams()
  const device = searchParams.get('device') ?? ''
  const scope = searchParams.get('scope') ?? ''
  const { data: shortages } = useShortages(device, scope)
  const { data: imports } = useImports()
  const [exportMessage, setExportMessage] = useState('')
  const [isExporting, setIsExporting] = useState(false)

  async function handleExport() {
    setIsExporting(true)
    try {
      const csv = await exportShortagesCSV(device, scope)
      const blob = new Blob([csv], { type: 'text/csv;charset=utf-8' })
      const url = URL.createObjectURL(blob)
      const link = document.createElement('a')
      link.href = url
      link.download = `shortages${device ? `-${device}` : ''}${scope ? `-${scope}` : ''}.csv`
      link.click()
      URL.revokeObjectURL(url)
      setExportMessage('CSV exported successfully.')
      setTimeout(() => setExportMessage(''), 3000)
    } finally {
      setIsExporting(false)
    }
  }

  return (
    <div className="space-y-6 p-6">
      <div className="space-y-2">
        <h1 className="text-3xl font-bold tracking-tight">Shortages</h1>
        <p className="text-muted-foreground">
          Shortage summary {device || scope ? `filtered by Device: ${device}, Scope: ${scope}` : 'for all contexts'}.
        </p>
      </div>

      <Card>
        <CardHeader className="flex flex-row items-center justify-between space-y-0">
          <div className="space-y-1">
            <CardTitle>Shortage List</CardTitle>
            <CardDescription>Items that need replenishment</CardDescription>
          </div>
          <Button
            onClick={() => void handleExport()}
            disabled={isExporting}
            size="sm"
            variant="outline"
            className="gap-2"
          >
            <Download className="w-4 h-4" />
            {isExporting ? 'Exporting...' : 'Export CSV'}
          </Button>
        </CardHeader>
        <CardContent>
          {exportMessage && (
            <div className="mb-4 p-3 bg-green-50 border border-green-200 text-green-800 text-sm rounded">
              {exportMessage}
            </div>
          )}
          <div className="overflow-x-auto">
            <Table>
              <TableHeader>
                <TableRow>
                  <TableHead>Device</TableHead>
                  <TableHead>Scope</TableHead>
                  <TableHead>Manufacturer</TableHead>
                  <TableHead>Item</TableHead>
                  <TableHead>Description</TableHead>
                  <TableHead className="w-20 text-right">Short Qty</TableHead>
                </TableRow>
              </TableHeader>
              <TableBody>
                {shortages?.rows.map((row) => (
                  <TableRow key={`${row.device}-${row.scope}-${row.itemNumber}`}>
                    <TableCell className="font-medium text-sm">{row.device}</TableCell>
                    <TableCell className="text-sm">{row.scope}</TableCell>
                    <TableCell className="text-sm">{row.manufacturer}</TableCell>
                    <TableCell className="text-sm">{row.itemNumber}</TableCell>
                    <TableCell className="text-sm text-muted-foreground">{row.description}</TableCell>
                    <TableCell className="text-right">
                      <Badge variant="destructive" className="text-xs">
                        {row.quantity}
                      </Badge>
                    </TableCell>
                  </TableRow>
                ))}
              </TableBody>
            </Table>
          </div>
          {shortages?.rows.length === 0 && (
            <div className="text-center py-8 text-muted-foreground">
              No shortages found.
            </div>
          )}
        </CardContent>
      </Card>

      <Card>
        <CardHeader>
          <CardTitle>Import History</CardTitle>
          <CardDescription>Recent item and alias imports</CardDescription>
        </CardHeader>
        <CardContent>
          <div className="overflow-x-auto">
            <Table>
              <TableHeader>
                <TableRow>
                  <TableHead className="w-16">ID</TableHead>
                  <TableHead>Type</TableHead>
                  <TableHead className="w-28">Status</TableHead>
                  <TableHead>File</TableHead>
                  <TableHead className="max-w-xs">Summary</TableHead>
                </TableRow>
              </TableHeader>
              <TableBody>
                {imports?.rows.map((row) => (
                  <TableRow key={row.id}>
                    <TableCell className="font-medium text-sm">{row.id}</TableCell>
                    <TableCell className="text-sm">{row.importType}</TableCell>
                    <TableCell>
                      <Badge variant={getStatusBadgeVariant(row.status)} className="text-xs">
                        {row.status}
                      </Badge>
                    </TableCell>
                    <TableCell className="text-sm">{row.fileName}</TableCell>
                    <TableCell className="text-sm text-muted-foreground">{row.summary}</TableCell>
                  </TableRow>
                ))}
              </TableBody>
            </Table>
          </div>
          {imports?.rows.length === 0 && (
            <div className="text-center py-8 text-muted-foreground">
              No import history.
            </div>
          )}
        </CardContent>
      </Card>
    </div>
  )
}
