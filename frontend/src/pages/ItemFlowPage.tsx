import { useParams } from 'react-router-dom'
import { Card, CardContent, CardHeader, CardTitle, CardDescription } from '../components/ui/card'
import { Badge } from '../components/ui/badge'
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from '../components/ui/table'
import { useItemFlow } from '../hooks/useItemFlow'

const eventTypeLabels: Record<string, string> = {
  receive: 'Receive',
  move: 'Move',
  consume: 'Consume',
  adjust: 'Adjust',
  reserve_allocate: 'Reserve',
  reserve_release: 'Release',
  undo: 'Undo',
  reverse: 'Reverse',
}

const eventTypeVariants: Record<string, 'default' | 'secondary' | 'destructive' | 'outline'> = {
  receive: 'default',
  move: 'outline',
  consume: 'destructive',
  adjust: 'secondary',
  reserve_allocate: 'destructive',
  reserve_release: 'default',
  undo: 'secondary',
  reverse: 'secondary',
}

export function ItemFlowPage() {
  const { id } = useParams<{ id: string }>()
  const { data, isLoading } = useItemFlow(id)

  return (
    <div className="space-y-6 p-6">
      <div className="space-y-2">
        <h1 className="text-3xl font-bold">Item Flow</h1>
        <p className="text-muted-foreground">
          Chronological inventory movement history
          {data?.itemNumber && ` for ${data.itemNumber}`}.
        </p>
      </div>

      <Card>
        <CardHeader>
          <CardTitle>{data?.itemNumber ?? id}</CardTitle>
          <CardDescription>
            {data?.rows.length ?? 0} events
            {data?.rows.length
              ? ` | Current balance: ${data.rows[data.rows.length - 1].runningBalance}`
              : ''}
          </CardDescription>
        </CardHeader>
        <CardContent>
          {isLoading ? (
            <p className="text-center text-muted-foreground py-8">Loading...</p>
          ) : (
            <div className="overflow-x-auto">
              <Table>
                <TableHeader>
                  <TableRow>
                    <TableHead>Date</TableHead>
                    <TableHead>Event</TableHead>
                    <TableHead>Location</TableHead>
                    <TableHead className="text-right">Delta</TableHead>
                    <TableHead className="text-right">Balance</TableHead>
                    <TableHead>Source</TableHead>
                    <TableHead>Note</TableHead>
                  </TableRow>
                </TableHeader>
                <TableBody>
                  {!data?.rows.length ? (
                    <TableRow>
                      <TableCell colSpan={7} className="text-center text-muted-foreground py-8">
                        No events found for this item.
                      </TableCell>
                    </TableRow>
                  ) : (
                    data.rows.map((row, i) => (
                      <TableRow key={i}>
                        <TableCell className="text-sm tabular-nums whitespace-nowrap">
                          {new Date(row.date).toLocaleDateString('ja-JP')}
                        </TableCell>
                        <TableCell>
                          <Badge variant={eventTypeVariants[row.eventType] ?? 'outline'}>
                            {eventTypeLabels[row.eventType] ?? row.eventType}
                          </Badge>
                        </TableCell>
                        <TableCell className="text-sm text-muted-foreground">{row.locationCode}</TableCell>
                        <TableCell
                          className={`text-right tabular-nums font-medium ${
                            row.quantityDelta > 0
                              ? 'text-green-600'
                              : row.quantityDelta < 0
                                ? 'text-red-600'
                                : ''
                          }`}
                        >
                          {row.quantityDelta > 0 ? '+' : ''}
                          {row.quantityDelta}
                        </TableCell>
                        <TableCell className="text-right tabular-nums font-semibold">
                          {row.runningBalance}
                        </TableCell>
                        <TableCell className="text-xs text-muted-foreground">
                          {row.sourceType && `${row.sourceType}`}
                          {row.sourceRef && `: ${row.sourceRef}`}
                        </TableCell>
                        <TableCell className="text-xs text-muted-foreground max-w-[12rem] truncate">
                          {row.note}
                        </TableCell>
                      </TableRow>
                    ))
                  )}
                </TableBody>
              </Table>
            </div>
          )}
        </CardContent>
      </Card>
    </div>
  )
}
