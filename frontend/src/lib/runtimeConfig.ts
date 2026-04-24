export type RuntimeConfig = {
  apiBaseUrl: string
  apiBaseUrlConfigured: boolean
  firebaseApiKey: string
  firebaseAuthDomain: string
  firebaseProjectId: string
  firebaseAppId: string
}

declare global {
  interface Window {
    __APP_CONFIG__?: Partial<RuntimeConfig>
  }
}

const runtime = window.__APP_CONFIG__ ?? {}
const runtimeApiBaseUrl = typeof runtime.apiBaseUrl === 'string' ? runtime.apiBaseUrl.trim() : ''

function normalizeApiBaseUrl(value: string) {
  const resolved = value || window.location.origin
  return resolved.replace(/\/+$/, '')
}

export const runtimeConfig: RuntimeConfig = {
  apiBaseUrl: normalizeApiBaseUrl(runtimeApiBaseUrl),
  apiBaseUrlConfigured: runtimeApiBaseUrl.length > 0,
  firebaseApiKey: runtime.firebaseApiKey ?? '',
  firebaseAuthDomain: runtime.firebaseAuthDomain ?? '',
  firebaseProjectId: runtime.firebaseProjectId ?? '',
  firebaseAppId: runtime.firebaseAppId ?? '',
}
