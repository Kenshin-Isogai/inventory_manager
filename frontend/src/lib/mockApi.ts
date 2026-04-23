import type {
  AppUserSummary,
  AuthSessionResponse,
  BootstrapResponse,
  DashboardResponse,
  ImportHistoryResponse,
  InventoryOverviewResponse,
  MasterSyncRunEntry,
  MasterDataSummaryResponse,
  ProcurementRequestCreateInput,
  ProcurementRequestDetailResponse,
  ProcurementRequestListResponse,
  ProjectSummary,
  OCRJobDetailResponse,
  OCRJobListResponse,
  OCRLineAssistInput,
  OCRLineAssistSuggestion,
  OCRProcurementDraftCreateResult,
  OCRReviewUpdateInput,
  OCRRetryResult,
  OCRRegisterItemInput,
  ReservationListResponse,
  ShortageListResponse,
  BudgetCategorySummary,
  MasterSyncResult,
  ProcurementReconcileResult,
  ProcurementSubmitResult,
  RegistrationInput,
  RoleSummary,
  WebhookEventEntry,
} from '../types'
import { config } from './config'
import { authorizationHeaders, getStoredToken } from './auth'

function allowMockApiFallback() {
  return import.meta.env.DEV || ['localhost', '127.0.0.1'].includes(window.location.hostname)
}

const bootstrap: BootstrapResponse = {
  frontendBaseUrl: 'http://localhost:5173',
  authMode: 'none',
  authProvider: 'local',
  rbacMode: 'dry_run',
  storageMode: 'local',
  capabilities: [
    'local-mode',
    'mock-read-models',
    'phase0-shell',
    'cloud-ready-config',
  ],
}

let currentSession: AuthSessionResponse = {
  authenticated: false,
  authMode: 'none',
  authProvider: 'local',
  rbacMode: 'dry_run',
  user: {
    authenticated: false,
    userId: '',
    email: '',
    displayName: '',
    status: 'anonymous',
    roles: [],
    provider: 'local',
    subject: '',
    emailVerified: false,
    registrationNeeded: false,
    rejectionReason: '',
  },
}

const users: AppUserSummary[] = [
  {
    id: '00000000-0000-0000-0000-000000000001',
    email: 'admin@example.local',
    displayName: 'Local Admin',
    status: 'active',
    roles: ['admin', 'inventory', 'operator', 'procurement'],
    provider: 'local',
    lastLoginAt: '',
    updatedAt: '2026-04-23T00:00:00Z',
    rejectionReason: '',
  },
  {
    id: '00000000-0000-0000-0000-000000000002',
    email: 'operator@example.local',
    displayName: 'Local Operator',
    status: 'active',
    roles: ['operator'],
    provider: 'local',
    lastLoginAt: '',
    updatedAt: '2026-04-23T00:00:00Z',
    rejectionReason: '',
  },
  {
    id: '00000000-0000-0000-0000-000000000005',
    email: 'inspector@example.local',
    displayName: 'Local Inspector',
    status: 'active',
    roles: ['receiving_inspector'],
    provider: 'local',
    lastLoginAt: '',
    updatedAt: '2026-04-23T00:00:00Z',
    rejectionReason: '',
  },
]

const roles: RoleSummary[] = [
  { key: 'admin', description: 'Application administrator' },
  { key: 'operator', description: 'Operator application access' },
  { key: 'inventory', description: 'Inventory application access' },
  { key: 'procurement', description: 'Procurement application access' },
  { key: 'receiving_inspector', description: 'Acceptance inspection access' },
]

const syncRuns: MasterSyncRunEntry[] = [
  {
    id: 'sync-001',
    syncType: 'projects',
    projectId: '',
    projectKey: '',
    status: 'completed',
    rowCount: 2,
    source: 'mock_sync_adapter',
    triggeredBy: 'manual',
    errorMessage: '',
    startedAt: '2026-04-23T08:30:00Z',
    finishedAt: '2026-04-23T08:30:03Z',
  },
  {
    id: 'sync-002',
    syncType: 'budget_categories',
    projectId: 'proj-er2-upgrade',
    projectKey: 'ER2-UPGRADE',
    status: 'completed',
    rowCount: 2,
    source: 'mock_sync_adapter',
    triggeredBy: 'manual',
    errorMessage: '',
    startedAt: '2026-04-23T08:32:00Z',
    finishedAt: '2026-04-23T08:32:02Z',
  },
]

const webhookEvents: WebhookEventEntry[] = [
  {
    id: 'webhook-001',
    eventType: 'master.projects_changed',
    externalRequestReference: '',
    projectKey: '',
    normalizedStatus: '',
    rawStatus: '',
    receivedAt: '2026-04-23T08:35:00Z',
    processedAt: '2026-04-23T08:35:01Z',
    processingError: '',
  },
]

const dashboard: DashboardResponse = {
  generatedAt: '2026-04-22T08:00:00Z',
  metrics: [
    { label: 'Open shortages', value: '18', delta: '+4 since yesterday' },
    { label: 'Pending reservations', value: '42', delta: '7 need review' },
    { label: 'Draft requests', value: '6', delta: '2 from OCR' },
  ],
  alerts: [
    'Device ER2 / Scope powerboard has an unresolved shortage.',
    'Two OCR drafts still need item master registration.',
  ],
}

const reservations: ReservationListResponse = {
  rows: [
    {
      id: 'RES-001',
      itemNumber: 'ER2',
      description: 'Control relay',
      quantity: 12,
      device: 'ER2',
      scope: 'powerboard',
      status: 'reserved',
    },
    {
      id: 'RES-002',
      itemNumber: 'MK-44',
      description: 'Terminal block 4P',
      quantity: 8,
      device: 'MK4',
      scope: 'cabinet',
      status: 'awaiting_stock',
    },
  ],
}

const inventoryOverview: InventoryOverviewResponse = {
  balances: [
    {
      itemId: 'item-er2',
      itemNumber: 'ER2',
      description: 'Control relay',
      manufacturer: 'Omron',
      category: 'Relay',
      locationCode: 'TOKYO-A1',
      onHandQuantity: 9,
      reservedQuantity: 12,
      availableQuantity: -3,
    },
    {
      itemId: 'item-cn88',
      itemNumber: 'CN-88',
      description: 'I/O connector housing',
      manufacturer: 'Molex',
      category: 'Connector',
      locationCode: 'TOKYO-C1',
      onHandQuantity: 20,
      reservedQuantity: 0,
      availableQuantity: 20,
    },
  ],
}

const shortages: ShortageListResponse = {
  rows: [
    {
      device: 'ER2',
      scope: 'powerboard',
      manufacturer: 'Omron',
      itemNumber: 'ER2',
      description: 'Control relay',
      quantity: 3,
    },
    {
      device: 'MK4',
      scope: 'cabinet',
      manufacturer: 'Phoenix Contact',
      itemNumber: 'MK-44',
      description: 'Terminal block 4P',
      quantity: 3,
    },
  ],
}

