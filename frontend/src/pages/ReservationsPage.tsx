import { Card, CardContent } from '@/components/ui/card'
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from '@/components/ui/table'
import { Badge } from '@/components/ui/badge'
import { useReservations } from '@/hooks/useReservations'

function getStatusBadgeVariant(status: string) {
  switch (status.toLowerCase()) {
    case 'reserved':
      return 'default'
    case 'partially_allocated':
      return 'secondary'
    case 'awaiting_stock':
      return 'destructive'
    default:
      return 'outline'
  }
}

export function ReservationsPage() {
  const { data } = useReservations()

  return (
    <div className="space-y-6 p-6">
      <div className="space-y-2">
        <h1 className="text-3xl font-bold tracking-tight">Reservations</h1>
        <p className="text-muted-foreground">
          Local read model contract for reservation visibility.
        </p>
      </div>

      <Card>
        <CardContent className="pt-6">
          <div className="overflow-x-auto">
            <Table>
              <TableHeader>
                <TableRow>
                  <TableHead className="w-16">ID</TableHead>
                  <TableHead>Item</TableHead>
                  <TableHead>Description</TableHead>
                  <TableHead className="w-16">Qty</TableHead>
                  <TableHead>Device</TableHead>
                  <TableHead>Scope</TableHead>
                  <TableHead className="w-28">Status</TableHead>
                </TableRow>
              </TableHeader>
              <TableBody>
                {data?.rows.map((row) => (
                  <TableRow key={row.id}>
                    <TableCell className="font-medium text-sm">{row.id}</TableCell>
                    <TableCell className="text-sm">{row.itemNumber}</TableCell>
                    <TableCell className="text-sm text-muted-foreground">{row.description}</TableCell>
                    <TableCell className="text-sm text-right">{row.quantity}</TableCell>
                    <TableCell className="text-sm">{row.device}</TableCell>
                    <TableCell className="text-sm">{row.scope}</TableCell>
                    <TableCell>
                      <Badge variant={getStatusBadgeVariant(row.status)} className="text-xs">
                        {row.status.replace(/_/g, ' ')}
                      </Badge>
                    </TableCell>
                  </TableRow>
                ))}
              </TableBody>
            </Table>
          </div>
          {data?.rows.length === 0 && (
            <div className="text-center py-8 text-muted-foreground">
              No reservations found.
            </div>
          )}
        </CardContent>
      </Card>
    </div>
  )
}
