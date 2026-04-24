import type {
  ScopeOverviewResponse,
  ItemFlowResponse,
  EnhancedShortageResponse,
  ShortageTimelineResponse,
  ArrivalCalendarResponse,
  ItemSuggestionResponse,
  CategorySuggestionResponse,
  BulkReservationPreviewResponse,
  BulkReservationConfirmInput,
  BulkReservationResult,
  CSVImportApplyResult,
  ImportPreviewResult,
  RequirementsImportPreviewResponse,
  RequirementsImportResult,
} from '../types'
import { config } from './config'
import { authorizationHeaders } from './auth'

async function fetchAPI<T>(path: string): Promise<T> {
  const response = await fetch(`${config.apiBaseUrl}${path}`, {
    headers: authorizationHeaders(),
  })
  if (!response.ok) {
    throw new Error(`request failed: ${response.status}`)
  }
  const payload = await response.json()
  return payload.data as T
}

async function postAPI<T>(path: string, body?: unknown): Promise<T> {
  const response = await fetch(`${config.apiBaseUrl}${path}`, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json', ...authorizationHeaders() },
    body: body ? JSON.stringify(body) : undefined,
  })
  if (!response.ok) {
    throw new Error(`request failed: ${response.status}`)
  }
  const payload = await response.json()
  return payload.data as T
}

export async function fetchScopeOverview(device?: string): Promise<ScopeOverviewResponse> {
  const params = new URLSearchParams()
  if (device) params.set('device', device)
  const qs = params.size > 0 ? `?${params.toString()}` : ''
  return fetchAPI<ScopeOverviewResponse>(`/api/v1/operator/scope-overview${qs}`)
}

export async function fetchItemFlow(itemId: string): Promise<ItemFlowResponse> {
  return fetchAPI<ItemFlowResponse>(`/api/v1/inventory/items/${itemId}/flow`)
}

export async function fetchEnhancedShortages(
  device?: string,
  scope?: string,
  coverageRule?: string,
): Promise<EnhancedShortageResponse> {
  const params = new URLSearchParams()
  if (device) params.set('device', device)
  if (scope) params.set('scope', scope)
  if (coverageRule) params.set('coverageRule', coverageRule)
  const qs = params.size > 0 ? `?${params.toString()}` : ''
  return fetchAPI<EnhancedShortageResponse>(`/api/v1/operator/shortages/enhanced${qs}`)
}

export async function fetchShortageTimeline(
  device: string,
  scope: string,
): Promise<ShortageTimelineResponse> {
  const params = new URLSearchParams({ device, scope })
  return fetchAPI<ShortageTimelineResponse>(`/api/v1/operator/shortages/timeline?${params.toString()}`)
}

export async function fetchArrivalCalendar(yearMonth: string): Promise<ArrivalCalendarResponse> {
  return fetchAPI<ArrivalCalendarResponse>(`/api/v1/inventory/arrivals/calendar?yearMonth=${yearMonth}`)
}

export async function fetchItemSuggest(query: string): Promise<ItemSuggestionResponse> {
  return fetchAPI<ItemSuggestionResponse>(`/api/v1/admin/master-data/items/suggest?q=${encodeURIComponent(query)}`)
}

export async function fetchCategorySuggest(query: string): Promise<CategorySuggestionResponse> {
  return fetchAPI<CategorySuggestionResponse>(`/api/v1/admin/master-data/categories/suggest?q=${encodeURIComponent(query)}`)
}

export async function fetchBulkReservationPreview(scopeId: string): Promise<BulkReservationPreviewResponse> {
  return fetchAPI<BulkReservationPreviewResponse>(`/api/v1/operator/reservations/bulk-preview?scopeId=${scopeId}`)
}

export async function confirmBulkReservation(input: BulkReservationConfirmInput): Promise<BulkReservationResult> {
  return postAPI<BulkReservationResult>('/api/v1/operator/reservations/bulk-confirm', input)
}

export async function fetchInventorySnapshotAtDate(
  device?: string,
  scope?: string,
  itemId?: string,
  targetDate?: string,
): Promise<unknown> {
  const params = new URLSearchParams()
  if (device) params.set('device', device)
  if (scope) params.set('scope', scope)
  if (itemId) params.set('itemId', itemId)
  if (targetDate) params.set('target_date', targetDate)
  const qs = params.size > 0 ? `?${params.toString()}` : ''
  return fetchAPI<unknown>(`/api/v1/inventory/snapshot${qs}`)
}

