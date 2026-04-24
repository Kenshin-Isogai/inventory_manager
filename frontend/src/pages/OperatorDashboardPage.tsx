import { useState } from 'react'
import type { FormEvent } from 'react'
import { useSWRConfig } from 'swr'
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card'
import { Badge } from '@/components/ui/badge'
import { useDashboard } from '@/hooks/useDashboard'
import { useDeviceScopes } from '@/hooks/useDeviceScopes'
import { useDevices } from '@/hooks/useDevices'
import { upsertDeviceScope } from '@/lib/mockApi'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select'
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from '@/components/ui/table'
import { AlertCircle } from 'lucide-react'

const NEW_SCOPE = '__new__'
const scopeTypeOptions = ['device_root', 'subsystem', 'module', 'area', 'work_package']
const statusOptions = ['active', 'inactive']

export function OperatorDashboardPage() {
  const { data } = useDashboard()
  const { data: deviceScopeData } = useDeviceScopes()
  const { data: deviceData } = useDevices()
  const { mutate } = useSWRConfig()
  const [selectedScopeId, setSelectedScopeId] = useState(NEW_SCOPE)
  const [deviceKey, setDeviceKey] = useState('')
  const [scopeKey, setScopeKey] = useState('')
  const [scopeName, setScopeName] = useState('')
  const [scopeType, setScopeType] = useState('subsystem')
  const [ownerDepartmentKey, setOwnerDepartmentKey] = useState('')
  const [status, setStatus] = useState('active')
  const [message, setMessage] = useState('')
  const [isSaving, setIsSaving] = useState(false)

  const scopes = deviceScopeData?.rows ?? []
  const devices = deviceData?.rows.filter((row) => row.status !== 'inactive') ?? []
  const selectedDeviceKey = deviceKey || devices[0]?.deviceKey || ''

  function resetForm(nextDeviceKey?: string) {
    setSelectedScopeId(NEW_SCOPE)
    setDeviceKey(nextDeviceKey ?? devices[0]?.deviceKey ?? '')
    setScopeKey('')
    setScopeName('')
    setScopeType('subsystem')
    setOwnerDepartmentKey('')
    setStatus('active')
  }

  function handleTargetChange(value: string) {
    setSelectedScopeId(value)
    if (value === NEW_SCOPE) {
      resetForm(deviceKey)
      return
    }
    const selected = scopes.find((row) => row.id === value)
    if (!selected) {
      return
    }
    setDeviceKey(selected.deviceKey)
    setScopeKey(selected.scopeKey)
    setScopeName(selected.scopeName)
    setScopeType(selected.scopeType || 'subsystem')
    setOwnerDepartmentKey(selected.ownerDepartmentKey)
    setStatus(selected.status || 'active')
  }

  async function handleScopeSubmit(event: FormEvent<HTMLFormElement>) {
    event.preventDefault()
    setIsSaving(true)
    setMessage('')
    try {
      await upsertDeviceScope({
        id: selectedScopeId === NEW_SCOPE ? undefined : selectedScopeId,
        deviceKey: selectedDeviceKey,
        scopeKey,
        scopeName,
        scopeType,
        ownerDepartmentKey,
        status,
      })
      await Promise.all([mutate('device-scopes'), mutate('devices')])
      setMessage(selectedScopeId === NEW_SCOPE ? 'Scope created.' : 'Scope updated.')
      if (selectedScopeId === NEW_SCOPE) {
        resetForm(selectedDeviceKey)
      }
    } finally {
      setIsSaving(false)
    }
  }

  return (
    <div className="space-y-6 p-6">
      <div className="space-y-2">
        <h1 className="text-3xl font-bold tracking-tight">Operator Dashboard</h1>
        <p className="text-muted-foreground">
          Requirements, shortages, and pending follow-up.
        </p>
      </div>

      <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-4 gap-4">
        {data?.metrics.map((metric) => (
          <Card key={metric.label}>
            <CardContent className="pt-6">
              <div className="space-y-2">
                <p className="text-sm text-muted-foreground">{metric.label}</p>
                <div className="flex items-baseline justify-between">
                  <p className="text-2xl font-bold">{metric.value}</p>
                  {metric.delta && (
                    <Badge variant="outline" className="text-xs">
                      {metric.delta}
                    </Badge>
                  )}
                </div>
              </div>
            </CardContent>
          </Card>
        ))}
      </div>

      {data?.alerts && data.alerts.length > 0 && (
        <Card className="border-amber-200 bg-amber-50">
          <CardHeader>
            <CardTitle className="flex items-center gap-2 text-lg">
              <AlertCircle className="w-5 h-5 text-amber-600" />
              Alerts
            </CardTitle>
            <CardDescription>Operational alerts that need attention</CardDescription>
          </CardHeader>
          <CardContent>
            <ul className="space-y-2">
              {data.alerts.map((alert, idx) => (
                <li key={idx} className="text-sm text-amber-900 flex gap-2">
                  <span className="text-amber-600">•</span>
                  <span>{alert}</span>
                </li>
              ))}
            </ul>
          </CardContent>
        </Card>
      )}

      <div className="grid gap-6 xl:grid-cols-[minmax(0,1.2fr)_minmax(0,0.8fr)]">
        <Card>
          <CardHeader>
            <CardTitle>Scope Master</CardTitle>
            <CardDescription>Operator-managed device and scope definitions used by reservations and filters.</CardDescription>
          </CardHeader>
          <CardContent>
            <div className="overflow-x-auto">
              <Table>
                <TableHeader>
                  <TableRow>
                    <TableHead>Device</TableHead>
                    <TableHead>Scope Key</TableHead>
                    <TableHead>Scope Name</TableHead>
                    <TableHead>Type</TableHead>
                    <TableHead>Owner</TableHead>
                    <TableHead>Status</TableHead>
                  </TableRow>
                </TableHeader>
                <TableBody>
                  {scopes.map((row) => (
                    <TableRow
                      key={row.id}
                      className="cursor-pointer"
                      onClick={() => handleTargetChange(row.id)}
                    >
                      <TableCell className="font-medium">{row.deviceKey}</TableCell>
                      <TableCell>{row.scopeKey}</TableCell>
                      <TableCell>{row.scopeName}</TableCell>
                      <TableCell>{row.scopeType || '-'}</TableCell>
                      <TableCell>{row.ownerDepartmentKey || '-'}</TableCell>
                      <TableCell>
                        <Badge variant={row.status === 'active' ? 'default' : 'secondary'}>{row.status}</Badge>
                      </TableCell>
                    </TableRow>
                  ))}
                </TableBody>
              </Table>
            </div>
          </CardContent>
        </Card>

        <Card>
          <CardHeader>
            <CardTitle>Scope Editor</CardTitle>
            <CardDescription>Create a new scope or update an existing one.</CardDescription>
          </CardHeader>
          <CardContent className="space-y-4">
            <form onSubmit={handleScopeSubmit} className="space-y-4">
              <div className="space-y-2">
                <Label htmlFor="scope-record">Target</Label>
                <Select value={selectedScopeId} onValueChange={handleTargetChange}>
                  <SelectTrigger id="scope-record">
                    <SelectValue placeholder="Select target scope" />
                  </SelectTrigger>
                  <SelectContent>
                    <SelectItem value={NEW_SCOPE}>Create new scope</SelectItem>
                    {scopes.map((row) => (
                      <SelectItem key={row.id} value={row.id}>
                        {row.deviceKey} / {row.scopeName || row.scopeKey}
                      </SelectItem>
                    ))}
                  </SelectContent>
                </Select>
              </div>

              <div className="space-y-2">
                <Label htmlFor="device-key">Device</Label>
                <Select value={selectedDeviceKey} onValueChange={setDeviceKey}>
                  <SelectTrigger id="device-key">
                    <SelectValue placeholder="Select a device" />
                  </SelectTrigger>
                  <SelectContent>
                    {devices.map((row) => (
                      <SelectItem key={row.id} value={row.deviceKey}>
                        {row.deviceKey} / {row.name}
                      </SelectItem>
                    ))}
                  </SelectContent>
                </Select>
              </div>

              <div className="grid gap-4 sm:grid-cols-2">
                <div className="space-y-2">
                  <Label htmlFor="scope-key">Scope Key</Label>
                  <Input id="scope-key" value={scopeKey} onChange={(event) => setScopeKey(event.target.value)} />
                </div>
                <div className="space-y-2">
                  <Label htmlFor="scope-name">Scope Name</Label>
                  <Input id="scope-name" value={scopeName} onChange={(event) => setScopeName(event.target.value)} />
                </div>
              </div>

              <div className="grid gap-4 sm:grid-cols-2">
                <div className="space-y-2">
                  <Label htmlFor="scope-type">Scope Type</Label>
                  <Select value={scopeType} onValueChange={setScopeType}>
                    <SelectTrigger id="scope-type">
                      <SelectValue placeholder="Select scope type" />
                    </SelectTrigger>
                    <SelectContent>
                      {scopeTypeOptions.map((option) => (
                        <SelectItem key={option} value={option}>
                          {option}
                        </SelectItem>
                      ))}
                    </SelectContent>
                  </Select>
                </div>
                <div className="space-y-2">
                  <Label htmlFor="scope-status">Status</Label>
                  <Select value={status} onValueChange={setStatus}>
                    <SelectTrigger id="scope-status">
                      <SelectValue placeholder="Select status" />
                    </SelectTrigger>
                    <SelectContent>
                      {statusOptions.map((option) => (
                        <SelectItem key={option} value={option}>
                          {option}
                        </SelectItem>
                      ))}
                    </SelectContent>
                  </Select>
                </div>
              </div>

              <div className="space-y-2">
                <Label htmlFor="owner-department-key">Owner Department</Label>
                <Input
                  id="owner-department-key"
                  value={ownerDepartmentKey}
                  onChange={(event) => setOwnerDepartmentKey(event.target.value)}
                  placeholder="e.g. optics, mechanical, controls"
                />
              </div>

              {message ? <p className="text-sm text-green-700">{message}</p> : null}

              <div className="flex gap-2">
                <Button type="submit" disabled={isSaving || !selectedDeviceKey || !scopeKey || !scopeName}>
                  {isSaving ? 'Saving...' : selectedScopeId === NEW_SCOPE ? 'Create Scope' : 'Update Scope'}
                </Button>
                <Button type="button" variant="outline" onClick={() => resetForm(selectedDeviceKey)}>
                  Reset
                </Button>
              </div>
            </form>
          </CardContent>
        </Card>
      </div>
    </div>
  )
}