const imports: ImportHistoryResponse = {
  rows: [
    {
      id: 'imp-001',
      importType: 'items',
      status: 'completed',
      fileName: 'items_master_20260422.csv',
      summary: '{"inserted":3,"updated":0}',
      createdAt: '2026-04-22T09:00:00Z',
    },
    {
      id: 'imp-002',
      importType: 'aliases',
      status: 'pending',
      fileName: 'supplier_aliases_20260422.csv',
      summary: '{"rows":14}',
      createdAt: '2026-04-22T10:30:00Z',
    },
  ],
}

const masterData: MasterDataSummaryResponse = {
  itemCount: 3,
  supplierCount: 2,
  aliasCount: 2,
  manufacturers: ['Molex', 'Omron', 'Phoenix Contact'],
  categories: [
    { key: 'connector', name: 'Connector' },
    { key: 'relay', name: 'Relay' },
    { key: 'terminal', name: 'Terminal Block' },
  ],
  suppliers: [
    { id: 'sup-misumi', name: 'MISUMI' },
    { id: 'sup-thorlabs', name: 'Thorlabs Japan' },
  ],
  aliases: [
    {
      id: 'alias-er2-pack4',
      supplierId: 'sup-misumi',
      supplierName: 'MISUMI',
      itemId: 'item-er2',
      canonicalItemNumber: 'ER2',
      supplierItemNumber: 'ER2-P4',
      unitsPerOrder: 4,
    },
    {
      id: 'alias-mk44-bulk',
      supplierId: 'sup-thorlabs',
      supplierName: 'Thorlabs Japan',
      itemId: 'item-mk44',
      canonicalItemNumber: 'MK-44',
      supplierItemNumber: 'MK44-BX',
      unitsPerOrder: 10,
    },
  ],
  recentItems: [
    {
      itemNumber: 'ER2',
      description: 'Control relay',
      manufacturer: 'Omron',
      category: 'Relay',
      supplier: 'MISUMI',
    },
    {
      itemNumber: 'MK-44',
      description: 'Terminal block 4P',
      manufacturer: 'Phoenix Contact',
      category: 'Terminal Block',
      supplier: 'Thorlabs Japan',
    },
  ],
  recentImportFiles: ['supplier_aliases_20260422.csv', 'items_master_20260422.csv'],
}

const projects: ProjectSummary[] = [
  { id: 'proj-er2-upgrade', key: 'ER2-UPGRADE', name: 'ER2 Production Upgrade', syncedAt: '2026-04-22T08:00:00Z' },
  { id: 'proj-mk4-refresh', key: 'MK4-REFRESH', name: 'MK4 Cabinet Refresh', syncedAt: '2026-04-22T08:00:00Z' },
]

const budgetCategories: BudgetCategorySummary[] = [
  {
    id: 'budget-er2-material',
    projectId: 'proj-er2-upgrade',
    key: 'material',
    name: 'Material Cost',
    syncedAt: '2026-04-22T08:00:00Z',
  },
  {
    id: 'budget-er2-maintenance',
    projectId: 'proj-er2-upgrade',
    key: 'maintenance',
    name: 'Maintenance',
    syncedAt: '2026-04-22T08:00:00Z',
  },
  {
    id: 'budget-mk4-material',
    projectId: 'proj-mk4-refresh',
    key: 'material',
    name: 'Material Cost',
    syncedAt: '2026-04-22T08:00:00Z',
  },
]

const procurementRequests: ProcurementRequestListResponse = {
  rows: [
    {
      id: 'batch-002',
      batchNumber: 'PR-20260423-002',
      title: 'MK4 cabinet restock',
      projectName: 'MK4 Cabinet Refresh',
      budgetCategoryName: 'Material Cost',
      supplierName: 'Thorlabs Japan',
      normalizedStatus: 'submitted',
      sourceType: 'manual',
      requestedItems: 1,
      dispatchStatus: 'submitted',
      artifactDeleteStatus: 'retained',
      createdAt: '2026-04-23T09:00:00Z',
    },
    {
      id: 'batch-001',
      batchNumber: 'PR-20260422-001',
      title: 'ER2 shortage replenishment',
      projectName: 'ER2 Production Upgrade',
      budgetCategoryName: 'Material Cost',
      supplierName: 'MISUMI',
      normalizedStatus: 'draft',
      sourceType: 'shortage',
      requestedItems: 1,
      dispatchStatus: 'not_submitted',
      artifactDeleteStatus: 'retained',
      createdAt: '2026-04-22T08:00:00Z',
    },
  ],
}

const procurementDetails: Record<string, ProcurementRequestDetailResponse> = {
  'batch-001': {
    id: 'batch-001',
    batchNumber: 'PR-20260422-001',
    title: 'ER2 shortage replenishment',
    projectName: 'ER2 Production Upgrade',
    budgetCategoryName: 'Material Cost',
    supplierName: 'MISUMI',
    quotationNumber: 'MISUMI-Q-20260422',
    quotationIssueDate: '2026-04-22',
    artifactPath: '/artifacts/quotations/misumi-q-20260422.pdf',
    artifactDeleteStatus: 'retained',
    artifactDeletedAt: '',
    normalizedStatus: 'draft',
    rawStatus: 'draft',
    externalRequestReference: '',
    dispatchStatus: 'not_submitted',
    dispatchAttempts: 0,
    lastDispatchAt: '',
    dispatchErrorCode: '',
    dispatchErrorMessage: '',
    quantityProgression: '{"requested":12,"ordered":0,"received":0}',
    lastReconciledAt: '',
    syncSource: '',
    syncError: '',
    lines: [
      {
        id: 'pline-001',
        itemNumber: 'ER2',
        description: 'Control relay',
        requestedQuantity: 12,
        deliveryLocation: 'Tokyo Assembly',
        accountingCategory: 'parts',
        leadTimeDays: 14,
        note: 'Created from ER2 shortage',
      },
    ],
    statusHistory: [
      {
        id: 'psh-001',
        normalizedStatus: 'draft',
        rawStatus: 'draft',
        observedAt: '2026-04-22T08:00:00Z',
        note: 'Draft created from shortage',
      },
    ],
    dispatchHistory: [],
  },
  'batch-002': {
    id: 'batch-002',
    batchNumber: 'PR-20260423-002',
    title: 'MK4 cabinet restock',
    projectName: 'MK4 Cabinet Refresh',
    budgetCategoryName: 'Material Cost',
    supplierName: 'Thorlabs Japan',
    quotationNumber: 'THORLABS-Q-20260423',
    quotationIssueDate: '2026-04-23',
    artifactPath: '/artifacts/quotations/thorlabs-q-20260423.pdf',
    artifactDeleteStatus: 'retained',
    artifactDeletedAt: '',
    normalizedStatus: 'submitted',
    rawStatus: 'submitted_to_internal_flow',
    externalRequestReference: 'LOCAL-SUBMIT-002',
    dispatchStatus: 'submitted',
    dispatchAttempts: 1,
    lastDispatchAt: '2026-04-23T09:00:00Z',
    dispatchErrorCode: '',
    dispatchErrorMessage: '',
    quantityProgression: '{"requested":8,"ordered":8,"received":0}',
    lastReconciledAt: '',
    syncSource: '',
    syncError: '',
    lines: [
      {
        id: 'pline-002',
        itemNumber: 'MK-44',
        description: 'Terminal block 4P',
        requestedQuantity: 8,
        deliveryLocation: 'Tokyo Assembly',
        accountingCategory: 'parts',
        leadTimeDays: 10,
        note: 'Restock for cabinet build',
      },
    ],
    statusHistory: [
      {
        id: 'psh-003',
        normalizedStatus: 'submitted',
        rawStatus: 'submitted_to_internal_flow',
        observedAt: '2026-04-23T09:00:00Z',
        note: 'Submitted for local tracking',
      },
      {
        id: 'psh-002',
        normalizedStatus: 'draft',
        rawStatus: 'draft',
        observedAt: '2026-04-22T08:00:00Z',
        note: 'Draft created manually',
      },
    ],
    dispatchHistory: [
      {
        id: 'dispatch-001',
        dispatchStatus: 'submitted',
        externalRequestReference: 'LOCAL-SUBMIT-002',
        retryable: false,
        errorCode: '',
        errorMessage: '',
        observedAt: '2026-04-23T09:00:00Z',
      },
    ],
  },
}

