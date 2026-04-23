import { useState } from 'react'
import type { FormEvent } from 'react'
import { useNavigate } from 'react-router-dom'
import { useSWRConfig } from 'swr'

import { Button } from '@/components/ui/button'
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import { Badge } from '@/components/ui/badge'
import { Dialog, DialogContent, DialogDescription, DialogHeader, DialogTitle, DialogTrigger } from '@/components/ui/dialog'
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from '@/components/ui/select'
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from '@/components/ui/table'

import { useProcurementBudgetCategories } from '@/hooks/useProcurementBudgetCategories'
import { useProcurementProjects } from '@/hooks/useProcurementProjects'
import { useProcurementRequests } from '@/hooks/useProcurementRequests'
import {
  createProcurementRequest,
  refreshProcurementBudgetCategories,
  refreshProcurementProjects,
} from '@/lib/mockApi'

export function ProcurementPage() {
  const navigate = useNavigate()
  const { mutate } = useSWRConfig()
  const { data: requests } = useProcurementRequests()
  const { data: projects } = useProcurementProjects()

  const [title, setTitle] = useState('New shortage follow-up')
  const [projectId, setProjectId] = useState('proj-er2-upgrade')
  const [budgetCategoryId, setBudgetCategoryId] = useState('budget-er2-material')
  const { data: budgetCategories } = useProcurementBudgetCategories(projectId)
  const selectedBudgetCategoryId =
    budgetCategories?.some((budget) => budget.id === budgetCategoryId) ? budgetCategoryId : budgetCategories?.[0]?.id || budgetCategoryId
  const [submitting, setSubmitting] = useState(false)
  const [refreshingProjects, setRefreshingProjects] = useState(false)
  const [refreshingBudgets, setRefreshingBudgets] = useState(false)
  const [submitError, setSubmitError] = useState('')
  const [dialogOpen, setDialogOpen] = useState(false)

  async function handleSubmit(event: FormEvent<HTMLFormElement>) {
    event.preventDefault()
    setSubmitting(true)
    setSubmitError('')
    try {
      const result = await createProcurementRequest({
        title,
        projectId,
        budgetCategoryId: selectedBudgetCategoryId,
        supplierId: 'sup-misumi',
        quotationId: 'quote-001',
        sourceType: 'manual',
        createdBy: 'local-user',
        lines: [
          {
            itemId: 'item-er2',
            quotationLineId: 'quote-line-001',
            requestedQuantity: 4,
            deliveryLocation: 'Tokyo Assembly',
            accountingCategory: 'parts',
            note: 'Created from Procurement page form',
          },
        ],
      })
      await mutate('procurement-requests')
      if ((result as { id?: string } | undefined)?.id) {
        setDialogOpen(false)
        navigate(`/app/procurement/requests/${(result as { id: string }).id}`)
      }
    } catch (error) {
      setSubmitError(error instanceof Error ? error.message : 'Failed to create request')
    } finally {
      setSubmitting(false)
    }
  }

  async function handleRefreshProjects() {
    setRefreshingProjects(true)
    setSubmitError('')
    try {
      await refreshProcurementProjects()
      await mutate('procurement-projects')
    } catch (error) {
      setSubmitError(error instanceof Error ? error.message : 'Failed to refresh project cache')
    } finally {
      setRefreshingProjects(false)
    }
  }

  async function handleRefreshBudgets() {
    setRefreshingBudgets(true)
    setSubmitError('')
    try {
      await refreshProcurementBudgetCategories(projectId)
      await mutate(['procurement-budget-categories', projectId])
    } catch (error) {
      setSubmitError(error instanceof Error ? error.message : 'Failed to refresh budget categories')
    } finally {
      setRefreshingBudgets(false)
    }
  }

  const getStatusVariant = (status: string): 'default' | 'secondary' | 'destructive' | 'outline' => {
    switch (status) {
      case 'draft':
        return 'secondary'
      case 'submitted':
        return 'default'
      case 'ordered':
        return 'default'
      case 'partially_received':
        return 'outline'
      case 'received':
        return 'default'
      default:
        return 'default'
    }
  }

  return (
    <div className="p-6 space-y-6">
      <div>
        <h1 className="text-3xl font-bold tracking-tight">Procurement Requests</h1>
        <p className="text-muted-foreground mt-1">Track requests, projects, and budget categories.</p>
      </div>

      <Card>
        <CardHeader className="flex flex-row items-center justify-between">
          <div>
            <CardTitle>Requests</CardTitle>
            <CardDescription>All procurement requests</CardDescription>
          </div>
          <Dialog open={dialogOpen} onOpenChange={setDialogOpen}>
            <DialogTrigger asChild>
              <Button>Create Request</Button>
            </DialogTrigger>
            <DialogContent className="max-w-md">
              <DialogHeader>
                <DialogTitle>Create Procurement Request</DialogTitle>
                <DialogDescription>Create a new procurement request with project and budget details.</DialogDescription>
              </DialogHeader>
              <form onSubmit={handleSubmit} className="space-y-4">
                <div className="space-y-2">
                  <Label htmlFor="title">Title</Label>
                  <Input
                    id="title"
                    value={title}
                    onChange={(event) => setTitle(event.target.value)}
                    placeholder="Request title"
                  />
                </div>
                <div className="space-y-2">
                  <Label htmlFor="project">Project</Label>
                  <Select value={projectId} onValueChange={(value) => {
                    setProjectId(value)
                    setBudgetCategoryId('')
                  }}>
                    <SelectTrigger id="project">
                      <SelectValue />
                    </SelectTrigger>
                    <SelectContent>
                      {projects?.map((project) => (
                        <SelectItem key={project.id} value={project.id}>
                          {project.name}
                        </SelectItem>
                      ))}
                    </SelectContent>
                  </Select>
                </div>
                <div className="space-y-2">
                  <Label htmlFor="budget">Budget Category</Label>
                  <Select value={selectedBudgetCategoryId} onValueChange={(value) => setBudgetCategoryId(value)}>
                    <SelectTrigger id="budget">
                      <SelectValue />
                    </SelectTrigger>
                    <SelectContent>
                      {budgetCategories?.map((budget) => (
                        <SelectItem key={budget.id} value={budget.id}>
                          {budget.name}
                        </SelectItem>
                      ))}
                    </SelectContent>
                  </Select>
                </div>
                <div className="space-y-2">
                  <Button type="button" variant="outline" onClick={handleRefreshProjects} disabled={refreshingProjects} className="w-full">
                    {refreshingProjects ? 'Refreshing...' : 'Refresh Projects'}
                  </Button>
                  <Button type="button" variant="outline" onClick={handleRefreshBudgets} disabled={refreshingBudgets} className="w-full">
                    {refreshingBudgets ? 'Refreshing...' : 'Refresh Budget Categories'}
                  </Button>
                </div>
                {submitError && <p className="text-sm text-destructive">{submitError}</p>}
                <Button type="submit" disabled={submitting} className="w-full">
                  {submitting ? 'Creating...' : 'Create Request'}
                </Button>
              </form>
            </DialogContent>
          </Dialog>
        </CardHeader>
        <CardContent>
          <div className="border rounded-lg overflow-hidden">
            <Table>
              <TableHeader>
                <TableRow>
                  <TableHead>Batch</TableHead>
                  <TableHead>Title</TableHead>
                  <TableHead>Project</TableHead>
                  <TableHead>Supplier</TableHead>
                  <TableHead>Status</TableHead>
                  <TableHead>Source</TableHead>
                  <TableHead>Dispatch</TableHead>
                </TableRow>
              </TableHeader>
              <TableBody>
                {requests?.rows.map((row) => (
                  <TableRow
                    key={row.id}
                    onClick={() => navigate(`/app/procurement/requests/${row.id}`)}
                    className="cursor-pointer hover:bg-muted"
                  >
                    <TableCell className="font-mono text-sm">{row.batchNumber}</TableCell>
                    <TableCell>{row.title}</TableCell>
                    <TableCell>{row.projectName}</TableCell>
                    <TableCell>{row.supplierName}</TableCell>
                    <TableCell>
                      <Badge variant={getStatusVariant(row.normalizedStatus)}>
                        {row.normalizedStatus}
                      </Badge>
                    </TableCell>
                    <TableCell className="text-sm text-muted-foreground">{row.sourceType}</TableCell>
                    <TableCell className="text-sm text-muted-foreground">{row.dispatchStatus}</TableCell>
                  </TableRow>
                ))}
              </TableBody>
            </Table>
          </div>
        </CardContent>
      </Card>
    </div>
  )
}
