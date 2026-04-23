import { useLocation, useSearchParams } from 'react-router-dom'
import { ChevronRight, Menu } from 'lucide-react'
import { routes, APP_SECTION_LABELS } from '@/lib/routes'
import { Input } from '@/components/ui/input'
import { Button } from '@/components/ui/button'
import { useSidebar } from './SidebarContext'

const PORTAL_PATHS = ['/app/portal']
const ADMIN_PATHS = ['/app/admin']

function Breadcrumb() {
  const location = useLocation()
  const pathname = location.pathname

  const route = routes.find((r) => r.path === pathname)

  if (!route) {
    return (
      <div className="flex items-center gap-1.5">
        <span className="text-sm font-medium text-foreground">Portal</span>
      </div>
    )
  }

  return (
    <div className="flex items-center gap-1.5">
      <span className="text-xs font-medium text-muted-foreground uppercase tracking-wide">
        {APP_SECTION_LABELS[route.app]}
      </span>
      <ChevronRight className="w-3.5 h-3.5 text-muted-foreground/50" />
      <span className="text-sm font-medium text-foreground">{route.label}</span>
    </div>
  )
}

function ContextControls() {
  const [searchParams, setSearchParams] = useSearchParams()
  const location = useLocation()
  const pathname = location.pathname

  const showContextControls = !PORTAL_PATHS.includes(pathname) && !ADMIN_PATHS.some((p) => pathname.startsWith(p))

  if (!showContextControls) return null

  const device = searchParams.get('device') ?? ''
  const scope = searchParams.get('scope') ?? ''

  const updateContext = (key: 'device' | 'scope', value: string) => {
    const next = new URLSearchParams(searchParams)
    if (value.trim() === '') {
      next.delete(key)
    } else {
      next.set(key, value)
    }
    setSearchParams(next, { replace: true })
  }

  return (
    <div className="hidden sm:flex items-center gap-3">
      <div className="flex items-center gap-1.5">
        <label htmlFor="ctx-device" className="text-xs font-medium text-muted-foreground uppercase tracking-wide">
          Device
        </label>
        <Input
          id="ctx-device"
          value={device}
          onChange={(e) => updateContext('device', e.target.value)}
          placeholder="Not selected"
          className="h-8 w-24 text-sm"
        />
      </div>
      <div className="flex items-center gap-1.5">
        <label htmlFor="ctx-scope" className="text-xs font-medium text-muted-foreground uppercase tracking-wide">
          Scope
        </label>
        <Input
          id="ctx-scope"
          value={scope}
          onChange={(e) => updateContext('scope', e.target.value)}
          placeholder="Not selected"
          className="h-8 w-28 text-sm"
        />
      </div>
    </div>
  )
}

export function Header() {
  const { setMobileOpen } = useSidebar()

  return (
    <header className="bg-card/80 backdrop-blur-sm border-b border-border sticky top-0 z-40">
      <div className="flex items-center justify-between h-14 px-4 sm:px-6">
        <div className="flex items-center gap-3">
          {/* Mobile-only hamburger */}
          <Button
            variant="ghost"
            size="icon"
            className="md:hidden h-8 w-8"
            onClick={() => setMobileOpen(true)}
          >
            <Menu className="h-5 w-5" />
          </Button>
          <Breadcrumb />
        </div>
        <ContextControls />
      </div>
    </header>
  )
}
