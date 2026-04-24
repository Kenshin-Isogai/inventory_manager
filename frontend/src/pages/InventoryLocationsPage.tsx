import { Fragment, type FormEvent, useMemo, useState } from 'react'
import { useSWRConfig } from 'swr'
import { Activity, ChevronDown, ChevronRight, Edit, Loader2, MapPin, Plus } from 'lucide-react'
import { Link } from 'react-router-dom'

import { Badge } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'
import { Dialog, DialogContent, DialogFooter, DialogHeader, DialogTitle } from '@/components/ui/dialog'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from '@/components/ui/select'
import { Switch } from '@/components/ui/switch'
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from '@/components/ui/table'
import { useInventoryLocations } from '@/hooks/useInventoryLocations'
import { useInventoryOverview } from '@/hooks/useInventoryOverview'
import { upsertLocation } from '@/lib/mockApi'
import type { LocationSummary, LocationUpsertInput } from '@/types'

const ALL = '__all__'

const emptyLocation: LocationUpsertInput = {
  code: '',
  name: '',
  locationType: 'warehouse',
  isActive: true,
}

export function InventoryLocationsPage() {
  const { data: locationsData } = useInventoryLocations()
  const { data: overviewData } = useInventoryOverview()
  const { mutate } = useSWRConfig()
  const [searchTerm, setSearchTerm] = useState('')
  const [statusFilter, setStatusFilter] = useState(ALL)
  const [expandedLocations, setExpandedLocations] = useState<Set<string>>(new Set())
  const [dialogOpen, setDialogOpen] = useState(false)
  const [form, setForm] = useState<LocationUpsertInput>(emptyLocation)
  const [isSubmitting, setIsSubmitting] = useState(false)
  const [message, setMessage] = useState<{ tone: 'success' | 'error'; text: string } | null>(null)

  const locations = useMemo(() => locationsData?.rows ?? [], [locationsData?.rows])
  const balances = useMemo(() => overviewData?.balances ?? [], [overviewData?.balances])

  const balancesByLocation = useMemo(() => {
    return balances.reduce((acc, balance) => {
      const current = acc.get(balance.locationCode) ?? []
      current.push(balance)
      acc.set(balance.locationCode, current)
      return acc
    }, new Map<string, typeof balances>())
  }, [balances])

  const locationTypes = useMemo(
    () => Array.from(new Set(locations.map((row) => row.locationType).filter(Boolean))).sort(),
    [locations],
  )

  const filteredLocations = locations.filter((row) => {
    const term = searchTerm.trim().toLowerCase()
    const items = balancesByLocation.get(row.code) ?? []
    const matchesSearch = !term ||
      row.code.toLowerCase().includes(term) ||
      row.name.toLowerCase().includes(term) ||
      row.locationType.toLowerCase().includes(term) ||
      items.some((item) =>
        item.itemNumber.toLowerCase().includes(term) ||
        item.description.toLowerCase().includes(term) ||
        item.manufacturer.toLowerCase().includes(term),
      )
    const matchesStatus = statusFilter === ALL || (statusFilter === 'active' ? row.isActive : !row.isActive)
    return matchesSearch && matchesStatus
  })

  const totals = filteredLocations.reduce(
    (acc, row) => {
      acc.onHand += row.onHandQuantity
      acc.reserved += row.reservedQuantity
      acc.available += row.availableQuantity
      return acc
    },
    { onHand: 0, reserved: 0, available: 0 },
  )

  function toggleLocation(locationCode: string) {
    setExpandedLocations((current) => {
      const next = new Set(current)
      if (next.has(locationCode)) {
        next.delete(locationCode)
      } else {
        next.add(locationCode)
      }
      return next
    })
  }

  function openCreateDialog() {
    setForm(emptyLocation)
    setDialogOpen(true)
  }

  function openEditDialog(location: LocationSummary) {
    setForm({
      code: location.code,
      name: location.name,
      locationType: location.locationType,
      isActive: location.isActive,
    })
    setDialogOpen(true)
  }

  async function handleSubmit(event: FormEvent<HTMLFormElement>) {
    event.preventDefault()
    setMessage(null)
    if (!form.code.trim() || !form.name.trim()) {
      setMessage({ tone: 'error', text: 'Location code and name are required.' })
      return
    }
    setIsSubmitting(true)
    try {
      await upsertLocation({
        code: form.code.trim().toUpperCase(),
        name: form.name.trim(),
        locationType: form.locationType.trim() || 'warehouse',
        isActive: form.isActive,
      })
      await Promise.all([mutate('inventory-locations'), mutate('inventory-overview')])
      setDialogOpen(false)
      setMessage({ tone: 'success', text: `Saved location ${form.code.trim().toUpperCase()}.` })
    } catch (caught) {
      setMessage({ tone: 'error', text: caught instanceof Error ? caught.message : 'Location save failed.' })
    } finally {
      setIsSubmitting(false)
    }
  }

  return (
    <div className="space-y-6 p-6">
      <div className="flex flex-wrap items-start justify-between gap-4">
        <div>
          <h1 className="text-3xl font-bold tracking-tight">Inventory by Location</h1>
          <p className="mt-2 text-muted-foreground">Manage location master data and inspect stock at each storage point.</p>
        </div>
        <div className="flex flex-wrap gap-2">
          <Button asChild variant="outline" className="gap-2">
            <Link to="/app/inventory/events">
              <Activity className="h-4 w-4" />
              Record Operation
            </Link>
          </Button>
          <Button className="gap-2" onClick={openCreateDialog}>
            <Plus className="h-4 w-4" />
            New Location
          </Button>
        </div>
      </div>

      {message && (
        <div className={`rounded-lg border px-4 py-3 text-sm ${message.tone === 'success' ? 'border-green-200 bg-green-50 text-green-800' : 'border-red-200 bg-red-50 text-red-800'}`}>
          {message.text}
        </div>
      )}

      <div className="grid gap-4 md:grid-cols-4">
        <Card>
          <CardHeader className="pb-2"><CardTitle className="text-sm font-medium">Locations</CardTitle></CardHeader>
          <CardContent><p className="text-2xl font-bold">{filteredLocations.length}</p></CardContent>
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
            <MapPin className="h-5 w-5" />
            Locations
          </CardTitle>
        </CardHeader>
        <CardContent className="space-y-4">
          <div className="grid gap-3 md:grid-cols-[1fr_220px]">
            <Input
              placeholder="Search location code, name, type, item number, or manufacturer..."
              value={searchTerm}
              onChange={(event) => setSearchTerm(event.target.value)}
            />
            <Select value={statusFilter} onValueChange={setStatusFilter}>
              <SelectTrigger><SelectValue placeholder="Status" /></SelectTrigger>
              <SelectContent>
                <SelectItem value={ALL}>All statuses</SelectItem>
                <SelectItem value="active">Active</SelectItem>
                <SelectItem value="inactive">Inactive</SelectItem>
              </SelectContent>
            </Select>
          </div>

          <div className="overflow-x-auto rounded-md border">
            <Table>
              <TableHeader>
                <TableRow>
                  <TableHead className="w-10" />
                  <TableHead>Location</TableHead>
                  <TableHead>Type</TableHead>
                  <TableHead>Status</TableHead>
                  <TableHead className="text-right">On Hand</TableHead>
                  <TableHead className="text-right">Reserved</TableHead>
                  <TableHead className="text-right">Available</TableHead>
                  <TableHead className="text-right">Actions</TableHead>
                </TableRow>
              </TableHeader>
              <TableBody>
                {filteredLocations.map((location) => {
                  const items = balancesByLocation.get(location.code) ?? []
                  const expanded = expandedLocations.has(location.code)
                  return (
                    <Fragment key={location.code}>
                      <TableRow key={location.code}>
                        <TableCell>
                          <Button variant="ghost" size="icon" className="h-8 w-8" onClick={() => toggleLocation(location.code)}>
                            {expanded ? <ChevronDown className="h-4 w-4" /> : <ChevronRight className="h-4 w-4" />}
                          </Button>
                        </TableCell>
                        <TableCell>
                          <div className="font-medium">{location.code}</div>
                          <div className="text-sm text-muted-foreground">{location.name}</div>
                        </TableCell>
                        <TableCell><Badge variant="outline">{location.locationType}</Badge></TableCell>
                        <TableCell><Badge variant={location.isActive ? 'default' : 'secondary'}>{location.isActive ? 'Active' : 'Inactive'}</Badge></TableCell>
                        <TableCell className="text-right">{location.onHandQuantity}</TableCell>
                        <TableCell className="text-right">{location.reservedQuantity}</TableCell>
                        <TableCell className="text-right">
                          <Badge variant={location.availableQuantity < 0 ? 'destructive' : location.availableQuantity > 0 ? 'default' : 'secondary'}>
                            {location.availableQuantity}
                          </Badge>
                        </TableCell>
                        <TableCell className="text-right">
                          <Button variant="ghost" size="icon" className="h-8 w-8" onClick={() => openEditDialog(location)}>
                            <Edit className="h-4 w-4" />
                          </Button>
                        </TableCell>
                      </TableRow>
                      {expanded && (
                        <TableRow key={`${location.code}-items`}>
                          <TableCell />
                          <TableCell colSpan={7} className="bg-muted/30">
                            <div className="grid gap-2 py-2">
                              {items.map((item) => (
                                <div key={`${item.itemId}-${item.locationCode}`} className="grid gap-3 rounded-md border bg-background p-3 text-sm lg:grid-cols-[1fr_160px_repeat(3,110px)]">
                                  <div>
                                    <div className="font-medium">{item.itemNumber}</div>
                                    <div className="text-muted-foreground">{item.description}</div>
                                  </div>
                                  <div>{item.manufacturer}</div>
                                  <div className="text-right">On hand {item.onHandQuantity}</div>
                                  <div className="text-right">Reserved {item.reservedQuantity}</div>
                                  <div className="text-right">Available {item.availableQuantity}</div>
                                </div>
                              ))}
                              {items.length === 0 && <p className="py-3 text-sm text-muted-foreground">No stock has been recorded at this location.</p>}
                            </div>
                          </TableCell>
                        </TableRow>
                      )}
                    </Fragment>
                  )
                })}
                {filteredLocations.length === 0 && (
                  <TableRow>
                    <TableCell colSpan={8} className="py-10 text-center text-muted-foreground">
                      No locations match the current filters.
                    </TableCell>
                  </TableRow>
                )}
              </TableBody>
            </Table>
          </div>
        </CardContent>
      </Card>

      <Dialog open={dialogOpen} onOpenChange={setDialogOpen}>
        <DialogContent>
          <DialogHeader>
            <DialogTitle>{locations.some((row) => row.code === form.code) ? 'Edit Location' : 'New Location'}</DialogTitle>
          </DialogHeader>
          <form onSubmit={handleSubmit} className="space-y-4">
            <div className="grid gap-4 sm:grid-cols-2">
              <div className="space-y-2">
                <Label htmlFor="location-code">Code</Label>
                <Input
                  id="location-code"
                  value={form.code}
                  onChange={(event) => setForm((current) => ({ ...current, code: event.target.value.toUpperCase() }))}
                  placeholder="TOKYO-A1"
                  disabled={locations.some((row) => row.code === form.code)}
                />
              </div>
              <div className="space-y-2">
                <Label htmlFor="location-type">Type</Label>
                <Input
                  id="location-type"
                  value={form.locationType}
                  onChange={(event) => setForm((current) => ({ ...current, locationType: event.target.value }))}
                  list="location-types"
                  placeholder="warehouse"
                />
                <datalist id="location-types">
                  {locationTypes.map((type) => <option key={type} value={type} />)}
                </datalist>
              </div>
              <div className="space-y-2 sm:col-span-2">
                <Label htmlFor="location-name">Name</Label>
                <Input
                  id="location-name"
                  value={form.name}
                  onChange={(event) => setForm((current) => ({ ...current, name: event.target.value }))}
                  placeholder="Tokyo Assembly A1"
                />
              </div>
              <div className="flex items-center justify-between rounded-md border p-3 sm:col-span-2">
                <div>
                  <Label htmlFor="location-active">Active</Label>
                  <p className="text-sm text-muted-foreground">Inactive locations remain visible for history but should not be selected for new stock operations.</p>
                </div>
                <Switch
                  id="location-active"
                  checked={form.isActive}
                  onCheckedChange={(checked) => setForm((current) => ({ ...current, isActive: checked }))}
                />
              </div>
            </div>
            <DialogFooter>
              <Button type="button" variant="outline" onClick={() => setDialogOpen(false)} disabled={isSubmitting}>Cancel</Button>
              <Button type="submit" className="gap-2" disabled={isSubmitting}>
                {isSubmitting && <Loader2 className="h-4 w-4 animate-spin" />}
                Save Location
              </Button>
            </DialogFooter>
          </form>
        </DialogContent>
      </Dialog>
    </div>
  )
}
