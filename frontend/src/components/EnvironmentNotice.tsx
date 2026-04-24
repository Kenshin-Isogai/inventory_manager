import { useBootstrap } from '@/hooks/useBootstrap'
import { config } from '@/lib/config'
import { isFirebaseAuthConfigured } from '@/lib/firebaseAuth'
import { runtimeConfig } from '@/lib/runtimeConfig'

type EnvironmentNoticeProps = {
  className?: string
}

function normalizeOrigin(value: string) {
  try {
    return new URL(value).origin
  } catch {
    return value
  }
}

export function EnvironmentNotice({ className }: EnvironmentNoticeProps) {
  const { data: bootstrap } = useBootstrap()
  const firebaseConfigured = isFirebaseAuthConfigured()
  const apiConfigured = runtimeConfig.apiBaseUrlConfigured
  const hostname = typeof window === 'undefined' ? '' : window.location.hostname
  const isLocalHost = hostname === 'localhost' || hostname === '127.0.0.1'
  const apiOrigin = normalizeOrigin(config.apiBaseUrl) || 'same-origin'

  let message = `API ${apiOrigin}.`

  if (bootstrap) {
    if (bootstrap.authMode === 'none') {
      message = `Authentication is not enforced. API ${apiOrigin}.`
    } else {
      message = `Auth ${bootstrap.authMode}/${bootstrap.authProvider}. API ${apiOrigin}.`
    }
  } else if (!isLocalHost && (!apiConfigured || !firebaseConfigured)) {
    const missing = [
      !apiConfigured ? 'API_BASE_URL' : null,
      !firebaseConfigured ? 'FIREBASE_*' : null,
    ].filter(Boolean).join(', ')
    message = `Runtime config is missing: ${missing}.`
  } else if (!isLocalHost) {
    message = `Backend bootstrap is unavailable. Check API_BASE_URL, backend reachability, and CORS.`
  }

  return <p className={className}>{message}</p>
}
