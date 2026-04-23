import { useState } from 'react'
import type { ChangeEvent } from 'react'
import { useSWRConfig } from 'swr'

import { SectionCard } from '../components/SectionCard'
import { useRoles } from '../hooks/useRoles'
import { useUsers } from '../hooks/useUsers'
import { useBootstrap } from '../hooks/useBootstrap'
import { useMasterData } from '../hooks/useMasterData'
import { useProcurementProjects } from '../hooks/useProcurementProjects'
import { approveUser, exportMasterDataCSV, importMasterDataCSV, refreshProcurementProjects, rejectUser } from '../lib/mockApi'
import type { RoleKey } from '../types'

type AdminPageProps = {
  initialTab?: 'overview' | 'users' | 'roles'
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
  const pendingUsers = users?.filter((user) => user.status === 'pending') ?? []

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
    <div className="page-grid">
      {initialTab === 'overview' ? (
        <SectionCard title="Admin / Platform" subtitle="Configuration and extension points visible to admins.">
          <dl className="definition-list">
            <div>
              <dt>Auth mode</dt>
              <dd>{data?.authMode}</dd>
            </div>
            <div>
              <dt>RBAC mode</dt>
              <dd>{data?.rbacMode}</dd>
            </div>
            <div>
              <dt>Storage mode</dt>
              <dd>{data?.storageMode}</dd>
            </div>
            <div>
              <dt>Capabilities</dt>
              <dd>{data?.capabilities.join(', ')}</dd>
            </div>
            <div>
              <dt>Project cache sync</dt>
              <dd>{latestProjectSync || 'not synced yet'}</dd>
            </div>
          </dl>
          <button type="button" className="primary-button" disabled={refreshingProjects} onClick={() => void handleProjectRefresh()}>
            {refreshingProjects ? 'Refreshing project cache...' : 'Refresh Procurement Project Cache'}
          </button>
        </SectionCard>
      ) : null}

      {initialTab !== 'roles' ? (
      <SectionCard title="Users / Approval" subtitle="Phase 5 onboarding: register, approve, reject, and attach app roles.">
        <div className="metric-grid">
          <article className="metric-card">
            <span>Pending</span>
            <strong>{pendingUsers.length}</strong>
          </article>
          <article className="metric-card">
            <span>Active</span>
            <strong>{users?.filter((user) => user.status === 'active').length ?? 0}</strong>
          </article>
          <article className="metric-card">
            <span>Rejected</span>
            <strong>{users?.filter((user) => user.status === 'rejected').length ?? 0}</strong>
          </article>
        </div>

        <table className="data-table">
          <thead>
            <tr>
              <th>User</th>
              <th>Status</th>
              <th>Roles</th>
              <th>Approval</th>
            </tr>
          </thead>
          <tbody>
            {users?.map((user) => (
              <tr key={user.id}>
                <td>
                  <strong>{user.displayName}</strong>
                  <div className="muted-copy">{user.email}</div>
                </td>
                <td>
                  {user.status}
                  {user.rejectionReason ? <div className="muted-copy">{user.rejectionReason}</div> : null}
                </td>
                <td>
                  <div className="role-chip-row">
                    {(roles ?? []).map((role) => {
                      const selected = (pendingRoles[user.id] ?? user.roles).includes(role.key)
                      return (
                        <label key={`${user.id}-${role.key}`} className="role-chip">
                          <input
                            type="checkbox"
                            checked={selected}
                            onChange={() => togglePendingRole(user.id, role.key)}
                            disabled={user.status === 'active' && user.roles.includes(role.key)}
                          />
                          <span>{role.key}</span>
                        </label>
                      )
                    })}
                  </div>
                </td>
                <td>
                  <div className="button-row">
                    <button
                      type="button"
                      className="primary-button"
                      onClick={async () => {
                        await approveUser(user.id, pendingRoles[user.id] ?? user.roles)
                        await Promise.all([mutate('users'), mutate('auth-session')])
                        setMessage(`Approved ${user.displayName}`)
                      }}
                    >
                      Approve
                    </button>
                    <button
                      type="button"
                      className="secondary-button"
                      onClick={async () => {
                        await rejectUser(user.id, 'Rejected by admin review')
                        await Promise.all([mutate('users'), mutate('auth-session')])
                        setMessage(`Rejected ${user.displayName}`)
                      }}
                    >
                      Reject
                    </button>
                  </div>
                </td>
              </tr>
            ))}
          </tbody>
        </table>
      </SectionCard>
      ) : null}

      {initialTab === 'overview' ? (
        <SectionCard title="Master Data" subtitle="Phase 1 master and import summary from the local database.">
        <div className="metric-grid">
          <article className="metric-card">
            <span>Items</span>
            <strong>{masterData?.itemCount ?? 0}</strong>
          </article>
          <article className="metric-card">
            <span>Suppliers</span>
            <strong>{masterData?.supplierCount ?? 0}</strong>
          </article>
          <article className="metric-card">
            <span>Aliases</span>
            <strong>{masterData?.aliasCount ?? 0}</strong>
          </article>
        </div>

        <div className="two-column">
          <button type="button" className="primary-button" onClick={() => void handleExport('items')}>
            Export Items CSV
          </button>
          <label>
            <span>Import Items CSV</span>
            <input type="file" accept=".csv,text/csv" onChange={(event) => void handleImport('items', event)} />
          </label>
        </div>

        <div className="two-column">
          <button type="button" className="primary-button" onClick={() => void handleExport('aliases')}>
            Export Aliases CSV
          </button>
          <label>
            <span>Import Aliases CSV</span>
            <input type="file" accept=".csv,text/csv" onChange={(event) => void handleImport('aliases', event)} />
          </label>
        </div>

        {message ? <p className="muted-copy">{message}</p> : null}

        <div className="two-column">
          <div>
            <h3 className="subheading">Recent Items</h3>
            <table className="data-table">
              <thead>
                <tr>
                  <th>Item</th>
                  <th>Description</th>
                  <th>Manufacturer</th>
                  <th>Category</th>
                  <th>Supplier</th>
                </tr>
              </thead>
              <tbody>
                {masterData?.recentItems.map((row) => (
                  <tr key={row.itemNumber}>
                    <td>{row.itemNumber}</td>
                    <td>{row.description}</td>
                    <td>{row.manufacturer}</td>
                    <td>{row.category}</td>
                    <td>{row.supplier}</td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
          <div>
            <h3 className="subheading">Categories</h3>
            <ul className="list">
              {masterData?.categories.map((category) => (
                <li key={category.key}>
                  {category.name} ({category.key})
                </li>
              ))}
            </ul>
            <h3 className="subheading">Suppliers</h3>
            <ul className="list">
              {masterData?.suppliers.map((supplier) => (
                <li key={supplier.id}>
                  {supplier.name} ({supplier.id})
                </li>
              ))}
            </ul>
            <h3 className="subheading">Recent Imports</h3>
            <ul className="list">
              {masterData?.recentImportFiles.map((file) => <li key={file}>{file}</li>)}
            </ul>
          </div>
        </div>

        <h3 className="subheading">Supplier Aliases</h3>
        <table className="data-table">
          <thead>
            <tr>
              <th>Supplier</th>
              <th>Canonical Item</th>
              <th>Alias</th>
              <th>Units / Order</th>
            </tr>
          </thead>
          <tbody>
            {masterData?.aliases.map((alias) => (
              <tr key={alias.id}>
                <td>{alias.supplierName}</td>
                <td>{alias.canonicalItemNumber}</td>
                <td>{alias.supplierItemNumber}</td>
                <td>{alias.unitsPerOrder}</td>
              </tr>
            ))}
          </tbody>
        </table>
        </SectionCard>
      ) : null}

      {initialTab !== 'users' ? (
      <SectionCard title="Role Catalog" subtitle="Current role-permission matrix anchor for route guards and admin approval.">
        <ul className="list">
          {roles?.map((role) => (
            <li key={role.key}>
              <strong>{role.key}</strong>: {role.description}
            </li>
          ))}
        </ul>
      </SectionCard>
      ) : null}
    </div>
  )
}
