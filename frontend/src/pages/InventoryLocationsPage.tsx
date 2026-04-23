import { useState } from 'react'
import { useInventoryOverview } from '@/hooks/useInventoryOverview'
import { Card, CardContent } from '@/components/ui/card'
import { Badge } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'
import { ChevronDown, ChevronRight } from 'lucide-react'

export function InventoryLocationsPage() {
  const { data } = useInventoryOverview()
  const [expandedLocations, setExpandedLocations] = useState<Set<string>>(new Set())

  // Group balances by location
  const balances = data?.balances ?? []
  const locationGroups = balances.reduce(
    (acc, balance) => {
      const key = balance.locationCode
      if (!acc[key]) {
        acc[key] = []
      }
      acc[key].push(balance)
      return acc
    },
    {} as Record<string, typeof balances>
  )

  const locations = Object.keys(locationGroups).sort()

  const toggleLocation = (locationCode: string) => {
    const newExpanded = new Set(expandedLocations)
    if (newExpanded.has(locationCode)) {
      newExpanded.delete(locationCode)
    } else {
      newExpanded.add(locationCode)
    }
    setExpandedLocations(newExpanded)
  }

  const getLocationStats = (items: typeof balances) => {
    return {
      totalOnHand: items.reduce((sum, item) => sum + item.onHandQuantity, 0),
      totalReserved: items.reduce((sum, item) => sum + item.reservedQuantity, 0),
      totalAvailable: items.reduce((sum, item) => sum + item.availableQuantity, 0),
      itemCount: items.length,
    }
  }

  const getAvailabilityColor = (quantity: number) => {
    if (quantity < 0) return 'destructive'
    if (quantity > 0) return 'default'
    return 'secondary'
  }

  return (
    <div className="space-y-6 p-6">
      <div>
        <h1 className="text-3xl font-bold tracking-tight">Inventory by Location</h1>
        <p className="text-muted-foreground mt-2">View inventory balances organized by storage location.</p>
      </div>

      <div className="space-y-4">
        {locations.map((locationCode) => {
          const items = locationGroups[locationCode]
          const stats = getLocationStats(items)
          const isExpanded = expandedLocations.has(locationCode)

          return (
            <Card key={locationCode}>
              <div className="border-b">
                <Button
                  variant="ghost"
                  className="w-full justify-start rounded-none h-auto py-4"
                  onClick={() => toggleLocation(locationCode)}
                >
                  <div className="flex items-center gap-3 flex-1">
                    {isExpanded ? <ChevronDown className="h-4 w-4" /> : <ChevronRight className="h-4 w-4" />}
                    <div className="text-left flex-1">
                      <h3 className="font-semibold text-lg">{locationCode}</h3>
                      <p className="text-sm text-muted-foreground">{stats.itemCount} items</p>
                    </div>
                  </div>
                  <div className="flex gap-6 text-right">
                    <div>
                      <p className="text-xs text-muted-foreground">On Hand</p>
                      <p className="font-semibold">{stats.totalOnHand}</p>
                    </div>
                    <div>
                      <p className="text-xs text-muted-foreground">Reserved</p>
                      <p className="font-semibold">{stats.totalReserved}</p>
                    </div>
                    <div>
                      <p className="text-xs text-muted-foreground">Available</p>
                      <p className={`font-semibold ${stats.totalAvailable < 0 ? 'text-destructive' : ''}`}>
                        {stats.totalAvailable}
                      </p>
                    </div>
                  </div>
                </Button>
              </div>

              {isExpanded && (
                <CardContent className="pt-4">
                  <div className="space-y-3">
                    {items?.map((item) => (
                      <div
                        key={`${item.itemId}-${item.locationCode}`}
                        className="border rounded-lg p-4 space-y-2"
                      >
                        <div className="flex items-start justify-between">
                          <div className="flex-1">
                            <p className="font-semibold">{item.itemNumber}</p>
                            <p className="text-sm text-muted-foreground">{item.description}</p>
                          </div>
                          <div className="flex gap-2">
                            <Badge variant="outline">{item.category}</Badge>
                            <Badge variant="outline">{item.manufacturer}</Badge>
                          </div>
                        </div>
                        <div className="flex gap-8 text-sm">
                          <div>
                            <span className="text-muted-foreground">On Hand: </span>
                            <span className="font-semibold">{item.onHandQuantity}</span>
                          </div>
                          <div>
                            <span className="text-muted-foreground">Reserved: </span>
                            <span className="font-semibold">{item.reservedQuantity}</span>
                          </div>
                          <div>
                            <span className="text-muted-foreground">Available: </span>
                            <Badge variant={getAvailabilityColor(item.availableQuantity)}>
                              {item.availableQuantity}
                            </Badge>
                          </div>
                        </div>
                      </div>
                    ))}
                  </div>
                </CardContent>
              )}
            </Card>
          )
        })}
      </div>

      {locations.length === 0 && (
        <Card>
          <CardContent className="py-12 text-center">
            <p className="text-muted-foreground">No inventory locations found.</p>
          </CardContent>
        </Card>
      )}
    </div>
  )
}