export async function exportReservationsCSV(device?: string, scope?: string): Promise<void> {
  const params = new URLSearchParams()
  if (device) params.set('device', device)
  if (scope) params.set('scope', scope)
  const qs = params.size > 0 ? `?${params.toString()}` : ''
  const response = await fetch(`${config.apiBaseUrl}/api/v1/operator/reservations/export${qs}`, {
    headers: authorizationHeaders(),
  })
  if (!response.ok) throw new Error(`export failed: ${response.status}`)
  const blob = await response.blob()
  const url = URL.createObjectURL(blob)
  const a = document.createElement('a')
  a.href = url
  a.download = `reservations${device ? `_${device}` : ''}${scope ? `_${scope}` : ''}.csv`
  a.click()
  URL.revokeObjectURL(url)
}

export async function exportRequirementsCSV(device?: string, scope?: string): Promise<void> {
  const params = new URLSearchParams()
  if (device) params.set('device', device)
  if (scope) params.set('scope', scope)
  const qs = params.size > 0 ? `?${params.toString()}` : ''
  const response = await fetch(`${config.apiBaseUrl}/api/v1/operator/requirements/export${qs}`, {
    headers: authorizationHeaders(),
  })
  if (!response.ok) throw new Error(`export failed: ${response.status}`)
  const blob = await response.blob()
  const url = URL.createObjectURL(blob)
  const a = document.createElement('a')
  a.href = url
  a.download = `requirements${device ? `_${device}` : ''}${scope ? `_${scope}` : ''}.csv`
  a.click()
  URL.revokeObjectURL(url)
}

export async function previewRequirementsImport(file: File): Promise<RequirementsImportPreviewResponse> {
  const form = new FormData()
  form.append('file', file)
  const response = await fetch(`${config.apiBaseUrl}/api/v1/operator/requirements/import/preview`, {
    method: 'POST',
    headers: authorizationHeaders(),
    body: form,
  })
  if (!response.ok) throw new Error(`preview failed: ${response.status}`)
  const payload = await response.json()
  return payload.data as RequirementsImportPreviewResponse
}

export async function applyRequirementsImport(file: File): Promise<RequirementsImportResult> {
  const form = new FormData()
  form.append('file', file)
  const response = await fetch(`${config.apiBaseUrl}/api/v1/operator/requirements/import`, {
    method: 'POST',
    headers: authorizationHeaders(),
    body: form,
  })
  if (!response.ok) throw new Error(`import failed: ${response.status}`)
  const payload = await response.json()
  return payload.data as RequirementsImportResult
}

async function postCSVPreview(path: string, file: File): Promise<ImportPreviewResult> {
  const form = new FormData()
  form.append('file', file)
  const response = await fetch(`${config.apiBaseUrl}${path}`, {
    method: 'POST',
    headers: authorizationHeaders(),
    body: form,
  })
  if (!response.ok) {
    const payload = await response.json().catch(() => null)
    throw new Error((payload as { error?: string } | null)?.error ?? `preview failed: ${response.status}`)
  }
  const payload = await response.json()
  return payload.data as ImportPreviewResult
}

async function postCSVApply(path: string, file: File, actorId: string): Promise<CSVImportApplyResult> {
  const form = new FormData()
  form.append('file', file)
  form.append('actorId', actorId)
  const response = await fetch(`${config.apiBaseUrl}${path}`, {
    method: 'POST',
    headers: authorizationHeaders(),
    body: form,
  })
  if (!response.ok) {
    const payload = await response.json().catch(() => null)
    throw new Error((payload as { error?: string } | null)?.error ?? `import failed: ${response.status}`)
  }
  const payload = await response.json()
  return payload.data as CSVImportApplyResult
}

export function previewReservationsImport(file: File) {
  return postCSVPreview('/api/v1/operator/reservations/import/preview', file)
}

export function applyReservationsImport(file: File, actorId: string) {
  return postCSVApply('/api/v1/operator/reservations/import', file, actorId)
}

export function previewAllocationsImport(file: File) {
  return postCSVPreview('/api/v1/operator/reservations/allocations/import/preview', file)
}

export function applyAllocationsImport(file: File, actorId: string) {
  return postCSVApply('/api/v1/operator/reservations/allocations/import', file, actorId)
}

export function previewInventoryOperationImport(operation: 'adjust' | 'receive' | 'move', file: File) {
  return postCSVPreview(`/api/v1/inventory/operations/${operation}/import/preview`, file)
}

export function applyInventoryOperationImport(operation: 'adjust' | 'receive' | 'move', file: File, actorId: string) {
  return postCSVApply(`/api/v1/inventory/operations/${operation}/import`, file, actorId)
}
