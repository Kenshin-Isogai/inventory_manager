import { Fragment, useMemo, useState } from 'react'
import { Link } from 'react-router-dom'
import { Activity, Boxes, ChevronDown, ChevronRight, MapPin, PackageSearch } from 'lucide-react'

import { Badge } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'
import { Input } from '@/components/ui/input'
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from '@/components/ui/select'
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from '@/components/ui/table'
import { useInventoryItems } from '@/hooks/useInventoryItems'
import { useInventoryOverview } from '@/hooks/useInventoryOverview'

const ALL = '__all__'

export function InventoryPage() {
  const { data: itemsData } = useInventoryItems()
  const { data: overviewData } = useInventoryOverview()
  const [searchTerm, setSearchTerm] = useState('')
  const [manufacturerFilter, setManufacturerFilter] = useState(ALL)
  const [locationFilter, setLocationFilter] = useState(ALL)
  const [expandedItems, setExpandedItems] = useState<Set<string>>(new Set())

  const items = useMemo(() => itemsData?.rows ?? [], [itemsData?.rows])
  const balances = useMemo(() => overviewData?.balances ?? [], [overviewData?.balances])
  const manufacturers = useMemo(
    () => Array.from(new Set(items.map((row) => row.manufacturer).filter(Boolean))).sort(),
    [items],
  )
  const locations = useMemo(
    () => Array.from(new Set(balances.map((row) => row.locationCode).filter(Boolean))).sort(),
    [balances],
  )

  const locationBalancesByItem = useMemo(() => {
    return balances.reduce((acc, row) => {
      const current = acc.get(row.itemId) ?? []
      current.push(row)
      acc.set(row.itemId, current)
      return acc
    }, new Map<string, typeof balances>())
  }, [balances])

  const filteredItems = items.filter((row) => {
    const term = searchTerm.trim().toLowerCase()
    const itemLocations = locationBalancesByItem.get(row.itemId) ?? []
    const matchesSearch = !term ||
      row.itemNumber.toLowerCase().includes(term) ||
      row.description.toLowerCase().includes(term) ||
      row.manufacturer.toLowerCase().includes(term) ||
      row.category.toLowerCase().includes(term) ||
      itemLocations.some((balance) => balance.locationCode.toLowerCase().includes(term))
    const matchesManufacturer = manufacturerFilter === ALL || row.manufacturer === manufacturerFilter
    const matchesLocation = locationFilter === ALL || itemLocations.some((balance) => balance.locationCode === locationFilter)
    return matchesSearch && matchesManufacturer && matchesLocation
  })

  const totals = filteredItems.reduce(
    (acc, row) => {
      acc.onHand += row.onHandQuantity
      acc.reserved += row.reservedQuantity
      acc.available += row.availableQuantity
      return acc
    },
    { onHand: 0, reserved: 0, available: 0 },
  )

  function toggleItem(itemId: string) {
    setExpandedItems((current) => {
      const next = new Set(current)
      if (next.has(itemId)) {
        next.delete(itemId)
      } else {
        next.add(itemId)
      }
      return next
    })
  }

  return (
    <div className="space-y-6 p-6">
      <div className="flex flex-wrap items-start justify-between gap-4">
        <div>
          <h1 className="text-3xl font-bold tracking-tight">Inventory by Item</h1>
          <p className="mt-2 text-muted-foreground">Item-level stock totals with per-location balance drilldown.</p>
        </div>
        <Button asChild variant="outline" className="gap-2">
          <Link to="/app/inventory/events">
            <Activity className="h-4 w-4" />
            Record Operation
          </Link>
        </Button>
      </div>

      <div className="grid gap-4 md:grid-cols-4">
        <Card>
          <CardHeader className="pb-2"><CardTitle className="text-sm font-medium">Items</CardTitle></CardHeader>
          <CardContent><p className="text-2xl font-bold">{filteredItems.length}</p></CardContent>
        </Card>
        <Card>
          <CardHeader className="pb-2"><CardTitle className="text-sm font-medium">On Hand</CardTitle></CardHeader>
          <CardContent><p className="text-2xl font-bold">{totals.onHand}</p></CardContent>
        </Card>
        <Card>
          <CardHeader className="pb-2"><CardTitle className="text-sm font-medium">Reserved</CardTitle></CardHeader>
          <CardContent><p className="text-2xl font-bold">{totals.reserved}</p></CardContent>
        </Card>
        <Card>
          <CardHeader className="pb-2"><CardTitle className="text-sm font-medium">Available</CardTitle></CardHeader>
          <CardContent><p className={`text-2xl font-bold ${totals.available < 0 ? 'text-destructive' : ''}`}>{totals.available}</p></CardContent>
        </Card>
      </div>

      <Card>
        <CardHeader>
          <CardTitle className="flex items-center gap-2">
            <PackageSearch className="h-5 w-5" />
            Item Balances
          </CardTitle>
        </CardHeader>
        <CardContent className="space-y-4">
          <div className="grid gap-3 lg:grid-cols-[1fr_220px_220px]">
            <Input
              placeholder="Search item number, description, manufacturer, category, or location..."
              value={searchTerm}
              onChange={(event) => setSearchTerm(event.target.value)}
            />
            <Select value={manufacturerFilter} onValueChange={setManufacturerFilter}>
              <SelectTrigger><SelectValue placeholder="Manufacturer" /></SelectTrigger>
              <SelectContent>
                <SelectItem value={ALL}>All manufacturers</SelectItem>
                {manufacturers.map((manufacturer) => (
                  <SelectItem key={manufacturer} value={manufacturer}>{manufacturer}</SelectItem>
                ))}
              </SelectContent>
            </Select>
            <Select value={locationFilter} onValueChange={setLocationFilter}>
              <SelectTrigger><SelectValue placeholder="Location" /></SelectTrigger>
              <SelectContent>
                <SelectItem value={ALL}>All locations</SelectItem>
                {locations.map((location) => (
                  <SelectItem key={location} value={location}>{location}</SelectItem>
                ))}
              </SelectContent>
            </Select>
          </div>

          <div className="overflow-x-auto rounded-md border">
            <Table>
              <TableHeader>
                <TableRow>
                  <TableHead className="w-10" />
                  <TableHead>Item</TableHead>
                  <TableHead>Manufacturer</TableHead>
                  <TableHead>Category</TableHead>
                  <TableHead className="text-right">On Hand</TableHead>
                  <TableHead className="text-right">Reserved</TableHead>
                  <TableHead className="text-right">Available</TableHead>
                  <TableHead className="text-right">Flow</TableHead>
                </TableRow>
              </TableHeader>
              <TableBody>
                {filteredItems.map((row) => {
                  const itemBalances = (locationBalancesByItem.get(row.itemId) ?? []).filter(
                    (balance) => locationFilter === ALL || balance.locationCode === locationFilter,
                  )
                  const expanded = expandedItems.has(row.itemId)
                  return (
                    <Fragment key={row.itemId}>
                      <TableRow key={row.itemId}>
                        <TableCell>
                          <Button variant="ghost" size="icon" className="h-8 w-8" onClick={() => toggleItem(row.itemId)}>
                            {expanded ? <ChevronDown className="h-4 w-4" /> : <ChevronRight className="h-4 w-4" />}
                          </Button>
                        </TableCell>
                        <TableCell>
                          <div className="font-medium">{row.itemNumber}</div>
                          <div className="text-sm text-muted-foreground">{row.description}</div>
                        </TableCell>
                        <TableCell>{row.manufacturer}</TableCell>
                        <TableCell><Badge variant="outline">{row.category}</Badge></TableCell>
                        <TableCell className="text-right">{row.onHandQuantity}</TableCell>
                        <TableCell className="text-right">{row.reservedQuantity}</TableCell>
                        <TableCell className="text-right">
                          <Badge variant={row.availableQuantity < 0 ? 'destructive' : row.availableQuantity > 0 ? 'default' : 'secondary'}>
                            {row.availableQuantity}
                          </Badge>
                        </TableCell>
                        <TableCell className="text-right">
                          <Button asChild variant="ghost" size="sm">
                            <Link to={`/app/inventory/items/${row.itemId}/flow`}>Open</Link>
                          </Button>
                        </TableCell>
                      </TableRow>
                      {expanded && (
                        <TableRow key={`${row.itemId}-locations`}>
                          <TableCell />
                          <TableCell colSpan={7} className="bg-muted/30">
                            <div className="grid gap-2 py-2">
                              {itemBalances.map((balance) => (
                                <div key={`${balance.itemId}-${balance.locationCode}`} className="grid gap-3 rounded-md border bg-background p-3 text-sm md:grid-cols-[1fr_repeat(3,120px)]">
                                  <div className="flex items-center gap-2 font-medium"><MapPin className="h-4 w-4" />{balance.locationCode}</div>
                                  <div className="text-right">On hand {balance.onHandQuantity}</div>
                                  <div className="text-right">Reserved {balance.reservedQuantity}</div>
                                  <div className="text-right">Available {balance.availableQuantity}</div>
                                </div>
                              ))}
                              {itemBalances.length === 0 && <p className="py-3 text-sm text-muted-foreground">No balances for the selected location.</p>}
                            </div>
                          </TableCell>
                        </TableRow>
                      )}
                    </Fragment>
                  )
                })}
                {filteredItems.length === 0 && (
                  <TableRow>
                    <TableCell colSpan={8} className="py-10 text-center text-muted-foreground">
                      No items match the current filters.
                    </TableCell>
                  </TableRow>
                )}
              </TableBody>
            </Table>
          </div>
          <div className="flex items-center gap-2 text-sm text-muted-foreground">
            <Boxes className="h-4 w-4" />
            Location rows come from current inventory balances; create empty locations from the Locations tab.
          </div>
        </CardContent>
      </Card>
    </div>
  )
}
