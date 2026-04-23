import { Navigate, Outlet, useLocation } from 'react-router-dom'

import { useBootstrap } from '../hooks/useBootstrap'
import { useAuthSession } from '../hooks/useAuthSession'
import { canAccessApp, defaultPathForApp, localFallbackSession, resolveSessionUser } from '../lib/auth'
import type { AppSection } from '../types'

type AuthGateProps = {
  app: AppSection
}

export function AuthGate({ app }: AuthGateProps) {
  const location = useLocation()
  const { data: bootstrap } = useBootstrap()
  const { data: session } = useAuthSession()
  const authMode = bootstrap?.authMode ?? 'none'
  const activeSession = authMode === 'none' ? localFallbackSession : resolveSessionUser(session)

  if (authMode !== 'none') {
    if (!session?.authenticated) {
      return <Navigate to="/auth/login" replace state={{ from: location.pathname }} />
    }
    if (!session.user.emailVerified) {
      return <Navigate to="/auth/verify-email" replace />
    }
    if (session.user.status === 'pending') {
      return <Navigate to="/auth/pending" replace />
    }
    if (session.user.status === 'rejected') {
      return <Navigate to="/auth/rejected" replace />
    }
    if (session.user.status === 'unregistered' || session.user.registrationNeeded) {
      return <Navigate to="/auth/register" replace />
    }
  }

  if (!canAccessApp(activeSession.roles, app)) {
    return <Navigate to={defaultPathForApp(activeSession.defaultApp)} replace state={{ from: location.pathname }} />
  }

  return <Outlet />
}
