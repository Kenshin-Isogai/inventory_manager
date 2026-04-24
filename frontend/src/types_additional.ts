// Additional spec 042401 types

export type ScopeOverviewRow = {
  deviceKey: string
  deviceName: string
  scopeId: string
  scopeKey: string
  scopeName: string
  scopeType: string
  parentScopeId: string
  status: string
  plannedStartAt: string
  requirementsCount: number
  reservationsCount: number
  shortageItemCount: number
  ownerDepartment: string
}

export type ScopeOverviewResponse = {
  rows: ScopeOverviewRow[]
}

export type ItemFlowEntry = {
  date: string
  eventType: string
  quantityDelta: number
  runningBalance: number
  sourceType: string
  sourceRef: string
  note: string
  locationCode: string
}

export type ItemFlowResponse = {
  itemId: string
  itemNumber: string
  rows: ItemFlowEntry[]
}

export type EnhancedShortageRow = {
  device: string
  scope: string
  manufacturer: string
  itemNumber: string
  description: string
  itemId: string
  requiredQuantity: number
  reservedQuantity: number
  availableQuantity: number
  rawShortage: number
  inRequestFlowQuantity: number
  orderedQuantity: number
  receivedQuantity: number
  actionableShortage: number
  relatedProcurementRequests: string[]
}

export type EnhancedShortageResponse = {
  coverageRule: string
  rows: EnhancedShortageRow[]
}

export type DelayedArrival = {
  expectedDate: string
  quantity: number
  purchaseOrderNumber: string
  purchaseOrderLineId: string
}

export type ShortageTimelineEntry = {
  itemId: string
  itemNumber: string
  manufacturer: string
  description: string
  requiredQuantity: number
  availableByStart: number
  shortageAtStart: number
  delayedArrivals: DelayedArrival[]
}

export type ShortageTimelineResponse = {
  device: string
  scope: string
  plannedStartAt: string
  rows: ShortageTimelineEntry[]
}

export type ArrivalCalendarItem = {
  itemId: string
  itemNumber: string
  manufacturer: string
  description: string
  quantity: number
  purchaseOrderNumber: string
  purchaseOrderLineId: string
  quotationNumber: string
  supplierName: string
}

export type ArrivalCalendarDay = {
  date: string
  items: ArrivalCalendarItem[]
}

export type ArrivalCalendarResponse = {
  yearMonth: string
  days: ArrivalCalendarDay[]
}

export type ItemSuggestion = {
  id: string
  itemNumber: string
  description: string
  manufacturer: string
  category: string
}

export type ItemSuggestionResponse = {
  rows: ItemSuggestion[]
}

export type CategorySuggestionResponse = {
  rows: { key: string; name: string }[]
}

export type BulkReservationPreviewRow = {
  itemId: string
  itemNumber: string
  manufacturer: string
  description: string
  requiredQuantity: number
  allocFromStock: number
  allocFromStockLocs: { locationCode: string; quantity: number }[]
  allocFromOrders: number
  allocFromOrderLocs: {
    purchaseOrderLineId: string
    purchaseOrderNumber: string
    expectedArrival: string
    quantity: number
  }[]
  unallocated: number
}

export type BulkReservationPreviewResponse = {
  scopeId: string
  rows: BulkReservationPreviewRow[]
}

export type BulkReservationConfirmInput = {
  scopeId: string
  actorId: string
  rows: {
    itemId: string
    stockAllocations: { locationCode: string; quantity: number }[]
    orderAllocations: {
      purchaseOrderLineId: string
      purchaseOrderNumber: string
      expectedArrival: string
      quantity: number
    }[]
    purpose: string
    priority: string
    neededByAt: string
  }[]
}

export type BulkReservationResult = {
  created: number
  ids: string[]
}

export type RequirementsImportPreviewRow = {
  rowNumber: number
  deviceKey: string
  scopeKey: string
  itemNumber: string
  manufacturer: string
  description: string
  quantity: number
  status: string
  message: string
  itemId: string
  scopeId: string
  itemRegistered: boolean
}

export type RequirementsImportPreviewResponse = {
  fileName: string
  rows: RequirementsImportPreviewRow[]
}

export type RequirementsImportResult = {
  created: number
  updated: number
  skipped: number
  errored: number
}
