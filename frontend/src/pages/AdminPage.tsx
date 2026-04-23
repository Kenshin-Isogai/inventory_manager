import { useState } from 'react'
import type { ChangeEvent } from 'react'
import { useSWRConfig } from 'swr'

import { Button } from '@/components/ui/button'
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card'
import { Badge } from '@/components/ui/badge'
import { Tabs, TabsContent, TabsList, TabsTrigger } from '@/components/ui/tabs'
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from '@/components/ui/table'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import { Checkbox } from '@/components/ui/checkbox'
import { Separator } from '@/components/ui/separator'

import { useRoles } from '@/hooks/useRoles'
import { useUsers } from '@/hooks/useUsers'
import { useBootstrap } from '@/hooks/useBootstrap'
import { useMasterData } from '@/hooks/useMasterData'
import { useProcurementProjects } from '@/hooks/useProcurementProjects'
import { approveUser, exportMasterDataCSV, importMasterDataCSV, refreshProcurementProjects, rejectUser } from '@/lib/mockApi'
import type { RoleKey } from '@/types'

type AdminPageProps = {
  initialTab?: 'overview' | 'users' | 'roles' | 'master-data'
}

export function AdminPage({ initialTab = 'overview' }: AdminPageProps) {
  const { data } = useBootstrap()
  const { data: masterData } = useMasterData()
  const { data: projects } = useProcurementProjects()
  const { data: users } = useUsers()
  const { data: roles } = useRoles()
  const { mutate } = useSWRConfig()
  const [message, setMessage] = useState('')
  const [refreshingProjects, setRefreshingProjects] = useState(false)
  const [pendingRoles, setPendingRoles] = useState<Record<string, RoleKey[]>>({})

  async function handleExport(exportType: 'items' | 'aliases') {
    const csv = await exportMasterDataCSV(exportType)
    const blob = new Blob([csv], { type: 'text/csv;charset=utf-8' })
    const url = URL.createObjectURL(blob)
    const link = document.createElement('a')
    link.href = url
    link.download = `${exportType}.csv`
    link.click()
    URL.revokeObjectURL(url)
    setMessage(`Exported ${exportType}.csv`)
  }

  async function handleImport(importType: 'items' | 'aliases', event: ChangeEvent<HTMLInputElement>) {
    const file = event.target.files?.[0]
    if (!file) {
      return
    }
    await importMasterDataCSV(importType, file)
    await Promise.all([mutate('master-data'), mutate('imports')])
    setMessage(`Imported ${file.name}`)
    event.target.value = ''
  }

  async function handleProjectRefresh() {
    setRefreshingProjects(true)
    try {
      const result = await refreshProcurementProjects()
      await mutate('procurement-projects')
      setMessage(`Refreshed project cache at ${result.syncedAt}`)
    } finally {
      setRefreshingProjects(false)
    }
  }

  const latestProjectSync = projects?.reduce((latest, project) => (project.syncedAt > latest ? project.syncedAt : latest), '') ?? ''
  const activeUsers = users?.filter((user) => user.status === 'active') ?? []
  const pendingUsers = users?.filter((user) => user.status === 'pending') ?? []
  const rejectedUsers = users?.filter((user) => user.status === 'rejected') ?? []

  function togglePendingRole(userId: string, role: RoleKey) {
    setPendingRoles((current) => {
      const selected = current[userId] ?? []
      return {
        ...current,
        [userId]: selected.includes(role) ? selected.filter((candidate) => candidate !== role) : [...selected, role],
      }
    })
  }

  return (
    <div className="p-6 space-y-6">
      <div>
        <h1 className="text-3xl font-bold tracking-tight">Admin</h1>
        <p className="text-muted-foreground mt-1">Configuration and platform management</p>
      </div>

      <Tabs defaultValue={initialTab} className="space-y-4">
        <TabsList className="grid w-full grid-cols-4">
          <TabsTrigger value="overview">Overview</TabsTrigger>
          <TabsTrigger value="users">Users</TabsTrigger>
          <TabsTrigger value="roles">Roles</TabsTrigger>
          <TabsTrigger value="master-data">Master Data</TabsTrigger>
        </TabsList>

        <TabsContent value="overview" className="space-y-4">
          <Card>
            <CardHeader>
              <CardTitle>Platform Configuration</CardTitle>
              <CardDescription>System settings and capabilities</CardDescription>
            </CardHeader>
            <CardContent className="space-y-6">
              <div className="grid grid-cols-2 gap-6">
                <div>
                  <p className="text-sm text-muted-foreground">Auth Mode</p>
                  <p className="font-medium">{data?.authMode || '-'}</p>
                </div>
                <div>
                  <p className="text-sm text-muted-foreground">RBAC Mode</p>
                  <p className="font-medium">{data?.rbacMode || '-'}</p>
                </div>
                <div>
                  <p className="text-sm text-muted-foreground">Storage Mode</p>
                  <p className="font-medium">{data?.storageMode || '-'}</p>
                </div>
                <div>
                  <p className="text-sm text-muted-foreground">Project Cache Sync</p>
                  <p className="font-medium">{latestProjectSync || 'Not synced yet'}</p>
                </div>
              </div>

              <Separator />

              <div>
                <p className="text-sm font-medium mb-3">Capabilities</p>
                <div className="flex flex-wrap gap-2">
                  {data?.capabilities.map((capability) => (
                    <Badge key={capability} variant="secondary">
                      {capability}
                    </Badge>
                  ))}
                </div>
              </div>

              <Button onClick={() => void handleProjectRefresh()} disabled={refreshingProjects}>
                {refreshingProjects ? 'Refreshing...' : 'Refresh Procurement Project Cache'}
              </Button>
            </CardContent>
          </Card>
        </TabsContent>

        <TabsContent value="users" className="space-y-4">
          <div className="grid grid-cols-3 gap-4 mb-6">
            <Card>
              <CardHeader className="pb-3">
                <CardTitle className="text-sm font-medium">Active Users</CardTitle>
              </CardHeader>
              <CardContent>
                <p className="text-2xl font-bold">{activeUsers.length}</p>
              </CardContent>
            </Card>
            <Card>
              <CardHeader className="pb-3">
                <CardTitle className="text-sm font-medium">Pending Approval</CardTitle>
              </CardHeader>
              <CardContent>
                <p className="text-2xl font-bold text-amber-600">{pendingUsers.length}</p>
              </CardContent>
            </Card>
            <Card>
              <CardHeader className="pb-3">
                <CardTitle className="text-sm font-medium">Rejected</CardTitle>
              </CardHeader>
              <CardContent>
                <p className="text-2xl font-bold text-red-600">{rejectedUsers.length}</p>
              </CardContent>
            </Card>
          </div>

          <Tabs defaultValue="active" className="space-y-4">
            <TabsList className="grid w-full grid-cols-3">
              <TabsTrigger value="active">Active</TabsTrigger>
              <TabsTrigger value="pending">Pending</TabsTrigger>
              <TabsTrigger value="rejected">Rejected</TabsTrigger>
            </TabsList>

            <TabsContent value="active">
              <Card>
                <CardHeader>
                  <CardTitle>Active Users</CardTitle>
                </CardHeader>
                <CardContent>
                  <div className="border rounded-lg overflow-hidden">
                    <Table>
                      <TableHeader>
                        <TableRow>
                          <TableHead>Name</TableHead>
                          <TableHead>Email</TableHead>
                          <TableHead>Roles</TableHead>
                          <TableHead>Last Login</TableHead>
                        </TableRow>
                      </TableHeader>
                      <TableBody>
                        {activeUsers.map((user) => (
                          <TableRow key={user.id}>
                            <TableCell className="font-medium">{user.displayName}</TableCell>
                            <TableCell className="text-sm text-muted-foreground">{user.email}</TableCell>
                            <TableCell>
                              <div className="flex flex-wrap gap-1">
                                {user.roles.map((role) => (
                                  <Badge key={role} variant="secondary" className="text-xs">
                                    {role}
                                  </Badge>
                                ))}
                              </div>
                            </TableCell>
                            <TableCell className="text-sm text-muted-foreground">{user.lastLoginAt}</TableCell>
                          </TableRow>
                        ))}
                      </TableBody>
                    </Table>
                  </div>
                </CardContent>
              </Card>
            </TabsContent>

            <TabsContent value="pending">
              <Card>
                <CardHeader>
                  <CardTitle>Pending Approval</CardTitle>
                </CardHeader>
                <CardContent>
                  <div className="border rounded-lg overflow-hidden">
                    <Table>
                      <TableHeader>
                        <TableRow>
                          <TableHead>Name</TableHead>
                          <TableHead>Email</TableHead>
                          <TableHead>Roles</TableHead>
                          <TableHead>Actions</TableHead>
                        </TableRow>
                      </TableHeader>
                      <TableBody>
                        {pendingUsers.map((user) => (
                          <TableRow key={user.id}>
                            <TableCell className="font-medium">{user.displayName}</TableCell>
                            <TableCell className="text-sm text-muted-foreground">{user.email}</TableCell>
                            <TableCell>
                              <div className="flex flex-wrap gap-2">
                                {(roles ?? []).map((role) => {
                                  const selected = (pendingRoles[user.id] ?? user.roles).includes(role.key)
                                  return (
                                    <label key={`${user.id}-${role.key}`} className="flex items-center space-x-2">
                                      <Checkbox
                                        checked={selected}
                                        onCheckedChange={() => togglePendingRole(user.id, role.key)}
                                      />
                                      <span className="text-sm">{role.key}</span>
                                    </label>
                                  )
                                })}
                              </div>
                            </TableCell>
                            <TableCell className="space-x-2">
                              <Button
                                size="sm"
                                onClick={async () => {
                                  await approveUser(user.id, pendingRoles[user.id] ?? user.roles)
                                  await Promise.all([mutate('users'), mutate('auth-session')])
                                  setMessage(`Approved ${user.displayName}`)
                                }}
                              >
                                Approve
                              </Button>
                              <Button
                                size="sm"
                                variant="destructive"
                                onClick={async () => {
                                  await rejectUser(user.id, 'Rejected by admin review')
                                  await Promise.all([mutate('users'), mutate('auth-session')])
                                  setMessage(`Rejected ${user.displayName}`)
                                }}
                              >
                                Reject
                              </Button>
                            </TableCell>
                          </TableRow>
                        ))}
                      </TableBody>
                    </Table>
                  </div>
                </CardContent>
              </Card>
            </TabsContent>

            <TabsContent value="rejected">
              <Card>
                <CardHeader>
                  <CardTitle>Rejected Users</CardTitle>
                </CardHeader>
                <CardContent>
                  <div className="border rounded-lg overflow-hidden">
                    <Table>
                      <TableHeader>
                        <TableRow>
                          <TableHead>Name</TableHead>
                          <TableHead>Email</TableHead>
                          <TableHead>Reason</TableHead>
                        </TableRow>
                      </TableHeader>
                      <TableBody>
                        {rejectedUsers.map((user) => (
                          <TableRow key={user.id}>
                            <TableCell className="font-medium">{user.displayName}</TableCell>
                            <TableCell className="text-sm text-muted-foreground">{user.email}</TableCell>
                            <TableCell className="text-sm text-destructive">{user.rejectionReason || 'No reason provided'}</TableCell>
                          </TableRow>
                        ))}
                      </TableBody>
                    </Table>
                  </div>
                </CardContent>
              </Card>
            </TabsContent>
          </Tabs>

          {message && (
            <Card className="border-green-500">
              <CardContent className="pt-6">
                <p className="text-green-600">{message}</p>
              </CardContent>
            </Card>
          )}
        </TabsContent>

        <TabsContent value="roles">
          <Card>
            <CardHeader>
              <CardTitle>Role Catalog</CardTitle>
              <CardDescription>Current role-permission matrix for route guards and admin approval</CardDescription>
            </CardHeader>
            <CardContent>
              <div className="space-y-4">
                {roles?.map((role) => (
                  <div key={role.key} className="flex items-start justify-between border-b pb-4 last:border-b-0">
                    <div>
                      <Badge>{role.key}</Badge>
                      <p className="text-sm text-muted-foreground mt-1">{role.description}</p>
                    </div>
                  </div>
                ))}
              </div>
            </CardContent>
          </Card>
        </TabsContent>

        <TabsContent value="master-data" className="space-y-4">
          <div className="grid grid-cols-3 gap-4 mb-6">
            <Card>
              <CardHeader className="pb-3">
                <CardTitle className="text-sm font-medium">Items</CardTitle>
              </CardHeader>
              <CardContent>
                <p className="text-2xl font-bold">{masterData?.itemCount ?? 0}</p>
              </CardContent>
            </Card>
            <Card>
              <CardHeader className="pb-3">
                <CardTitle className="text-sm font-medium">Suppliers</CardTitle>
              </CardHeader>
              <CardContent>
                <p className="text-2xl font-bold">{masterData?.supplierCount ?? 0}</p>
              </CardContent>
            </Card>
            <Card>
              <CardHeader className="pb-3">
                <CardTitle className="text-sm font-medium">Aliases</CardTitle>
              </CardHeader>
              <CardContent>
                <p className="text-2xl font-bold">{masterData?.aliasCount ?? 0}</p>
              </CardContent>
            </Card>
          </div>

          <Tabs defaultValue="items" className="space-y-4">
            <TabsList className="grid w-full grid-cols-3">
              <TabsTrigger value="items">Items</TabsTrigger>
              <TabsTrigger value="suppliers">Suppliers</TabsTrigger>
              <TabsTrigger value="aliases">Aliases</TabsTrigger>
            </TabsList>

            <TabsContent value="items">
              <Card>
                <CardHeader>
                  <CardTitle>Items Management</CardTitle>
                </CardHeader>
                <CardContent className="space-y-4">
                  <div className="flex gap-2">
                    <Button onClick={() => void handleExport('items')}>Export CSV</Button>
                    <Label className="flex items-center cursor-pointer">
                      <Input
                        type="file"
                        accept=".csv,text/csv"
                        onChange={(event) => void handleImport('items', event)}
                        className="hidden"
                      />
                      <Button variant="outline" type="button">Import CSV</Button>
                    </Label>
                  </div>

                  {message && <p className="text-sm text-green-600">{message}</p>}

                  <div>
                    <h3 className="font-semibold mb-3">Recent Items</h3>
                    <div className="border rounded-lg overflow-hidden">
                      <Table>
                        <TableHeader>
                          <TableRow>
                            <TableHead>Item</TableHead>
                            <TableHead>Description</TableHead>
                            <TableHead>Manufacturer</TableHead>
                            <TableHead>Category</TableHead>
                            <TableHead>Supplier</TableHead>
                          </TableRow>
                        </TableHeader>
                        <TableBody>
                          {masterData?.recentItems.map((row) => (
                            <TableRow key={row.itemNumber}>
                              <TableCell className="font-mono text-sm">{row.itemNumber}</TableCell>
                              <TableCell className="text-sm">{row.description}</TableCell>
                              <TableCell className="text-sm">{row.manufacturer}</TableCell>
                              <TableCell className="text-sm">{row.category}</TableCell>
                              <TableCell className="text-sm">{row.supplier}</TableCell>
                            </TableRow>
                          ))}
                        </TableBody>
                      </Table>
                    </div>
                  </div>

                  <div>
                    <h3 className="font-semibold mb-3">Categories</h3>
                    <div className="flex flex-wrap gap-2">
                      {masterData?.categories.map((category) => (
                        <Badge key={category.key} variant="secondary">
                          {category.name} ({category.key})
                        </Badge>
                      ))}
                    </div>
                  </div>
                </CardContent>
              </Card>
            </TabsContent>

            <TabsContent value="suppliers">
              <Card>
                <CardHeader>
                  <CardTitle>Suppliers</CardTitle>
                </CardHeader>
                <CardContent>
                  <div className="space-y-2">
                    {masterData?.suppliers.map((supplier) => (
                      <div key={supplier.id} className="flex items-center justify-between border-b pb-2 last:border-b-0">
                        <div>
                          <p className="font-medium">{supplier.name}</p>
                          <p className="text-sm text-muted-foreground font-mono">{supplier.id}</p>
                        </div>
                      </div>
                    ))}
                  </div>
                </CardContent>
              </Card>

              <Card className="mt-4">
                <CardHeader>
                  <CardTitle>Recent Imports</CardTitle>
                </CardHeader>
                <CardContent>
                  <ul className="space-y-2">
                    {masterData?.recentImportFiles.map((file) => (
                      <li key={file} className="text-sm font-mono text-muted-foreground">
                        {file}
                      </li>
                    ))}
                  </ul>
                </CardContent>
              </Card>
            </TabsContent>

            <TabsContent value="aliases">
              <Card>
                <CardHeader>
                  <CardTitle>Supplier Aliases</CardTitle>
                </CardHeader>
                <CardContent>
                  <div className="flex gap-2 mb-4">
                    <Button onClick={() => void handleExport('aliases')}>Export CSV</Button>
                    <Label className="flex items-center cursor-pointer">
                      <Input
                        type="file"
                        accept=".csv,text/csv"
                        onChange={(event) => void handleImport('aliases', event)}
                        className="hidden"
                      />
                      <Button variant="outline" type="button">Import CSV</Button>
                    </Label>
                  </div>

                  {message && <p className="text-sm text-green-600">{message}</p>}

                  <div className="border rounded-lg overflow-hidden">
                    <Table>
                      <TableHeader>
                        <TableRow>
                          <TableHead>Supplier</TableHead>
                          <TableHead>Canonical Item</TableHead>
                          <TableHead>Supplier Alias</TableHead>
                          <TableHead>Units/Order</TableHead>
                        </TableRow>
                      </TableHeader>
                      <TableBody>
                        {masterData?.aliases.map((alias) => (
                          <TableRow key={alias.id}>
                            <TableCell className="text-sm">{alias.supplierName}</TableCell>
                            <TableCell className="font-mono text-sm">{alias.canonicalItemNumber}</TableCell>
                            <TableCell className="font-mono text-sm">{alias.supplierItemNumber}</TableCell>
                            <TableCell className="text-sm">{alias.unitsPerOrder}</TableCell>
                          </TableRow>
                        ))}
                      </TableBody>
                    </Table>
                  </div>
                </CardContent>
              </Card>
            </TabsContent>
          </Tabs>
        </TabsContent>
      </Tabs>
    </div>
  )
}