const ocrJobs: OCRJobListResponse = {
  rows: [
    {
      id: 'ocr-001',
      fileName: 'misumi_april_quote.pdf',
      contentType: 'application/pdf',
      status: 'ready_for_review',
      provider: 'mock',
      retryCount: 0,
      createdAt: '2026-04-22T11:00:00Z',
      updatedAt: '2026-04-22T11:01:00Z',
    },
  ],
}

const ocrJobDetails: Record<string, OCRJobDetailResponse> = {
  'ocr-001': {
    id: 'ocr-001',
    fileName: 'misumi_april_quote.pdf',
    contentType: 'application/pdf',
    artifactPath: '/artifacts/ocr/ocr-001/misumi_april_quote.pdf',
    status: 'ready_for_review',
    provider: 'mock',
    errorMessage: '',
    retryCount: 0,
    supplierName: 'MISUMI',
    supplierId: 'sup-misumi',
    supplierMatch: [
      {
        id: 'sup-misumi',
        name: 'MISUMI',
        score: 1,
        matchReason: 'exact supplier name match',
      },
    ],
    quotationId: '',
    quotationNumber: 'MISUMI_APRIL_QUOTE',
    issueDate: '2026-04-22',
    procurementRequestId: '',
    procurementBatchNumber: '',
    rawPayload: '{"provider":"mock","confidence":"draft"}',
    lines: [
      {
        id: 'ocr-001-line-1',
        itemId: 'item-er2',
        manufacturerName: 'Omron',
        itemNumber: 'ER2-P4',
        itemDescription: 'Control relay pack of 4',
        quantity: 12,
        leadTimeDays: 14,
        deliveryLocation: '',
        budgetCategoryId: '',
        accountingCategory: '',
        supplierContact: '',
        isUserConfirmed: false,
        matchCandidates: [
          {
            itemId: 'item-er2',
            canonicalItemNumber: 'ER2',
            description: 'Control relay',
            manufacturerName: 'Omron',
            defaultSupplierId: 'sup-misumi',
            supplierAlias: 'ER2-P4',
            score: 0.98,
            matchReason: 'exact supplier alias match, manufacturer match, default supplier match',
          },
        ],
      },
      {
        id: 'ocr-001-line-2',
        itemId: 'item-mk44',
        manufacturerName: 'Phoenix Contact',
        itemNumber: 'MK44-BX',
        itemDescription: 'Terminal block bulk box',
        quantity: 8,
        leadTimeDays: 10,
        deliveryLocation: '',
        budgetCategoryId: '',
        accountingCategory: '',
        supplierContact: '',
        isUserConfirmed: false,
        matchCandidates: [
          {
            itemId: 'item-mk44',
            canonicalItemNumber: 'MK-44',
            description: 'Terminal block 4P',
            manufacturerName: 'Phoenix Contact',
            defaultSupplierId: 'sup-thorlabs',
            supplierAlias: 'MK44-BX',
            score: 0.96,
            matchReason: 'exact supplier alias match, manufacturer match',
          },
        ],
      },
    ],
  },
}

export async function fetchBootstrap() {
  return fetchWithFallback<BootstrapResponse>('/api/v1/bootstrap', bootstrap)
}

export async function fetchCurrentSession() {
  const token = getStoredToken()
  if (!token) {
    return delay(currentSession)
  }
  try {
    const response = await fetch(`${config.apiBaseUrl}/api/v1/auth/me`, {
      headers: authorizationHeaders(),
    })
    if (!response.ok) {
      throw new Error(`request failed: ${response.status}`)
    }
    const payload = await response.json()
    currentSession = payload.data as AuthSessionResponse
    return currentSession
  } catch (error) {
    if (!allowMockApiFallback()) {
      throw error
    }
    const fallback = resolveMockSession(token)
    currentSession = fallback
    return delay(fallback)
  }
}

export async function registerUser(input: RegistrationInput) {
  try {
    const response = await fetch(`${config.apiBaseUrl}/api/v1/auth/register`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json', ...authorizationHeaders() },
      body: JSON.stringify(input),
    })
    if (!response.ok) {
      const payload = await response.json().catch(() => null)
      const backendError = (payload as { error?: string } | null)?.error
      if (backendError) {
        throw new Error(`${backendError} (${response.status})`)
      }
      if (response.status === 401) {
        throw new Error('authentication required (401): check sign-in state, API_BASE_URL, and backend JWT settings')
      }
      if (response.status === 403) {
        throw new Error('active account required (403): the identity exists but is not approved for app access yet')
      }
      if (response.status === 404) {
        throw new Error('registration endpoint not found (404): check API_BASE_URL points to the backend service')
      }
      throw new Error(`request failed: ${response.status}`)
    }
    const payload = await response.json()
    return payload.data as AppUserSummary
  } catch (error) {
    if (!allowMockApiFallback()) {
      throw error
    }
    const existing = users.find((candidate) => candidate.email === input.email)
    const user: AppUserSummary = existing ?? {
      id: `mock-user-${Date.now()}`,
      email: input.email,
      displayName: input.displayName,
      status: 'pending',
      roles: [],
      provider: 'local',
      lastLoginAt: '',
      updatedAt: new Date().toISOString(),
      rejectionReason: '',
    }
    user.displayName = input.displayName
    user.status = 'pending'
    user.updatedAt = new Date().toISOString()
    user.rejectionReason = ''
    if (!existing) {
      users.unshift(user)
    }
    currentSession = {
      ...currentSession,
      authenticated: true,
      user: {
        authenticated: true,
        userId: user.id,
        email: user.email,
        displayName: user.displayName,
        status: 'pending',
        roles: [],
        provider: 'local',
        subject: `local:${user.email.toLowerCase()}`,
        emailVerified: true,
        registrationNeeded: false,
        rejectionReason: '',
      },
    }
    return delay(user)
  }
}

