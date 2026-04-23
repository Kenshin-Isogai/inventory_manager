import { Link, NavLink, Outlet, useSearchParams } from 'react-router-dom'
import { useSWRConfig } from 'swr'

import { useBootstrap } from '../hooks/useBootstrap'
import { useAuthSession } from '../hooks/useAuthSession'
import { canAccessApp, clearStoredToken, defaultPathForApp, localFallbackSession, resolveSessionUser } from '../lib/auth'
import { isFirebaseAuthConfigured } from '../lib/firebaseAuth'
import { routes } from '../lib/routes'
import type { AppSection } from '../types'

const apps: AppSection[] = ['operator', 'inventory', 'procurement', 'inspector', 'admin']

export function AppShell() {
  const [searchParams, setSearchParams] = useSearchParams()
  const { data: bootstrap } = useBootstrap()
  const { data: session } = useAuthSession()
  const { mutate } = useSWRConfig()
  const device = searchParams.get('device') ?? 'ER2'
  const scope = searchParams.get('scope') ?? 'powerboard'
  const authMode = bootstrap?.authMode ?? (isFirebaseAuthConfigured() ? 'enforced' : 'none')
  if (authMode !== 'none' && !session?.authenticated) {
    return null
  }
  const activeSession = authMode === 'none' ? localFallbackSession : resolveSessionUser(session)

  const updateContext = (key: 'device' | 'scope', value: string) => {
    const next = new URLSearchParams(searchParams)
    next.set(key, value)
    setSearchParams(next, { replace: true })
  }

  async function handleSignOut() {
    clearStoredToken()
    window.dispatchEvent(new Event('app-signout'))
    await mutate('auth-session')
  }

  return (
    <div className="app-shell">
      <aside className="sidebar">
        <Link to={defaultPathForApp(activeSession.defaultApp)} className="brand">
          Inventory Manager
        </Link>
        <p className="sidebar-copy">
          Local mode first. Cloud auth, OCR, and procurement adapters are staged behind the same shell.
        </p>
        <nav className="app-nav" aria-label="Applications">
          {apps.map((app) =>
            canAccessApp(activeSession.roles, app) ? (
              <div key={app} className="app-group">
                <p className="app-group-title">{app}</p>
                {routes
                  .filter((route) => route.app === app)
                  .map((route) => (
                    <NavLink key={route.path} to={route.path} className="nav-link">
                      <span>{route.label}</span>
                      <small>{route.description}</small>
                    </NavLink>
                  ))}
              </div>
            ) : null
          )}
        </nav>
      </aside>

      <main className="content">
        <header className="context-bar">
          <div>
            <p className="eyebrow">App Context</p>
            <h1>Inventory Operations Skeleton</h1>
          </div>
          <div className="context-controls">
            <label>
              <span>Device</span>
              <input value={device} onChange={(event) => updateContext('device', event.target.value)} />
            </label>
            <label>
              <span>Scope</span>
              <input value={scope} onChange={(event) => updateContext('scope', event.target.value)} />
            </label>
          </div>
          <div className="session-card">
            <strong>{activeSession.displayName}</strong>
            <span>{activeSession.roles.join(', ') || 'no app roles yet'}</span>
            {authMode !== 'none' ? (
              <button type="button" className="secondary-button" onClick={() => void handleSignOut()}>
                Sign out
              </button>
            ) : null}
          </div>
        </header>
        <Outlet />
      </main>
    </div>
  )
}
