import type { ChangeEvent } from 'react'
import { useRef, useState } from 'react'
import { useSWRConfig } from 'swr'

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
import { Label } from '@/components/ui/label'
import { useImports } from '@/hooks/useImports'
import { importMasterDataCSV } from '@/lib/mockApi'
import { Upload, FileUp } from 'lucide-react'

type OperatorImportsPageProps = {
  mode?: 'upload' | 'history'
}

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

export function OperatorImportsPage({ mode = 'upload' }: OperatorImportsPageProps) {
  const { data } = useImports()
  const { mutate } = useSWRConfig()
  const [message, setMessage] = useState('')
  const [isUploading, setIsUploading] = useState(false)
  const fileInputRef = useRef<HTMLInputElement>(null)

  async function handleUpload(event: ChangeEvent<HTMLInputElement>) {
    const file = event.target.files?.[0]
    if (!file) {
      return
    }
    setIsUploading(true)
    try {
      await importMasterDataCSV('items', file)
      await mutate('imports')
      setMessage(`Staged import ${file.name}`)
      setTimeout(() => setMessage(''), 4000)
    } finally {
      setIsUploading(false)
      if (fileInputRef.current) {
        fileInputRef.current.value = ''
      }
    }
  }

  return (
    <div className="space-y-6 p-6">
      <div className="space-y-2">
        <h1 className="text-3xl font-bold tracking-tight">Imports</h1>
        <p className="text-muted-foreground">
          Upload and manage CSV imports for master data
        </p>
      </div>

      {mode === 'upload' && (
        <Card className="border-dashed border-2 border-primary/30">
          <CardHeader>
            <CardTitle className="flex items-center gap-2">
              <Upload className="w-5 h-5" />
              Upload CSV
            </CardTitle>
            <CardDescription>
              Upload CSV files to update master data.
            </CardDescription>
          </CardHeader>
          <CardContent className="space-y-4">
            <div className="space-y-2">
              <Label htmlFor="csv-upload">Select CSV File</Label>
              <div className="flex gap-2">
                <input
                  ref={fileInputRef}
                  id="csv-upload"
                  type="file"
                  accept=".csv,text/csv"
                  onChange={(event) => void handleUpload(event)}
                  disabled={isUploading}
                  className="hidden"
                />
                <Button
                  variant="outline"
                  onClick={() => fileInputRef.current?.click()}
                  disabled={isUploading}
                  className="gap-2"
                >
                  <FileUp className="w-4 h-4" />
                  {isUploading ? 'Uploading...' : 'Choose File'}
                </Button>
              </div>
              <p className="text-xs text-muted-foreground">
                Uploaded files are checked before they are added to import history.
              </p>
            </div>
            {message && (
              <div className="p-3 bg-green-50 border border-green-200 text-green-800 text-sm rounded">
                {message}
              </div>
            )}
          </CardContent>
        </Card>
      )}

      <Card>
        <CardHeader>
          <CardTitle>Import History</CardTitle>
          <CardDescription>Review recent CSV uploads and their results</CardDescription>
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
                  <TableHead className="w-32">Created</TableHead>
                </TableRow>
              </TableHeader>
              <TableBody>
                {data?.rows.map((row) => (
                  <TableRow key={row.id}>
                    <TableCell className="text-sm font-medium">{row.importType}</TableCell>
                    <TableCell>
                      <Badge variant={getStatusBadgeVariant(row.status)} className="text-xs">
                        {row.status}
                      </Badge>
                    </TableCell>
                    <TableCell className="text-sm">{row.fileName}</TableCell>
                    <TableCell className="text-sm text-muted-foreground">{row.summary}</TableCell>
                    <TableCell className="text-sm text-muted-foreground">{row.createdAt}</TableCell>
                  </TableRow>
                ))}
              </TableBody>
            </Table>
          </div>
          {data?.rows.length === 0 && (
            <div className="text-center py-8 text-muted-foreground">
              No import history yet.
            </div>
          )}
        </CardContent>
      </Card>
    </div>
  )
}