export async function fetchUsers() {
  try {
    const response = await fetch(`${config.apiBaseUrl}/api/v1/admin/users`, {
      headers: authorizationHeaders(),
    })
    if (!response.ok) {
      throw new Error(`request failed: ${response.status}`)
    }
    const payload = await response.json()
    return payload.data as AppUserSummary[]
  } catch {
    return delay(users)
  }
}

export async function fetchRoles() {
  try {
    const response = await fetch(`${config.apiBaseUrl}/api/v1/admin/roles`, {
      headers: authorizationHeaders(),
    })
    if (!response.ok) {
      throw new Error(`request failed: ${response.status}`)
    }
    const payload = await response.json()
    return payload.data as RoleSummary[]
  } catch {
    return delay(roles)
  }
}

export async function approveUser(userId: string, selectedRoles: RoleSummary['key'][]) {
  try {
    const response = await fetch(`${config.apiBaseUrl}/api/v1/admin/users/${userId}/approve`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json', ...authorizationHeaders() },
      body: JSON.stringify({ roles: selectedRoles }),
    })
    if (!response.ok) {
      throw new Error(`request failed: ${response.status}`)
    }
    const payload = await response.json()
    return payload.data as AppUserSummary
  } catch {
    const target = users.find((candidate) => candidate.id === userId)
    if (!target) {
      throw new Error(`user not found: ${userId}`)
    }
    target.status = 'active'
    target.roles = [...selectedRoles]
    target.rejectionReason = ''
    target.updatedAt = new Date().toISOString()
    if (currentSession.user.email === target.email) {
      currentSession.user.status = 'active'
      currentSession.user.roles = [...selectedRoles]
    }
    return delay(target)
  }
}

export async function rejectUser(userId: string, reason: string) {
  try {
    const response = await fetch(`${config.apiBaseUrl}/api/v1/admin/users/${userId}/reject`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json', ...authorizationHeaders() },
      body: JSON.stringify({ reason }),
    })
    if (!response.ok) {
      throw new Error(`request failed: ${response.status}`)
    }
    const payload = await response.json()
    return payload.data as AppUserSummary
  } catch {
    const target = users.find((candidate) => candidate.id === userId)
    if (!target) {
      throw new Error(`user not found: ${userId}`)
    }
    target.status = 'rejected'
    target.roles = []
    target.rejectionReason = reason
    target.updatedAt = new Date().toISOString()
    if (currentSession.user.email === target.email) {
      currentSession.user.status = 'rejected'
      currentSession.user.roles = []
      currentSession.user.rejectionReason = reason
    }
    return delay(target)
  }
}

export async function fetchDashboard() {
  return fetchWithFallback<DashboardResponse>('/api/v1/operator/dashboard', dashboard, true)
}

export async function fetchReservations() {
  return fetchWithFallback<ReservationListResponse>('/api/v1/operator/reservations', reservations, true)
}

export async function fetchInventoryOverview() {
  return fetchWithFallback<InventoryOverviewResponse>('/api/v1/inventory/overview', inventoryOverview, true)
}

export async function fetchShortages(device?: string, scope?: string) {
  const search = new URLSearchParams()
  if (device) {
    search.set('device', device)
  }
  if (scope) {
    search.set('scope', scope)
  }
  const path = search.size > 0 ? `/api/v1/operator/shortages?${search.toString()}` : '/api/v1/operator/shortages'
  return fetchWithFallback<ShortageListResponse>(path, shortages, true)
}

export async function fetchImports() {
  return fetchWithFallback<ImportHistoryResponse>('/api/v1/operator/imports', imports, true)
}

export async function fetchMasterData() {
  return fetchWithFallback<MasterDataSummaryResponse>('/api/v1/admin/master-data', masterData, true)
}

export async function exportMasterDataCSV(exportType: 'items' | 'aliases') {
  try {
    const response = await fetch(`${config.apiBaseUrl}/api/v1/admin/master-data/export?type=${exportType}`)
    if (!response.ok) {
      throw new Error(`request failed: ${response.status}`)
    }
    return await response.text()
  } catch {
    if (exportType === 'items') {
      return delay([
        'canonical_item_number,description,manufacturer,category,default_supplier_id,note',
        'ER2,Control relay,Omron,Relay,sup-misumi,Standard relay used in powerboard assemblies',
        'MK-44,Terminal block 4P,Phoenix Contact,Terminal Block,sup-thorlabs,Common terminal block',
      ].join('\n'))
    }
    return delay([
      'supplier_id,supplier_name,canonical_item_number,supplier_item_number,units_per_order',
      'sup-misumi,MISUMI,ER2,ER2-P4,4',
      'sup-thorlabs,Thorlabs Japan,MK-44,MK44-BX,10',
    ].join('\n'))
  }
}

export async function importMasterDataCSV(importType: 'items' | 'aliases', file: File) {
  try {
    const formData = new FormData()
    formData.append('file', file)
    const response = await fetch(`${config.apiBaseUrl}/api/v1/admin/master-data/import?type=${importType}`, {
      method: 'POST',
      body: formData,
    })
    if (!response.ok) {
      const payload = await response.json().catch(() => null)
      throw new Error((payload as { error?: string } | null)?.error ?? `request failed: ${response.status}`)
    }
    const payload = await response.json()
    return payload.data
  } catch {
    const createdAt = new Date().toISOString()
    const job = {
      id: `imp-${Date.now()}`,
      importType,
      status: 'completed',
      fileName: file.name,
      summary: '{"inserted":1,"updated":0}',
      createdAt,
    }
    imports.rows.unshift(job)
    masterData.recentImportFiles.unshift(file.name)
    return delay(job)
  }
}

export async function fetchProcurementProjects() {
  return fetchWithFallback<ProjectSummary[]>('/api/v1/procurement/projects', projects, true)
}

export async function fetchProcurementBudgetCategories(projectId?: string) {
  const path = projectId ? `/api/v1/procurement/budget-categories?projectId=${projectId}` : '/api/v1/procurement/budget-categories'
  const filtered = projectId ? budgetCategories.filter((row) => row.projectId === projectId) : budgetCategories
  return fetchWithFallback<BudgetCategorySummary[]>(path, filtered, true)
}

export async function fetchProcurementRequests() {
  return fetchWithFallback<ProcurementRequestListResponse>('/api/v1/procurement/requests', procurementRequests, true)
}

export async function fetchProcurementRequestDetail(id: string) {
  return fetchWithFallback<ProcurementRequestDetailResponse>(
    `/api/v1/procurement/requests/${id}`,
    procurementDetails[id],
    true,
  )
}

