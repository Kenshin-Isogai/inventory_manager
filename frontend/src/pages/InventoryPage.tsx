import { useState } from 'react'
import { useInventoryOverview } from '@/hooks/useInventoryOverview'
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card'
import { Input } from '@/components/ui/input'
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from '@/components/ui/table'
import { Badge } from '@/components/ui/badge'

export function InventoryPage() {
  const { data } = useInventoryOverview()
  const [searchTerm, setSearchTerm] = useState('')

  const filteredBalances = data?.balances.filter((row) =>
    row.itemNumber.toLowerCase().includes(searchTerm.toLowerCase()) ||
    row.description.toLowerCase().includes(searchTerm.toLowerCase()) ||
    row.manufacturer.toLowerCase().includes(searchTerm.toLowerCase()) ||
    row.locationCode.toLowerCase().includes(searchTerm.toLowerCase())
  ) ?? []

  const getAvailabilityColor = (quantity: number) => {
    if (quantity < 0) return 'destructive'
    if (quantity > 0) return 'default'
    return 'secondary'
  }

  return (
    <div className="space-y-6 p-6">
      <div>
        <h1 className="text-3xl font-bold tracking-tight">Inventory</h1>
        <p className="text-muted-foreground mt-2">Current local balances with reserved and available quantities.</p>
      </div>

      <Card>
        <CardHeader>
          <CardTitle>Inventory Items</CardTitle>
          <CardDescription>Search and filter inventory by item details</CardDescription>
        </CardHeader>
        <CardContent className="space-y-4">
          <Input
            placeholder="Search by item number, description, manufacturer, or location..."
            value={searchTerm}
            onChange={(e) => setSearchTerm(e.target.value)}
            className="w-full"
          />

          <div className="overflow-x-auto">
            <Table>
              <TableHeader>
                <TableRow>
                  <TableHead>Item No</TableHead>
                  <TableHead>Description</TableHead>
                  <TableHead>Manufacturer</TableHead>
                  <TableHead>Category</TableHead>
                  <TableHead>Location</TableHead>
                  <TableHead className="text-right">On Hand</TableHead>
                  <TableHead className="text-right">Reserved</TableHead>
                  <TableHead className="text-right">Available</TableHead>
                </TableRow>
              </TableHeader>
              <TableBody>
                {filteredBalances.map((row) => (
                  <TableRow key={`${row.itemId}-${row.locationCode}`}>
                    <TableCell className="font-medium">{row.itemNumber}</TableCell>
                    <TableCell>{row.description}</TableCell>
                    <TableCell>{row.manufacturer}</TableCell>
                    <TableCell>
                      <Badge variant="outline">{row.category}</Badge>
                    </TableCell>
                    <TableCell>{row.locationCode}</TableCell>
                    <TableCell className="text-right">{row.onHandQuantity}</TableCell>
                    <TableCell className="text-right">{row.reservedQuantity}</TableCell>
                    <TableCell className="text-right">
                      <Badge variant={getAvailabilityColor(row.availableQuantity)}>
                        {row.availableQuantity}
                      </Badge>
                    </TableCell>
                  </TableRow>
                ))}
              </TableBody>
            </Table>
          </div>

          {filteredBalances.length === 0 && (
            <div className="text-center py-8">
              <p className="text-muted-foreground">No inventory items found matching your search.</p>
            </div>
          )}
        </CardContent>
      </Card>
    </div>
  )
}
