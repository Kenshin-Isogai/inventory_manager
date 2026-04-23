import type { AppSection, AuthSessionResponse, RoleKey } from '../types'

export type SessionUser = {
  id: string
  displayName: string
  roles: RoleKey[]
  defaultApp: AppSection
  email?: string
  status?: string
  emailVerified?: boolean
}

export const localFallbackSession: SessionUser = {
  id: 'local-user',
  displayName: 'Local Operator',
  roles: ['operator', 'inventory', 'admin'],
  defaultApp: 'operator',
  email: 'operator@example.local',
  status: 'active',
  emailVerified: true,
}

export function canAccessApp(roles: RoleKey[], app: AppSection) {
  if (roles.includes('admin')) {
    return true
  }
  if (app === 'inspector') {
    return roles.includes('receiving_inspector')
  }
  return roles.includes(app)
}

export function defaultPathForApp(app: AppSection) {
  switch (app) {
    case 'operator':
      return '/app/operator/requirements'
    case 'inventory':
      return '/app/inventory/items'
    case 'procurement':
      return '/app/procurement/requests'
    case 'inspector':
      return '/app/inspector/arrivals'
    case 'admin':
      return '/app/admin/master'
    default:
      return '/app/operator/requirements'
  }
}

export const LOCAL_LOGIN_PROFILES = [
  { label: 'Admin', token: 'local-admin-token', description: 'Users, roles, and master data' },
  { label: 'Operator', token: 'local-operator-token', description: 'Requirements, reservations, and shortages' },
  { label: 'Inventory', token: 'local-inventory-token', description: 'Inventory balances and adjustments' },
  { label: 'Procurement', token: 'local-procurement-token', description: 'OCR and procurement request tracking' },
  { label: 'Inspector', token: 'local-inspector-token', description: 'Arrival confirmation and receiving' },
] as const

const AUTH_TOKEN_KEY = 'inventory_manager.auth_token'

export function getStoredToken() {
  return window.localStorage.getItem(AUTH_TOKEN_KEY) ?? ''
}

export function setStoredToken(token: string) {
  const normalized = token.trim()
  if (normalized === '') {
    window.localStorage.removeItem(AUTH_TOKEN_KEY)
    return
  }
  window.localStorage.setItem(AUTH_TOKEN_KEY, normalized)
}

export function clearStoredToken() {
  window.localStorage.removeItem(AUTH_TOKEN_KEY)
}

export function authorizationHeaders() {
  const token = getStoredToken()
  if (!token) {
    return {} as Record<string, string>
  }
  return { Authorization: `Bearer ${token}` } as Record<string, string>
}

export function resolveSessionUser(session?: AuthSessionResponse | null): SessionUser {
  if (!session?.authenticated || session.user.status !== 'active') {
    return localFallbackSession
  }
  const roles = session.user.roles.length > 0 ? session.user.roles : localFallbackSession.roles
  const defaultApp = roles.includes('admin')
    ? 'admin'
    : roles.includes('receiving_inspector')
      ? 'inspector'
      : (roles.find((role) => ['operator', 'inventory', 'procurement', 'admin'].includes(role)) as AppSection | undefined) ?? 'operator'
  return {
    id: session.user.userId || session.user.email || 'session-user',
    displayName: session.user.displayName || session.user.email || localFallbackSession.displayName,
    email: session.user.email,
    roles,
    defaultApp,
    status: session.user.status,
    emailVerified: session.user.emailVerified,
  }
}