export async function submitProcurementRequest(id: string) {
  try {
    const response = await fetch(`${config.apiBaseUrl}/api/v1/procurement/requests/${id}/submit`, {
      method: 'POST',
    })
    if (!response.ok) {
      const payload = await response.json().catch(() => null)
      throw new Error((payload as { error?: string } | null)?.error ?? `request failed: ${response.status}`)
    }
    const payload = await response.json()
    return payload.data as ProcurementSubmitResult
  } catch {
    const detail = procurementDetails[id]
    if (!detail) {
      throw new Error(`request not found: ${id}`)
    }
    if (detail.dispatchStatus === 'submitted' && detail.externalRequestReference) {
      return delay<ProcurementSubmitResult>({
        requestId: id,
        externalRequestReference: detail.externalRequestReference,
        dispatchStatus: detail.dispatchStatus,
        artifactDeleteStatus: detail.artifactDeleteStatus,
      })
    }
    const observedAt = new Date().toISOString()
    const externalRequestReference = `LOCAL-SUBMIT-${Date.now()}`
    detail.normalizedStatus = 'submitted'
    detail.rawStatus = 'submitted_to_mock_adapter'
    detail.externalRequestReference = externalRequestReference
    detail.dispatchStatus = 'submitted'
    detail.dispatchAttempts += 1
    detail.lastDispatchAt = observedAt
    detail.artifactDeleteStatus = detail.artifactPath ? 'deleted' : detail.artifactDeleteStatus
    detail.artifactDeletedAt = detail.artifactPath ? observedAt : detail.artifactDeletedAt
    detail.dispatchErrorCode = ''
    detail.dispatchErrorMessage = ''
    detail.dispatchHistory = [
      {
        id: `dispatch-${Date.now()}`,
        dispatchStatus: 'submitted',
        externalRequestReference,
        retryable: false,
        errorCode: '',
        errorMessage: '',
        observedAt,
      },
      ...detail.dispatchHistory,
    ]
    const row = procurementRequests.rows.find((candidate) => candidate.id === id)
    if (row) {
      row.normalizedStatus = 'submitted'
      row.dispatchStatus = 'submitted'
      row.artifactDeleteStatus = detail.artifactDeleteStatus
    }
    return delay<ProcurementSubmitResult>({
      requestId: id,
      externalRequestReference,
      dispatchStatus: 'submitted',
      artifactDeleteStatus: detail.artifactDeleteStatus,
    })
  }
}

export async function reconcileProcurementRequest(id: string) {
  try {
    const response = await fetch(`${config.apiBaseUrl}/api/v1/procurement/requests/${id}/reconcile`, {
      method: 'POST',
    })
    if (!response.ok) {
      const payload = await response.json().catch(() => null)
      throw new Error((payload as { error?: string } | null)?.error ?? `request failed: ${response.status}`)
    }
    const payload = await response.json()
    return payload.data as ProcurementReconcileResult
  } catch {
    const detail = procurementDetails[id]
    if (!detail) {
      throw new Error(`request not found: ${id}`)
    }
    if (!detail.externalRequestReference) {
      throw new Error('Request must be submitted before reconciliation')
    }
    const progression = parseQuantityProgression(detail.quantityProgression)
    const nextStatus = nextMockStatus(detail.normalizedStatus)
    const nextProgression = nextMockProgression(nextStatus, progression)
    const observedAt = new Date().toISOString()
    detail.normalizedStatus = nextStatus
    detail.rawStatus = rawStatusForMock(nextStatus)
    detail.quantityProgression = JSON.stringify(nextProgression)
    detail.lastReconciledAt = observedAt
    detail.syncSource = 'mock_sync_adapter'
    detail.syncError = ''
    detail.statusHistory = [
      {
        id: `psh-${Date.now()}`,
        normalizedStatus: detail.normalizedStatus,
        rawStatus: detail.rawStatus,
        observedAt,
        note: 'Reconciled via local mock adapter',
      },
      ...detail.statusHistory,
    ]
    const row = procurementRequests.rows.find((candidate) => candidate.id === id)
    if (row) {
      row.normalizedStatus = detail.normalizedStatus
    }
    webhookEvents.unshift({
      id: `webhook-${Date.now()}`,
      eventType: 'procurement.status_changed',
      externalRequestReference: detail.externalRequestReference,
      projectKey: '',
      normalizedStatus: detail.normalizedStatus,
      rawStatus: detail.rawStatus,
      receivedAt: observedAt,
      processedAt: observedAt,
      processingError: '',
    })
    return delay<ProcurementReconcileResult>({
      requestId: id,
      normalizedStatus: detail.normalizedStatus,
      rawStatus: detail.rawStatus,
      quantityProgression: detail.quantityProgression,
      lastReconciledAt: observedAt,
      syncSource: detail.syncSource,
    })
  }
}

export async function refreshProcurementProjects() {
  try {
    const response = await fetch(`${config.apiBaseUrl}/api/v1/procurement/projects/refresh`, {
      method: 'POST',
    })
    if (!response.ok) {
      const payload = await response.json().catch(() => null)
      throw new Error((payload as { error?: string } | null)?.error ?? `request failed: ${response.status}`)
    }
    const payload = await response.json()
    return payload.data as MasterSyncResult
  } catch {
    const syncedAt = new Date().toISOString()
    for (const project of projects) {
      project.syncedAt = syncedAt
    }
    syncRuns.unshift({
      id: `sync-${Date.now()}`,
      syncType: 'projects',
      projectId: '',
      projectKey: '',
      status: 'completed',
      rowCount: projects.length,
      source: 'mock_sync_adapter',
      triggeredBy: 'manual',
      errorMessage: '',
      startedAt: syncedAt,
      finishedAt: syncedAt,
    })
    return delay<MasterSyncResult>({
      syncType: 'projects',
      projectId: '',
      projectKey: '',
      status: 'completed',
      rowCount: projects.length,
      source: 'mock_sync_adapter',
      triggeredBy: 'manual',
      syncedAt,
    })
  }
}

export async function refreshProcurementBudgetCategories(projectId?: string) {
  const query = projectId ? `?projectId=${projectId}` : ''
  try {
    const response = await fetch(`${config.apiBaseUrl}/api/v1/procurement/budget-categories/refresh${query}`, {
      method: 'POST',
    })
    if (!response.ok) {
      const payload = await response.json().catch(() => null)
      throw new Error((payload as { error?: string } | null)?.error ?? `request failed: ${response.status}`)
    }
    const payload = await response.json()
    return payload.data as MasterSyncResult
  } catch {
    const syncedAt = new Date().toISOString()
    const applicableRows = projectId ? budgetCategories.filter((row) => row.projectId === projectId) : budgetCategories
    for (const row of applicableRows) {
      row.syncedAt = syncedAt
    }
    const project = projectId ? projects.find((candidate) => candidate.id === projectId) : undefined
    syncRuns.unshift({
      id: `sync-${Date.now()}`,
      syncType: 'budget_categories',
      projectId: projectId ?? '',
      projectKey: project?.key ?? '',
      status: 'completed',
      rowCount: applicableRows.length,
      source: 'mock_sync_adapter',
      triggeredBy: 'manual',
      errorMessage: '',
      startedAt: syncedAt,
      finishedAt: syncedAt,
    })
    return delay<MasterSyncResult>({
      syncType: 'budget_categories',
      projectId: projectId ?? '',
      projectKey: project?.key ?? '',
      status: 'completed',
      rowCount: applicableRows.length,
      source: 'mock_sync_adapter',
      triggeredBy: 'manual',
      syncedAt,
    })
  }
}

