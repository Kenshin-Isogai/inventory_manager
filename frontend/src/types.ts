export type AppSection = 'operator' | 'inventory' | 'procurement' | 'inspector' | 'admin'

export type RoleKey =
  | 'admin'
  | 'operator'
  | 'inventory'
  | 'procurement'
  | 'receiving_inspector'

export type RouteDefinition = {
  app: AppSection
  path: string
  label: string
  description: string
  roles: RoleKey[]
  icon?: React.ComponentType<{ className?: string }>
}

export type BootstrapResponse = {
  frontendBaseUrl: string
  authMode: string
  authProvider: string
  rbacMode: string
  storageMode: string
  capabilities: string[]
}

export type AuthSessionUser = {
  authenticated: boolean
  userId: string
  email: string
  displayName: string
  status: string
  roles: RoleKey[]
  provider: string
  subject: string
  emailVerified: boolean
  registrationNeeded: boolean
  rejectionReason: string
}

export type AuthSessionResponse = {
  authenticated: boolean
  authMode: string
  authProvider: string
  rbacMode: string
  user: AuthSessionUser
}

export type RegistrationInput = {
  email: string
  displayName: string
}

export type AppUserSummary = {
  id: string
  email: string
  displayName: string
  status: string
  roles: RoleKey[]
  provider: string
  lastLoginAt: string
  updatedAt: string
  rejectionReason: string
}

export type RoleSummary = {
  key: RoleKey
  description: string
}

export type DashboardMetric = {
  label: string
  value: string
  delta: string
}

export type DashboardResponse = {
  generatedAt: string
  metrics: DashboardMetric[]
  alerts: string[]
}

export type ReservationSummary = {
  id: string
  itemNumber: string
  description: string
  quantity: number
  device: string
  scope: string
  status: 'reserved' | 'partially_allocated' | 'awaiting_stock'
}

export type ReservationListResponse = {
  rows: ReservationSummary[]
}

export type DeviceScopeRecord = {
  id: string
  deviceId: string
  deviceKey: string
  scopeKey: string
  scopeName: string
  scopeType: string
  ownerDepartmentKey: string
  status: string
}

export type DeviceScopeListResponse = {
  rows: DeviceScopeRecord[]
}

export type DeviceScopeUpsertInput = {
  id?: string
  deviceId?: string
  deviceKey: string
  scopeKey: string
  scopeName: string
  scopeType: string
  ownerDepartmentKey: string
  status: string
}

export type DeviceRecord = {
  id: string
  deviceKey: string
  name: string
  deviceType: string
  status: string
}

export type DeviceListResponse = {
  rows: DeviceRecord[]
}

export type InventoryBalance = {
  itemId: string
  itemNumber: string
  description: string
  manufacturer: string
  category: string
  locationCode: string
  onHandQuantity: number
  reservedQuantity: number
  availableQuantity: number
}

export type InventoryOverviewResponse = {
  balances: InventoryBalance[]
}

export type ShortageRow = {
  device: string
  scope: string
  manufacturer: string
  itemNumber: string
  description: string
  quantity: number
}

export type ShortageListResponse = {
  rows: ShortageRow[]
}

export type ImportJob = {
  id: string
  importType: string
  status: string
  fileName: string
  summary: string
  createdAt: string
}

export type ImportHistoryResponse = {
  rows: ImportJob[]
}

export type MasterItem = {
  itemNumber: string
  description: string
  manufacturer: string
  category: string
  supplier: string
}

export type CategorySummary = {
  key: string
  name: string
}

export type SupplierSummary = {
  id: string
  name: string
}

export type SupplierAliasSummary = {
  id: string
  supplierId: string
  supplierName: string
  itemId: string
  canonicalItemNumber: string
  supplierItemNumber: string
  unitsPerOrder: number
}

export type MasterDataSummaryResponse = {
  itemCount: number
  supplierCount: number
  aliasCount: number
  manufacturers: string[]
  categories: CategorySummary[]
  suppliers: SupplierSummary[]
  aliases: SupplierAliasSummary[]
  recentItems: MasterItem[]
  recentImportFiles: string[]
}

export type ProjectSummary = {
  id: string
  key: string
  name: string
  syncedAt: string
}

export type BudgetCategorySummary = {
  id: string
  projectId: string
  key: string
  name: string
  syncedAt: string
}

export type ProcurementRequestSummary = {
  id: string
  batchNumber: string
  title: string
  projectName: string
  budgetCategoryName: string
  supplierName: string
  normalizedStatus: string
  sourceType: string
  requestedItems: number
  dispatchStatus: string
  artifactDeleteStatus: string
  createdAt: string
}

export type ProcurementRequestListResponse = {
  rows: ProcurementRequestSummary[]
}

export type ProcurementLine = {
  id: string
  itemNumber: string
  description: string
  requestedQuantity: number
  deliveryLocation: string
  accountingCategory: string
  leadTimeDays: number
  note: string
}

export type StatusHistoryEntry = {
  id: string
  normalizedStatus: string
  rawStatus: string
  observedAt: string
  note: string
}

