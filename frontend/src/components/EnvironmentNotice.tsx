import { useBootstrap } from '@/hooks/useBootstrap'
import { config } from '@/lib/config'
import { isFirebaseAuthConfigured } from '@/lib/firebaseAuth'

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
  const hostname = typeof window === 'undefined' ? '' : window.location.hostname
  const isLocalHost = hostname === 'localhost' || hostname === '127.0.0.1'
  const apiOrigin = normalizeOrigin(config.apiBaseUrl)

  let message = `API ${apiOrigin}.`

  if (bootstrap) {
    if (bootstrap.authMode === 'none') {
      message = `Local auth mode is active. API ${apiOrigin}.`
    } else {
      message = `Auth ${bootstrap.authMode}/${bootstrap.authProvider}. API ${apiOrigin}.`
    }
  } else if (!isLocalHost && !firebaseConfigured) {
    message = `Cloud auth runtime config is missing. Check API_BASE_URL and FIREBASE_* env vars.`
  } else if (!isLocalHost) {
    message = `Backend bootstrap is unavailable. Check API_BASE_URL, backend reachability, and CORS.`
  }

  return <p className={className}>{message}</p>
}