export async function createReservation(input: {
  itemId: string
  deviceScopeId: string
  quantity: number
  requestedBy: string
  note: string
}) {
  try {
    const response = await fetch(`${config.apiBaseUrl}/api/v1/operator/reservations`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json', ...authorizationHeaders() },
      body: JSON.stringify(input),
    })
    if (!response.ok) {
      const payload = await response.json().catch(() => null)
      throw new Error((payload as { error?: string } | null)?.error ?? `request failed: ${response.status}`)
    }
    return await response.json()
  } catch {
    reservations.rows.unshift({
      id: `RES-${Date.now()}`,
      itemNumber: input.itemId,
      description: input.note || 'Local reservation draft',
      quantity: input.quantity,
      device: input.deviceScopeId.split('-')[1] ?? 'ER2',
      scope: input.deviceScopeId.split('-')[2] ?? 'powerboard',
      status: 'reserved',
    })
    return delay({ status: 'created' })
  }
}

export async function adjustInventory(input: {
  itemId: string
  locationCode: string
  quantityDelta: number
  deviceScopeId: string
  note: string
}) {
  try {
    const response = await fetch(`${config.apiBaseUrl}/api/v1/inventory/adjustments`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json', ...authorizationHeaders() },
      body: JSON.stringify(input),
    })
    if (!response.ok) {
      const payload = await response.json().catch(() => null)
      throw new Error((payload as { error?: string } | null)?.error ?? `request failed: ${response.status}`)
    }
    return await response.json()
  } catch {
    const target = inventoryOverview.balances.find((row) => row.itemId === input.itemId && row.locationCode === input.locationCode)
    if (target) {
      target.onHandQuantity += input.quantityDelta
      target.availableQuantity += input.quantityDelta
    }
    return delay({ status: 'created' })
  }
}

export async function fetchProcurementSyncRuns() {
  return fetchWithFallback<MasterSyncRunEntry[]>('/api/v1/procurement/sync-runs', syncRuns, true)
}

export async function fetchProcurementWebhookEvents() {
  return fetchWithFallback<WebhookEventEntry[]>('/api/v1/procurement/webhooks/external', webhookEvents, true)
}

export async function sendMockProcurementWebhook(input: {
  eventType: string
  requestId?: string
  projectKey?: string
  normalizedStatus?: string
  rawStatus?: string
}) {
  const payload = buildWebhookPayload(input)
  try {
    const response = await fetch(`${config.apiBaseUrl}/api/v1/procurement/webhooks/external`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json', ...authorizationHeaders() },
      body: JSON.stringify(payload),
    })
    if (!response.ok) {
      const result = await response.json().catch(() => null)
      throw new Error((result as { error?: string } | null)?.error ?? `request failed: ${response.status}`)
    }
    const result = await response.json()
    return result.data as { eventType: string; status: string; syncedAt: string; requestId?: string; projectKey?: string }
  } catch {
    const receivedAt = new Date().toISOString()
    webhookEvents.unshift({
      id: `webhook-${Date.now()}`,
      eventType: payload.eventType,
      externalRequestReference: payload.externalRequestReference ?? '',
      projectKey: payload.projectKey ?? '',
      normalizedStatus: payload.normalizedStatus ?? '',
      rawStatus: payload.rawStatus ?? '',
      receivedAt,
      processedAt: receivedAt,
      processingError: '',
    })

    if (payload.eventType === 'procurement.status_changed' && payload.externalRequestReference) {
      const target = Object.values(procurementDetails).find(
        (candidate) => candidate.externalRequestReference === payload.externalRequestReference,
      )
      if (target) {
        target.normalizedStatus = payload.normalizedStatus || target.normalizedStatus
        target.rawStatus = payload.rawStatus || target.rawStatus
        target.lastReconciledAt = receivedAt
        target.syncSource = 'mock_sync_adapter'
        target.syncError = ''
        target.statusHistory.unshift({
          id: `psh-${Date.now()}`,
          normalizedStatus: target.normalizedStatus,
          rawStatus: target.rawStatus,
          observedAt: receivedAt,
          note: 'Webhook-driven reconciliation',
        })
        const row = procurementRequests.rows.find((candidate) => candidate.id === target.id)
        if (row) {
          row.normalizedStatus = target.normalizedStatus
        }
      }
    }

    if (payload.eventType === 'master.projects_changed') {
      for (const project of projects) {
        project.syncedAt = receivedAt
      }
      syncRuns.unshift({
        id: `sync-${Date.now()}`,
        syncType: 'projects',
        projectId: '',
        projectKey: '',
        status: 'completed',
        rowCount: projects.length,
        source: 'mock_sync_adapter',
        triggeredBy: 'webhook',
        errorMessage: '',
        startedAt: receivedAt,
        finishedAt: receivedAt,
      })
    }

    if (payload.eventType === 'master.budget_categories_changed') {
      const project = projects.find((candidate) => candidate.key === payload.projectKey)
      const matchingBudgets = budgetCategories.filter((candidate) => candidate.projectId === project?.id)
      for (const row of matchingBudgets) {
        row.syncedAt = receivedAt
      }
      syncRuns.unshift({
        id: `sync-${Date.now()}`,
        syncType: 'budget_categories',
        projectId: project?.id ?? '',
        projectKey: payload.projectKey ?? '',
        status: 'completed',
        rowCount: matchingBudgets.length,
        source: 'mock_sync_adapter',
        triggeredBy: 'webhook',
        errorMessage: '',
        startedAt: receivedAt,
        finishedAt: receivedAt,
      })
    }

    return delay({
      eventType: payload.eventType,
      status: 'processed',
      syncedAt: receivedAt,
      requestId: input.requestId ?? '',
      projectKey: payload.projectKey ?? '',
    })
  }
}

export async function createProcurementRequest(input: ProcurementRequestCreateInput) {
  try {
    const response = await fetch(`${config.apiBaseUrl}/api/v1/procurement/requests`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify(input),
    })
    if (!response.ok) {
      throw new Error(`request failed: ${response.status}`)
    }
    return await response.json()
  } catch {
    return delay({ id: `mock-${Date.now()}`, status: 'created' })
  }
}

export async function fetchOCRJobs() {
  return fetchWithFallback<OCRJobListResponse>('/api/v1/procurement/ocr-jobs', ocrJobs, true)
}

export async function fetchOCRJobDetail(id: string) {
  return fetchWithFallback<OCRJobDetailResponse>(`/api/v1/procurement/ocr-jobs/${id}`, ocrJobDetails[id], true)
}

export async function uploadOCRJob(file: File) {
  try {
    const formData = new FormData()
    formData.append('file', file)
    formData.append('createdBy', 'local-user')
    const response = await fetch(`${config.apiBaseUrl}/api/v1/procurement/ocr-jobs`, {
      method: 'POST',
      body: formData,
    })
    if (!response.ok) {
      throw new Error(`request failed: ${response.status}`)
    }
    return await response.json()
  } catch {
    return delay({ data: { id: `ocr-${Date.now()}`, status: 'ready_for_review' } })
  }
}

