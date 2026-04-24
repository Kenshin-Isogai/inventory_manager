import { useState, useMemo } from 'react'
import type { FormEvent } from 'react'
import { useSWRConfig } from 'swr'
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card'
import { Badge } from '@/components/ui/badge'
import { useDashboard } from '@/hooks/useDashboard'
import { useDeviceScopes } from '@/hooks/useDeviceScopes'
import { useDevices } from '@/hooks/useDevices'
import { useScopeSystems } from '@/hooks/useScopeSystems'
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
import { AlertCircle, ChevronRight, Layers, Package, Box, MapPin, ClipboardList, Info } from 'lucide-react'

const NEW_SCOPE = '__new__'
const ROOT_SCOPE = '__root__'
const scopeTypeOptions = ['system', 'assembly', 'module', 'area', 'work_package']
const statusOptions = ['active', 'inactive']

function getScopeIcon(type: string) {
  switch (type) {
    case 'system': return <Layers className="w-4 h-4 text-blue-500" />
    case 'assembly': return <Package className="w-4 h-4 text-orange-500" />
    case 'module': return <Box className="w-4 h-4 text-green-500" />
    case 'area': return <MapPin className="w-4 h-4 text-purple-500" />
    case 'work_package': return <ClipboardList className="w-4 h-4 text-gray-500" />
    default: return <Info className="w-4 h-4" />
  }
}

