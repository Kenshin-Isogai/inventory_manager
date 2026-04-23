export type RuntimeConfig = {
  apiBaseUrl: string
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

export const runtimeConfig: RuntimeConfig = {
  apiBaseUrl: runtime.apiBaseUrl ?? 'http://localhost:8080',
  firebaseApiKey: runtime.firebaseApiKey ?? '',
  firebaseAuthDomain: runtime.firebaseAuthDomain ?? '',
  firebaseProjectId: runtime.firebaseProjectId ?? '',
  firebaseAppId: runtime.firebaseAppId ?? '',
}
