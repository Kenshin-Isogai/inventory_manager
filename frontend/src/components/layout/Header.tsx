import { useLocation, useSearchParams } from 'react-router-dom'
import { ChevronRight } from 'lucide-react'
import { routes, APP_SECTION_LABELS } from '@/lib/routes'
import { Input } from '@/components/ui/input'
import { Sidebar } from './Sidebar'

const PORTAL_PATHS = ['/auth/login', '/auth/verify-email', '/auth/register', '/auth/pending', '/auth/rejected', '/app/portal']
const ADMIN_PATHS = ['/app/admin']

function Breadcrumb() {
  const location = useLocation()
  const pathname = location.pathname

  // Find the matching route
  const route = routes.find((r) => r.path === pathname)

  if (!route) {
    return (
      <div className="flex items-center gap-1">
        <span className="text-xs font-medium text-muted-foreground uppercase tracking-wide">Portal</span>
      </div>
    )
  }

  return (
    <div className="flex items-center gap-1">
      <span className="text-xs font-medium text-muted-foreground uppercase tracking-wide">
        {APP_SECTION_LABELS[route.app]}
      </span>
      <ChevronRight className="w-4 h-4 text-muted-foreground" />
      <span className="text-sm font-medium">{route.label}</span>
    </div>
  )
}

function ContextControls() {
  const [searchParams, setSearchParams] = useSearchParams()
  const location = useLocation()
  const pathname = location.pathname

  // Only show context controls on relevant pages (not portal, auth, admin)
  const showContextControls = !PORTAL_PATHS.includes(pathname) && !ADMIN_PATHS.some((p) => pathname.startsWith(p))

  if (!showContextControls) return null

  const device = searchParams.get('device') ?? 'ER2'
  const scope = searchParams.get('scope') ?? 'powerboard'

  const updateContext = (key: 'device' | 'scope', value: string) => {
    const next = new URLSearchParams(searchParams)
    next.set(key, value)
    setSearchParams(next, { replace: true })
  }

  return (
    <div className="flex items-center gap-4">
      <div className="flex items-center gap-2">
        <label htmlFor="device" className="text-sm font-medium text-foreground">
          Device
        </label>
        <Input
          id="device"
          value={device}
          onChange={(e) => updateContext('device', e.target.value)}
          className="w-32"
        />
      </div>
      <div className="flex items-center gap-2">
        <label htmlFor="scope" className="text-sm font-medium text-foreground">
          Scope
        </label>
        <Input
          id="scope"
          value={scope}
          onChange={(e) => updateContext('scope', e.target.value)}
          className="w-32"
        />
      </div>
    </div>
  )
}

export function Header() {
  return (
    <header className="bg-card border-b border-border sticky top-0 z-40">
      <div className="flex items-center justify-between h-16 px-4 sm:px-6">
        <div className="flex items-center gap-4">
          <Sidebar />
          <Breadcrumb />
        </div>
        <ContextControls />
      </div>
    </header>
  )
}