export function OperatorDashboardPage() {
  const { data } = useDashboard()
  const { data: deviceScopeData } = useDeviceScopes()
  const { data: deviceData } = useDevices()
  const { data: scopeSystemData } = useScopeSystems()
  const { mutate } = useSWRConfig()
  const [selectedScopeId, setSelectedScopeId] = useState(NEW_SCOPE)
  const [deviceKey, setDeviceKey] = useState('')
  const [parentScopeId, setParentScopeId] = useState('')
  const [systemKey, setSystemKey] = useState('')
  const [scopeKey, setScopeKey] = useState('')
  const [scopeName, setScopeName] = useState('')
  const [scopeType, setScopeType] = useState('system')
  const [ownerDepartmentKey, setOwnerDepartmentKey] = useState('')
  const [status, setStatus] = useState('active')
  const [message, setMessage] = useState('')
  const [isSaving, setIsSaving] = useState(false)

  const scopes = useMemo(() => deviceScopeData?.rows ?? [], [deviceScopeData?.rows])
  const devices = useMemo(
    () => deviceData?.rows.filter((row) => row.status !== 'inactive') ?? [],
    [deviceData?.rows],
  )
  const scopeSystems = useMemo(
    () => scopeSystemData?.rows.filter((row) => row.status !== 'inactive') ?? [],
    [scopeSystemData?.rows],
  )
  const selectedDeviceKey = deviceKey || devices[0]?.deviceKey || ''
  const selectedParentScope = scopes.find((row) => row.id === parentScopeId)
  const isSystemScope = scopeType === 'system'
  const effectiveSystemKey = isSystemScope ? systemKey : selectedParentScope?.systemKey || ''
  const effectiveSystemName =
    scopeSystems.find((row) => row.key === effectiveSystemKey)?.name || selectedParentScope?.systemName || ''
  
  const parentScopeOptions = useMemo(() => {
    return scopes
      .filter((row) => row.deviceKey === selectedDeviceKey && row.id !== selectedScopeId)
      .sort((left, right) => `${left.parentScopeKey}/${left.scopeKey}`.localeCompare(`${right.parentScopeKey}/${right.scopeKey}`))
  }, [scopes, selectedDeviceKey, selectedScopeId])

  // Function to get full path for a scope
  const getScopePath = (scopeId: string): string[] => {
    const scope = scopes.find(s => s.id === scopeId)
    if (!scope) return []
    if (!scope.parentScopeId) return [scope.scopeName || scope.scopeKey]
    return [...getScopePath(scope.parentScopeId), scope.scopeName || scope.scopeKey]
  }

  function resetForm(nextDeviceKey?: string) {
    setSelectedScopeId(NEW_SCOPE)
    setDeviceKey(nextDeviceKey ?? devices[0]?.deviceKey ?? '')
    setParentScopeId('')
    setSystemKey('')
    setScopeKey('')
    setScopeName('')
    setScopeType('system')
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
    setParentScopeId(selected.parentScopeId)
    setSystemKey(selected.systemKey)
    setScopeKey(selected.scopeKey)
    setScopeName(selected.scopeName)
    setScopeType(selected.scopeType || 'assembly')
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
        parentScopeId: isSystemScope ? '' : parentScopeId,
        systemKey: effectiveSystemKey,
        scopeKey,
        scopeName,
        scopeType,
        ownerDepartmentKey: isSystemScope ? ownerDepartmentKey || effectiveSystemKey : ownerDepartmentKey,
        status,
      })
      await Promise.all([mutate('device-scopes'), mutate('devices'), mutate('scope-systems')])
      setMessage(selectedScopeId === NEW_SCOPE ? 'Scope created.' : 'Scope updated.')
      if (selectedScopeId === NEW_SCOPE) {
        resetForm(selectedDeviceKey)
      }
    } catch (err) {
      setMessage(`Error: ${err instanceof Error ? err.message : String(err)}`)
    } finally {
      setIsSaving(false)
    }
  }

  return (
    <div className="space-y-6 p-6">
      <div className="space-y-2">
        <h1 className="text-3xl font-bold tracking-tight">Operator Dashboard</h1>
        <p className="text-muted-foreground">Requirements, shortages, and device hierarchy.</p>
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
            <CardDescription>Defined logical boundaries for each device in a hierarchical tree.</CardDescription>
          </CardHeader>
          <CardContent>
            <div className="overflow-x-auto">
              <Table>
                <TableHeader>
                  <TableRow>
                    <TableHead>Device</TableHead>
                    <TableHead>Path / Breadcrumb</TableHead>
                    <TableHead>Type</TableHead>
                    <TableHead>Department</TableHead>
                    <TableHead>Status</TableHead>
                  </TableRow>
                </TableHeader>
                <TableBody>
                  {scopes.map((row) => {
                    const path = getScopePath(row.id)
                    return (
                      <TableRow
                        key={row.id}
                        className={`cursor-pointer transition-colors ${selectedScopeId === row.id ? 'bg-primary/5' : ''}`}
                        onClick={() => handleTargetChange(row.id)}
                      >
                        <TableCell className="font-mono text-sm">{row.deviceKey}</TableCell>
                        <TableCell>
                          <div className="flex items-center gap-1.5 flex-wrap">
                            {path.map((segment, idx) => (
                              <span key={idx} className="flex items-center gap-1.5">
                                {idx > 0 && <ChevronRight className="w-3 h-3 text-muted-foreground" />}
                                <span className={idx === path.length - 1 ? 'font-medium' : 'text-xs text-muted-foreground'}>
                                  {segment}
                                </span>
                              </span>
                            ))}
                          </div>
                        </TableCell>
                        <TableCell>
                          <div className="flex items-center gap-2 text-xs">
                            {getScopeIcon(row.scopeType)}
                            <span className="capitalize">{row.scopeType.replace('_', ' ')}</span>
                          </div>
                        </TableCell>
                        <TableCell className="text-xs font-mono">{row.ownerDepartmentKey || '-'}</TableCell>
                        <TableCell>
                          <Badge variant={row.status === 'active' ? 'default' : 'secondary'} className="text-[10px] uppercase">
                            {row.status}
                          </Badge>
                        </TableCell>
                      </TableRow>
                    )
                  })}
                </TableBody>
              </Table>
            </div>
          </CardContent>
        </Card>

        <Card className={selectedScopeId === NEW_SCOPE ? 'border-primary/50 shadow-md' : ''}>
          <CardHeader>
            <CardTitle>{selectedScopeId === NEW_SCOPE ? 'New Node' : 'Edit Node'}</CardTitle>
            <CardDescription>Manage the hierarchy entry for the selected device.</CardDescription>
          </CardHeader>
          <CardContent>
            <form onSubmit={handleScopeSubmit} className="space-y-4">
              <div className="space-y-2">
                <Label htmlFor="scope-record">Selection</Label>
                <Select value={selectedScopeId} onValueChange={handleTargetChange}>
                  <SelectTrigger id="scope-record">
                    <SelectValue placeholder="Select target scope" />
                  </SelectTrigger>
                  <SelectContent>
                    <SelectItem value={NEW_SCOPE}>+ Create new node</SelectItem>
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
                <Select
                  value={selectedDeviceKey}
                  onValueChange={(value) => {
                    setDeviceKey(value)
                    setParentScopeId('')
                    setSystemKey('')
                  }}
                >
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

              <div className="space-y-2">
                <Label htmlFor="scope-type">Scope Level</Label>
                <Select
                  value={scopeType}
                  onValueChange={(value) => {
                    setScopeType(value)
                    if (value === 'system') {
                      setParentScopeId('')
                      if (effectiveSystemKey) {
                        setSystemKey(effectiveSystemKey)
                      }
                    } else if (parentScopeId) {
                      const parent = scopes.find((row) => row.id === parentScopeId)
                      setSystemKey(parent?.systemKey || '')
                    } else {
                      setSystemKey('')
                    }
                  }}
                >
                  <SelectTrigger id="scope-type">
                    <SelectValue placeholder="Select scope type" />
                  </SelectTrigger>
                  <SelectContent>
                    {scopeTypeOptions.map((opt) => (
                      <SelectItem key={opt} value={opt}>
                        <div className="flex items-center gap-2">
                          {getScopeIcon(opt)}
                          <span className="capitalize">{opt.replace('_', ' ')}</span>
                        </div>
                      </SelectItem>
                    ))}
                  </SelectContent>
                </Select>
              </div>

              <div className="space-y-2">
                <Label htmlFor="parent-scope">Parent Node</Label>
                <Select
                  value={parentScopeId || ROOT_SCOPE}
                  disabled={isSystemScope}
                  onValueChange={(value) => {
                    if (isSystemScope) {
                      setParentScopeId('')
                      return
                    }
                    const nextParentId = value === ROOT_SCOPE ? '' : value
                    const parent = scopes.find((row) => row.id === nextParentId)
                    setParentScopeId(nextParentId)
                    if (parent?.systemKey) {
                      setSystemKey(parent.systemKey)
                    } else {
                      setSystemKey('')
                    }
                  }}
                >
                  <SelectTrigger id="parent-scope">
                    <SelectValue placeholder="Top-level scope" />
                  </SelectTrigger>
                  <SelectContent>
                    <SelectItem value={ROOT_SCOPE}>Root (System nodes only)</SelectItem>
                    {parentScopeOptions.map((row) => (
                      <SelectItem key={row.id} value={row.id}>
                        {row.scopeName || row.scopeKey}
                      </SelectItem>
                    ))}
                  </SelectContent>
                </Select>
              </div>

              <div className="space-y-2">
                <Label htmlFor="system-key">Engineering System</Label>
                {isSystemScope ? (
                  <Select
                    value={systemKey}
                    onValueChange={(value) => {
                      const selectedSystem = scopeSystems.find((row) => row.key === value)
                      const currentSystemName = scopeSystems.find((row) => row.key === systemKey)?.name || ''
                      setSystemKey(value)
                      setParentScopeId('')
                      setScopeKey(value)
                      if (!scopeName || scopeName === currentSystemName) {
                        setScopeName(selectedSystem?.name || '')
                      }
                      if (!ownerDepartmentKey || ownerDepartmentKey === systemKey) {
                        setOwnerDepartmentKey(value)
                      }
                    }}
                  >
                    <SelectTrigger id="system-key">
                      <SelectValue placeholder="Select a system" />
                    </SelectTrigger>
                    <SelectContent>
                      {scopeSystems.map((row) => (
                        <SelectItem key={row.key} value={row.key}>
                          {row.name} ({row.key})
                        </SelectItem>
                      ))}
                    </SelectContent>
                  </Select>
                ) : (
                  <div className="flex items-center gap-2 p-2 border rounded-md bg-muted/30 text-sm">
                    <Layers className="w-4 h-4 text-muted-foreground" />
                    <span>{effectiveSystemName ? `${effectiveSystemName} (${effectiveSystemKey})` : 'Inherited from parent'}</span>
                  </div>
                )}
              </div>

              <div className="grid gap-4 sm:grid-cols-2">
                <div className="space-y-2">
                  <Label htmlFor="scope-key">Scope Key</Label>
                  <Input
                    id="scope-key"
                    value={scopeKey}
                    onChange={(event) => setScopeKey(event.target.value)}
                    disabled={isSystemScope}
                    placeholder={isSystemScope ? 'Matches system key' : 'e.g. power-box'}
                  />
                </div>
                <div className="space-y-2">
                  <Label htmlFor="scope-name">Display Name</Label>
                  <Input id="scope-name" value={scopeName} onChange={(event) => setScopeName(event.target.value)} placeholder="e.g. Power Control Box" />
                </div>
              </div>

              <div className="space-y-2">
                <Label htmlFor="owner">Owner Department</Label>
                <Input
                  id="owner"
                  value={ownerDepartmentKey}
                  onChange={(event) => setOwnerDepartmentKey(event.target.value)}
                  placeholder="e.g. optics, mechanical, controls"
                />
              </div>

              <div className="space-y-2">
                <Label htmlFor="status">Status</Label>
                <Select value={status} onValueChange={setStatus}>
                  <SelectTrigger id="status">
                    <SelectValue />
                  </SelectTrigger>
                  <SelectContent>
                    {statusOptions.map((opt) => (
                      <SelectItem key={opt} value={opt}>
                        {opt}
                      </SelectItem>
                    ))}
                  </SelectContent>
                </Select>
              </div>

              {message ? (
                <div
                  className={`flex items-start gap-2 rounded-md border px-3 py-2 text-xs ${
                    message.startsWith('Error')
                      ? 'border-destructive/30 bg-destructive/10 text-destructive'
                      : 'border-primary/20 bg-primary/5 text-foreground'
                  }`}
                >
                  <AlertCircle className="h-4 w-4 shrink-0" />
                  <p>{message}</p>
                </div>
              ) : null}

              <div className="flex gap-2 pt-2">
                <Button
                  type="submit"
                  className="flex-1"
                  disabled={
                    isSaving ||
                    !selectedDeviceKey ||
                    !scopeKey ||
                    !scopeName ||
                    !effectiveSystemKey ||
                    (!isSystemScope && !parentScopeId)
                  }
                >
                  {isSaving ? 'Saving...' : selectedScopeId === NEW_SCOPE ? 'Create Node' : 'Update Node'}
                </Button>
                <Button type="button" variant="outline" onClick={() => resetForm(selectedDeviceKey)}>
                  Cancel
                </Button>
              </div>
            </form>
          </CardContent>
        </Card>
      </div>
    </div>
  )
}
