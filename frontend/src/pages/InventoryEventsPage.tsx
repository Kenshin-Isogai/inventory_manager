import { useState } from 'react'
import type { FormEvent } from 'react'
import { useSWRConfig } from 'swr'
import { useInventoryOverview } from '@/hooks/useInventoryOverview'
import { adjustInventory } from '@/lib/mockApi'
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card'
import { Tabs, TabsContent, TabsList, TabsTrigger } from '@/components/ui/tabs'
import { Label } from '@/components/ui/label'
import { Input } from '@/components/ui/input'
import { Button } from '@/components/ui/button'
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select'
import { CheckCircle } from 'lucide-react'

export function InventoryEventsPage() {
  const { data } = useInventoryOverview()
  const { mutate } = useSWRConfig()

  // Form state
  const [itemId, setItemId] = useState('item-er2')
  const [locationCode, setLocationCode] = useState('TOKYO-A1')
  const [quantityDelta, setQuantityDelta] = useState(-1)
  const [deviceScopeId, setDeviceScopeId] = useState('ds-er2-powerboard')
  const [note, setNote] = useState('Inventory adjustment')
  const [message, setMessage] = useState('')
  const [isSubmitting, setIsSubmitting] = useState(false)

  async function handleSubmit(event: FormEvent<HTMLFormElement>) {
    event.preventDefault()
    setIsSubmitting(true)
    try {
      await adjustInventory({ itemId, locationCode, quantityDelta, deviceScopeId, note })
      await mutate('inventory-overview')
      setMessage(`Recorded adjustment ${quantityDelta} for ${itemId} at ${locationCode}.`)
      // Reset form
      setNote('Inventory adjustment')
    } finally {
      setIsSubmitting(false)
    }
  }

  const selectedItem = data?.balances.find((b) => b.itemId === itemId)

  return (
    <div className="space-y-6 p-6">
      <div>
        <h1 className="text-3xl font-bold tracking-tight">Inventory Events</h1>
        <p className="text-muted-foreground mt-2">Record receive, move, and adjust operations for inventory items.</p>
      </div>

      <Tabs defaultValue="adjust" className="space-y-4">
        <TabsList>
          <TabsTrigger value="adjust">Adjust</TabsTrigger>
          <TabsTrigger value="receive">Receive</TabsTrigger>
          <TabsTrigger value="move">Move</TabsTrigger>
        </TabsList>

        <TabsContent value="adjust">
          <Card>
            <CardHeader>
              <CardTitle>Adjust Inventory</CardTitle>
              <CardDescription>Record quantity adjustments for inventory items</CardDescription>
            </CardHeader>
            <CardContent>
              <form onSubmit={handleSubmit} className="space-y-6">
                <div className="grid gap-4">
                  <div className="space-y-2">
                    <Label htmlFor="item">Item</Label>
                    <Select value={itemId} onValueChange={setItemId}>
                      <SelectTrigger id="item">
                        <SelectValue placeholder="Select an item" />
                      </SelectTrigger>
                      <SelectContent>
                        {data?.balances.map((row) => (
                          <SelectItem key={row.itemId} value={row.itemId}>
                            {row.itemNumber} / {row.description}
                          </SelectItem>
                        ))}
                      </SelectContent>
                    </Select>
                  </div>

                  {selectedItem && (
                    <div className="bg-muted p-4 rounded-lg space-y-2 text-sm">
                      <p>
                        <span className="text-muted-foreground">Current On Hand: </span>
                        <span className="font-semibold">{selectedItem.onHandQuantity}</span>
                      </p>
                      <p>
                        <span className="text-muted-foreground">Location: </span>
                        <span className="font-semibold">{selectedItem.locationCode}</span>
                      </p>
                    </div>
                  )}

                  <div className="grid grid-cols-2 gap-4">
                    <div className="space-y-2">
                      <Label htmlFor="location">Location</Label>
                      <Input
                        id="location"
                        value={locationCode}
                        onChange={(e) => setLocationCode(e.target.value)}
                        placeholder="e.g., TOKYO-A1"
                      />
                    </div>
                    <div className="space-y-2">
                      <Label htmlFor="quantity">Quantity Delta</Label>
                      <Input
                        id="quantity"
                        type="number"
                        value={quantityDelta}
                        onChange={(e) => setQuantityDelta(Number(e.target.value) || 0)}
                        placeholder="e.g., 5 or -3"
                      />
                    </div>
                  </div>

                  <div className="grid grid-cols-2 gap-4">
                    <div className="space-y-2">
                      <Label htmlFor="device">Device Scope</Label>
                      <Input
                        id="device"
                        value={deviceScopeId}
                        onChange={(e) => setDeviceScopeId(e.target.value)}
                        placeholder="e.g., ds-er2-powerboard"
                      />
                    </div>
                    <div className="space-y-2">
                      <Label htmlFor="note">Note</Label>
                      <Input
                        id="note"
                        value={note}
                        onChange={(e) => setNote(e.target.value)}
                        placeholder="Adjustment reason..."
                      />
                    </div>
                  </div>
                </div>

                {message && (
                  <div className="flex gap-3 rounded-lg border border-green-200 bg-green-50 p-3">
                    <CheckCircle className="h-5 w-5 text-green-600 flex-shrink-0 mt-0.5" />
                    <p className="text-sm text-green-800">{message}</p>
                  </div>
                )}

                <Button type="submit" disabled={isSubmitting} className="w-full">
                  {isSubmitting ? 'Recording...' : 'Record Adjustment'}
                </Button>
              </form>
            </CardContent>
          </Card>
        </TabsContent>

        <TabsContent value="receive">
          <Card>
            <CardHeader>
              <CardTitle>Receive Inventory</CardTitle>
              <CardDescription>Record receipt of new inventory items</CardDescription>
            </CardHeader>
            <CardContent>
              <form onSubmit={handleSubmit} className="space-y-6">
                <div className="grid gap-4">
                  <div className="space-y-2">
                    <Label htmlFor="receive-item">Item</Label>
                    <Select value={itemId} onValueChange={setItemId}>
                      <SelectTrigger id="receive-item">
                        <SelectValue placeholder="Select an item" />
                      </SelectTrigger>
                      <SelectContent>
                        {data?.balances.map((row) => (
                          <SelectItem key={row.itemId} value={row.itemId}>
                            {row.itemNumber} / {row.description}
                          </SelectItem>
                        ))}
                      </SelectContent>
                    </Select>
                  </div>

                  <div className="grid grid-cols-2 gap-4">
                    <div className="space-y-2">
                      <Label htmlFor="receive-location">Receive Location</Label>
                      <Input
                        id="receive-location"
                        value={locationCode}
                        onChange={(e) => setLocationCode(e.target.value)}
                        placeholder="e.g., TOKYO-A1"
                      />
                    </div>
                    <div className="space-y-2">
                      <Label htmlFor="receive-qty">Quantity Received</Label>
                      <Input
                        id="receive-qty"
                        type="number"
                        value={quantityDelta}
                        onChange={(e) => setQuantityDelta(Math.max(0, Number(e.target.value) || 0))}
                        placeholder="Number of units"
                        min="0"
                      />
                    </div>
                  </div>

                  <div className="space-y-2">
                    <Label htmlFor="receive-note">Receipt Note</Label>
                    <Input
                      id="receive-note"
                      value={note}
                      onChange={(e) => setNote(e.target.value)}
                      placeholder="e.g., PO reference, batch number..."
                    />
                  </div>
                </div>

                {message && (
                  <div className="flex gap-3 rounded-lg border border-green-200 bg-green-50 p-3">
                    <CheckCircle className="h-5 w-5 text-green-600 flex-shrink-0 mt-0.5" />
                    <p className="text-sm text-green-800">{message}</p>
                  </div>
                )}

                <Button type="submit" disabled={isSubmitting} className="w-full">
                  {isSubmitting ? 'Recording...' : 'Record Receipt'}
                </Button>
              </form>
            </CardContent>
          </Card>
        </TabsContent>

        <TabsContent value="move">
          <Card>
            <CardHeader>
              <CardTitle>Move Inventory</CardTitle>
              <CardDescription>Transfer inventory between locations</CardDescription>
            </CardHeader>
            <CardContent>
              <form onSubmit={handleSubmit} className="space-y-6">
                <div className="grid gap-4">
                  <div className="space-y-2">
                    <Label htmlFor="move-item">Item</Label>
                    <Select value={itemId} onValueChange={setItemId}>
                      <SelectTrigger id="move-item">
                        <SelectValue placeholder="Select an item" />
                      </SelectTrigger>
                      <SelectContent>
                        {data?.balances.map((row) => (
                          <SelectItem key={row.itemId} value={row.itemId}>
                            {row.itemNumber} / {row.description}
                          </SelectItem>
                        ))}
                      </SelectContent>
                    </Select>
                  </div>

                  <div className="grid grid-cols-2 gap-4">
                    <div className="space-y-2">
                      <Label htmlFor="from-location">From Location</Label>
                      <Input
                        id="from-location"
                        value={locationCode}
                        onChange={(e) => setLocationCode(e.target.value)}
                        placeholder="Current location"
                      />
                    </div>
                    <div className="space-y-2">
                      <Label htmlFor="to-location">To Location</Label>
                      <Input
                        id="to-location"
                        value={deviceScopeId}
                        onChange={(e) => setDeviceScopeId(e.target.value)}
                        placeholder="Destination location"
                      />
                    </div>
                  </div>

                  <div className="space-y-2">
                    <Label htmlFor="move-qty">Quantity to Move</Label>
                    <Input
                      id="move-qty"
                      type="number"
                      value={quantityDelta}
                      onChange={(e) => setQuantityDelta(Math.max(0, Number(e.target.value) || 0))}
                      placeholder="Number of units"
                      min="0"
                    />
                  </div>

                  <div className="space-y-2">
                    <Label htmlFor="move-note">Move Reason</Label>
                    <Input
                      id="move-note"
                      value={note}
                      onChange={(e) => setNote(e.target.value)}
                      placeholder="e.g., Reorganization, consolidation..."
                    />
                  </div>
                </div>

                {message && (
                  <div className="flex gap-3 rounded-lg border border-green-200 bg-green-50 p-3">
                    <CheckCircle className="h-5 w-5 text-green-600 flex-shrink-0 mt-0.5" />
                    <p className="text-sm text-green-800">{message}</p>
                  </div>
                )}

                <Button type="submit" disabled={isSubmitting} className="w-full">
                  {isSubmitting ? 'Recording...' : 'Record Move'}
                </Button>
              </form>
            </CardContent>
          </Card>
        </TabsContent>
      </Tabs>

      <Card>
        <CardHeader>
          <CardTitle>Recent Events</CardTitle>
          <CardDescription>History of recent inventory operations</CardDescription>
        </CardHeader>
        <CardContent>
          <p className="text-muted-foreground text-sm">No recent events recorded yet.</p>
        </CardContent>
      </Card>
    </div>
  )
}