export async function assistOCRLine(jobId: string, input: OCRLineAssistInput) {
  try {
    const response = await fetch(`${config.apiBaseUrl}/api/v1/procurement/ocr-jobs/${jobId}/assist`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify(input),
    })
    if (!response.ok) {
      throw new Error(`request failed: ${response.status}`)
    }
    const payload = await response.json()
    return payload.data as OCRLineAssistSuggestion
  } catch {
    const fallback = ocrJobDetails[jobId]?.lines.find((line) => line.id === input.lineId)
    return delay<OCRLineAssistSuggestion>({
      lineId: input.lineId,
      matchedItemId: fallback?.matchCandidates[0]?.itemId ?? '',
      suggestedCanonicalNumber: fallback?.itemNumber ?? '',
      suggestedManufacturer: fallback?.manufacturerName ?? '',
      suggestedCategoryKey: 'misc',
      suggestedAliasNumber: fallback?.itemNumber ?? '',
      confidence: fallback?.matchCandidates[0]?.score ?? 0.4,
      rationale: fallback?.matchCandidates[0]?.matchReason ?? 'mock assist fallback',
      candidates: fallback?.matchCandidates ?? [],
    })
  }
}

export async function updateOCRReview(jobId: string, input: OCRReviewUpdateInput) {
  try {
    const response = await fetch(`${config.apiBaseUrl}/api/v1/procurement/ocr-jobs/${jobId}/review`, {
      method: 'PATCH',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify(input),
    })
    if (!response.ok) {
      const payload = await response.json().catch(() => null)
      throw new Error((payload as { error?: string } | null)?.error ?? `request failed: ${response.status}`)
    }
    return await response.json()
  } catch {
    const detail = ocrJobDetails[jobId]
    if (!detail) {
      throw new Error(`ocr job not found: ${jobId}`)
    }
    detail.supplierId = input.supplierId
    detail.quotationNumber = input.quotationNumber
    detail.issueDate = input.issueDate
    detail.status = 'reviewed'
    detail.lines = detail.lines.map((line) => {
      const update = input.lines.find((candidate) => candidate.id === line.id)
      if (!update) {
        return line
      }
      return {
        ...line,
        itemId: update.itemId,
        deliveryLocation: update.deliveryLocation,
        budgetCategoryId: update.budgetCategoryId,
        accountingCategory: update.accountingCategory,
        supplierContact: update.supplierContact,
        isUserConfirmed: update.isUserConfirmed,
      }
    })
    const row = ocrJobs.rows.find((candidate) => candidate.id === jobId)
    if (row) {
      row.status = 'reviewed'
      row.updatedAt = new Date().toISOString()
    }
    return delay({ status: 'reviewed' })
  }
}

export async function retryOCRJob(jobId: string) {
  try {
    const response = await fetch(`${config.apiBaseUrl}/api/v1/procurement/ocr-jobs/${jobId}/retry`, {
      method: 'POST',
    })
    if (!response.ok) {
      const payload = await response.json().catch(() => null)
      throw new Error((payload as { error?: string } | null)?.error ?? `request failed: ${response.status}`)
    }
    const payload = await response.json()
    return payload.data as OCRRetryResult
  } catch {
    const row = ocrJobs.rows.find((candidate) => candidate.id === jobId)
    const detail = ocrJobDetails[jobId]
    if (!row || !detail) {
      throw new Error(`ocr job not found: ${jobId}`)
    }
    row.status = 'ready_for_review'
    row.retryCount += 1
    row.updatedAt = new Date().toISOString()
    detail.status = 'ready_for_review'
    detail.retryCount += 1
    detail.errorMessage = ''
    return delay<OCRRetryResult>({
      id: jobId,
      status: 'ready_for_review',
      retryCount: detail.retryCount,
    })
  }
}

export async function registerOCRItem(jobId: string, input: OCRRegisterItemInput) {
  try {
    const response = await fetch(`${config.apiBaseUrl}/api/v1/procurement/ocr-jobs/${jobId}/register-item`, {
      method: 'PUT',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify(input),
    })
    if (!response.ok) {
      throw new Error(`request failed: ${response.status}`)
    }
    return await response.json()
  } catch {
    return delay({ itemId: `item-${Date.now()}`, status: 'registered' })
  }
}

export async function createProcurementDraftFromOCR(jobId: string) {
  try {
    const response = await fetch(`${config.apiBaseUrl}/api/v1/procurement/ocr-jobs/${jobId}/create-draft`, {
      method: 'POST',
    })
    if (!response.ok) {
      const payload = await response.json().catch(() => null)
      throw new Error((payload as { error?: string } | null)?.error ?? `request failed: ${response.status}`)
    }
    const payload = await response.json()
    return payload.data as OCRProcurementDraftCreateResult
  } catch (error) {
    const detail = ocrJobDetails[jobId]
    if (!detail) {
      throw error
    }
    if (detail.procurementRequestId) {
      return delay<OCRProcurementDraftCreateResult>({
        procurementRequestId: detail.procurementRequestId,
        procurementBatchNumber: detail.procurementBatchNumber,
        quotationId: detail.quotationId,
        status: 'existing',
      })
    }

    const createdAt = new Date().toISOString()
    const batchId = `mock-batch-${Date.now()}`
    const batchNumber = `PR-${createdAt.slice(0, 10).replaceAll('-', '')}-OCR`
    const quotationId = detail.quotationId || `mock-quote-${Date.now()}`
    const requestedQuantity = detail.lines.reduce((total, line) => total + line.quantity, 0)
    const supplierName = detail.supplierMatch[0]?.name || detail.supplierName || detail.supplierId

    procurementRequests.rows.unshift({
      id: batchId,
      batchNumber,
      title: `${detail.supplierName || 'OCR'} quotation ${detail.quotationNumber}`,
      projectName: '',
      budgetCategoryName: '',
      supplierName,
      normalizedStatus: 'draft',
      sourceType: 'ocr',
      requestedItems: detail.lines.length,
      dispatchStatus: 'not_submitted',
      artifactDeleteStatus: 'retained',
      createdAt,
    })

    procurementDetails[batchId] = {
      id: batchId,
      batchNumber,
      title: `${detail.supplierName || 'OCR'} quotation ${detail.quotationNumber}`,
      projectName: '',
      budgetCategoryName: '',
      supplierName,
      quotationNumber: detail.quotationNumber,
      quotationIssueDate: detail.issueDate,
      artifactPath: detail.artifactPath,
      artifactDeleteStatus: 'retained',
      artifactDeletedAt: '',
      normalizedStatus: 'draft',
      rawStatus: 'draft',
      externalRequestReference: '',
      dispatchStatus: 'not_submitted',
      dispatchAttempts: 0,
      lastDispatchAt: '',
      dispatchErrorCode: '',
      dispatchErrorMessage: '',
      quantityProgression: JSON.stringify({ requested: requestedQuantity, ordered: 0, received: 0 }),
      lastReconciledAt: '',
      syncSource: '',
      syncError: '',
      lines: detail.lines.map((line, index) => ({
        id: `mock-pline-${index + 1}-${Date.now()}`,
        itemNumber: line.matchCandidates[0]?.canonicalItemNumber || line.itemNumber,
        description: line.itemDescription,
        requestedQuantity: line.quantity,
        deliveryLocation: line.deliveryLocation,
        accountingCategory: line.accountingCategory,
        leadTimeDays: line.leadTimeDays,
        note: `Created from OCR quotation line ${line.itemNumber || line.id}`,
      })),
      statusHistory: [
        {
          id: `mock-psh-${Date.now()}`,
          normalizedStatus: 'draft',
          rawStatus: 'draft',
          observedAt: createdAt,
          note: `Created from OCR job ${jobId}`,
        },
      ],
      dispatchHistory: [],
    }

    detail.quotationId = quotationId
    detail.procurementRequestId = batchId
    detail.procurementBatchNumber = batchNumber

    return delay<OCRProcurementDraftCreateResult>({
      procurementRequestId: batchId,
      procurementBatchNumber: batchNumber,
      quotationId,
      status: 'created',
    })
  }
}