export type ProcurementRequestDetailResponse = {
  id: string
  batchNumber: string
  title: string
  projectName: string
  budgetCategoryName: string
  supplierName: string
  quotationNumber: string
  quotationIssueDate: string
  artifactPath: string
  artifactDeleteStatus: string
  artifactDeletedAt: string
  normalizedStatus: string
  rawStatus: string
  externalRequestReference: string
  dispatchStatus: string
  dispatchAttempts: number
  lastDispatchAt: string
  dispatchErrorCode: string
  dispatchErrorMessage: string
  quantityProgression: string
  lastReconciledAt: string
  syncSource: string
  syncError: string
  lines: ProcurementLine[]
  statusHistory: StatusHistoryEntry[]
  dispatchHistory: ProcurementDispatchHistoryEntry[]
}

export type ProcurementDispatchHistoryEntry = {
  id: string
  dispatchStatus: string
  externalRequestReference: string
  retryable: boolean
  errorCode: string
  errorMessage: string
  observedAt: string
}

export type ProcurementReconcileResult = {
  requestId: string
  normalizedStatus: string
  rawStatus: string
  quantityProgression: string
  lastReconciledAt: string
  syncSource: string
}

export type ProcurementRequestCreateInput = {
  title: string
  projectId: string
  budgetCategoryId: string
  supplierId: string
  quotationId: string
  sourceType: string
  createdBy: string
  lines: {
    itemId: string
    quotationLineId: string
    requestedQuantity: number
    deliveryLocation: string
    accountingCategory: string
    note: string
  }[]
}

export type OCRJobSummary = {
  id: string
  fileName: string
  contentType: string
  status: string
  provider: string
  retryCount: number
  createdAt: string
  updatedAt: string
}

export type OCRJobListResponse = {
  rows: OCRJobSummary[]
}

export type OCRResultLine = {
  id: string
  itemId: string
  manufacturerName: string
  itemNumber: string
  itemDescription: string
  quantity: number
  leadTimeDays: number
  deliveryLocation: string
  budgetCategoryId: string
  accountingCategory: string
  supplierContact: string
  isUserConfirmed: boolean
  matchCandidates: OCRItemCandidate[]
}

export type OCRSupplierCandidate = {
  id: string
  name: string
  score: number
  matchReason: string
}

export type OCRItemCandidate = {
  itemId: string
  canonicalItemNumber: string
  description: string
  manufacturerName: string
  defaultSupplierId: string
  supplierAlias: string
  score: number
  matchReason: string
}

export type OCRLineAssistInput = {
  lineId: string
}

export type OCRLineAssistSuggestion = {
  lineId: string
  matchedItemId: string
  suggestedCanonicalNumber: string
  suggestedManufacturer: string
  suggestedCategoryKey: string
  suggestedAliasNumber: string
  confidence: number
  rationale: string
  candidates: OCRItemCandidate[]
}

export type OCRRegisterItemInput = {
  lineId: string
  canonicalItemNumber: string
  description: string
  manufacturerName: string
  categoryKey: string
  categoryName: string
  defaultSupplierId: string
  supplierAliasNumber: string
  unitsPerOrder: number
}

export type OCRJobDetailResponse = {
  id: string
  fileName: string
  contentType: string
  artifactPath: string
  status: string
  provider: string
  errorMessage: string
  retryCount: number
  supplierName: string
  supplierId: string
  supplierMatch: OCRSupplierCandidate[]
  quotationId: string
  quotationNumber: string
  issueDate: string
  procurementRequestId: string
  procurementBatchNumber: string
  rawPayload: string
  lines: OCRResultLine[]
}

export type OCRProcurementDraftCreateResult = {
  procurementRequestId: string
  procurementBatchNumber: string
  quotationId: string
  status: string
}

export type OCRLineUpdate = {
  id: string
  itemId: string
  deliveryLocation: string
  budgetCategoryId: string
  accountingCategory: string
  supplierContact: string
  isUserConfirmed: boolean
}

export type OCRReviewUpdateInput = {
  supplierId: string
  quotationNumber: string
  issueDate: string
  lines: OCRLineUpdate[]
}

export type MasterSyncResult = {
  syncType: string
  projectId: string
  projectKey: string
  status: string
  rowCount: number
  source: string
  triggeredBy: string
  syncedAt: string
}

export type MasterSyncRunEntry = {
  id: string
  syncType: string
  projectId: string
  projectKey: string
  status: string
  rowCount: number
  source: string
  triggeredBy: string
  errorMessage: string
  startedAt: string
  finishedAt: string
}

export type WebhookEventEntry = {
  id: string
  eventType: string
  externalRequestReference: string
  projectKey: string
  normalizedStatus: string
  rawStatus: string
  receivedAt: string
  processedAt: string
  processingError: string
}

export type OCRRetryResult = {
  id: string
  status: string
  retryCount: number
}

export type ProcurementSubmitResult = {
  requestId: string
  externalRequestReference: string
  dispatchStatus: string
  artifactDeleteStatus: string
}