export async function exportShortagesCSV(device?: string, scope?: string) {
  try {
    const search = new URLSearchParams()
    if (device) {
      search.set('device', device)
    }
    if (scope) {
      search.set('scope', scope)
    }
    const path = search.size > 0 ? `/api/v1/operator/shortages/export?${search.toString()}` : '/api/v1/operator/shortages/export'
    const response = await fetch(`${config.apiBaseUrl}${path}`)
    if (!response.ok) {
      throw new Error(`request failed: ${response.status}`)
    }
    return await response.text()
  } catch {
    const filtered = shortages.rows.filter((row) => (!device || row.device === device) && (!scope || row.scope === scope))
    return delay([
      'device,scope,manufacturer,item_number,description,quantity',
      ...filtered.map((row) => [row.device, row.scope, row.manufacturer, row.itemNumber, row.description, row.quantity].join(',')),
    ].join('\n'))
  }
}

function delay<T>(value: T, timeout = 120): Promise<T> {
  return new Promise((resolve) => {
    window.setTimeout(() => resolve(value), timeout)
  })
}

async function fetchWithFallback<T>(path: string, fallback: T, nestedData = false): Promise<T> {
  try {
    const response = await fetch(`${config.apiBaseUrl}${path}`, {
      headers: authorizationHeaders(),
    })
    if (!response.ok) {
      throw new Error(`request failed: ${response.status}`)
    }
    const payload = await response.json()
    return nestedData ? (payload.data as T) : (payload as T)
  } catch (error) {
    if (!allowMockApiFallback()) {
      throw error
    }
    return delay(fallback)
  }
}

function resolveMockSession(token: string): AuthSessionResponse {
  const knownUser = token === 'local-admin-token'
    ? users.find((candidate) => candidate.email === 'admin@example.local')
    : token === 'local-operator-token'
      ? users.find((candidate) => candidate.email === 'operator@example.local')
      : token === 'local-inspector-token'
        ? users.find((candidate) => candidate.email === 'inspector@example.local')
      : undefined
  if (knownUser) {
    return {
      authenticated: true,
      authMode: bootstrap.authMode,
      authProvider: bootstrap.authProvider,
      rbacMode: bootstrap.rbacMode,
      user: {
        authenticated: true,
        userId: knownUser.id,
        email: knownUser.email,
        displayName: knownUser.displayName,
        status: knownUser.status,
        roles: knownUser.roles,
        provider: 'local',
        subject: `local:${knownUser.email.toLowerCase()}`,
        emailVerified: true,
        registrationNeeded: false,
        rejectionReason: knownUser.rejectionReason,
      },
    }
  }

  if (token.startsWith('local:')) {
    const payload = token.slice('local:'.length).split('|')
    const email = payload[0] ?? ''
    const displayName = payload[1] ?? email
    const existing = users.find((candidate) => candidate.email === email)
    if (existing) {
      return {
        authenticated: true,
        authMode: bootstrap.authMode,
        authProvider: bootstrap.authProvider,
        rbacMode: bootstrap.rbacMode,
        user: {
          authenticated: true,
          userId: existing.id,
          email: existing.email,
          displayName: existing.displayName,
          status: existing.status,
          roles: existing.roles,
          provider: 'local',
          subject: `local:${existing.email.toLowerCase()}`,
          emailVerified: true,
          registrationNeeded: false,
          rejectionReason: existing.rejectionReason,
        },
      }
    }
    return {
      authenticated: true,
      authMode: bootstrap.authMode,
      authProvider: bootstrap.authProvider,
      rbacMode: bootstrap.rbacMode,
      user: {
        authenticated: true,
        userId: '',
        email,
        displayName,
        status: 'unregistered',
        roles: [],
        provider: 'local',
        subject: `local:${email.toLowerCase()}`,
        emailVerified: true,
        registrationNeeded: true,
        rejectionReason: '',
      },
    }
  }

  return currentSession
}

function parseQuantityProgression(raw: string) {
  try {
    return JSON.parse(raw) as { requested: number; ordered: number; received: number }
  } catch {
    return { requested: 0, ordered: 0, received: 0 }
  }
}

function nextMockStatus(currentStatus: string) {
  switch (currentStatus) {
    case 'draft':
      return 'submitted'
    case 'submitted':
      return 'ordered'
    case 'ordered':
      return 'partially_received'
    case 'partially_received':
      return 'received'
    default:
      return 'received'
  }
}

function nextMockProgression(
  nextStatus: string,
  progression: { requested: number; ordered: number; received: number },
) {
  if (nextStatus === 'ordered') {
    return { ...progression, ordered: Math.max(progression.ordered, progression.requested), received: 0 }
  }
  if (nextStatus === 'partially_received') {
    return {
      ...progression,
      ordered: Math.max(progression.ordered, progression.requested),
      received: Math.max(progression.received, Math.max(1, Math.floor(progression.requested / 2))),
    }
  }
  if (nextStatus === 'received') {
    return {
      ...progression,
      ordered: Math.max(progression.ordered, progression.requested),
      received: progression.requested,
    }
  }
  return progression
}

function rawStatusForMock(normalizedStatus: string) {
  switch (normalizedStatus) {
    case 'submitted':
      return 'submitted_to_external_flow'
    case 'ordered':
      return 'external_order_confirmed'
    case 'partially_received':
      return 'external_partial_receipt'
    case 'received':
      return 'external_receipt_completed'
    default:
      return normalizedStatus
  }
}

function buildWebhookPayload(input: {
  eventType: string
  requestId?: string
  projectKey?: string
  normalizedStatus?: string
  rawStatus?: string
}) {
  const detail = input.requestId ? procurementDetails[input.requestId] : undefined
  return {
    eventType: input.eventType,
    externalRequestReference: detail?.externalRequestReference ?? '',
    projectKey: input.projectKey ?? '',
    normalizedStatus: input.normalizedStatus ?? detail?.normalizedStatus ?? '',
    rawStatus: input.rawStatus ?? detail?.rawStatus ?? '',
  }
}
