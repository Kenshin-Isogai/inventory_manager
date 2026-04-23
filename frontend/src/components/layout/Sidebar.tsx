import { Link, NavLink, useNavigate } from 'react-router-dom'
import { Menu, LogOut } from 'lucide-react'
import { useAuthSession } from '@/hooks/useAuthSession'
import { useBootstrap } from '@/hooks/useBootstrap'
import { canAccessApp, clearStoredToken, localFallbackSession, resolveSessionUser, defaultPathForApp } from '@/lib/auth'
import { isFirebaseAuthConfigured } from '@/lib/firebaseAuth'
import { routes, APP_SECTION_LABELS } from '@/lib/routes'
import type { AppSection } from '@/types'
import { Button } from '@/components/ui/button'
import { Avatar, AvatarFallback } from '@/components/ui/avatar'
import { Separator } from '@/components/ui/separator'
import { Sheet, SheetContent, SheetTrigger } from '@/components/ui/sheet'
import { useSWRConfig } from 'swr'

const apps: AppSection[] = ['operator', 'inventory', 'procurement', 'inspector', 'admin']

function SidebarContent() {
  const navigate = useNavigate()
  const { data: bootstrap } = useBootstrap()
  const { data: session } = useAuthSession()
  const { mutate } = useSWRConfig()
  const authMode = bootstrap?.authMode ?? (isFirebaseAuthConfigured() ? 'enforced' : 'none')
  const activeSession = authMode === 'none' ? localFallbackSession : resolveSessionUser(session)

  async function handleSignOut() {
    clearStoredToken()
    window.dispatchEvent(new Event('app-signout'))
    await mutate('auth-session')
    navigate('/auth/login', { replace: true })
  }

  return (
    <div className="flex flex-col h-full bg-sidebar text-sidebar-foreground">
      {/* Brand */}
      <div className="p-6 border-b border-sidebar-border">
        <Link
          to={defaultPathForApp(activeSession.defaultApp)}
          className="flex items-center gap-2 text-lg font-semibold hover:text-sidebar-accent transition-colors"
        >
          <div className="w-8 h-8 rounded bg-sidebar-accent/20 flex items-center justify-center text-xs font-bold text-sidebar-accent">
            IM
          </div>
          Inventory Manager
        </Link>
      </div>

      {/* Navigation */}
      <nav className="flex-1 overflow-y-auto py-4 px-3 space-y-6">
        {apps.map((app) =>
          canAccessApp(activeSession.roles, app) ? (
            <div key={app} className="space-y-2">
              <p className="px-3 text-xs font-semibold text-sidebar-accent uppercase tracking-widest">
                {APP_SECTION_LABELS[app]}
              </p>
              <div className="space-y-1">
                {routes
                  .filter((route) => route.app === app)
                  .map((route) => {
                    const Icon = route.icon
                    return (
                      <NavLink
                        key={route.path}
                        to={route.path}
                        className={({ isActive }) =>
                          `flex items-center gap-3 px-3 py-2 rounded-md text-sm transition-colors relative ${
                            isActive
                              ? 'bg-sidebar-muted text-sidebar-accent font-medium before:absolute before:left-0 before:top-0 before:bottom-0 before:w-1 before:bg-sidebar-accent before:rounded-r'
                              : 'text-sidebar-foreground hover:bg-sidebar-muted/50'
                          }`
                        }
                      >
                        {Icon && <Icon className="w-4 h-4 flex-shrink-0" />}
                        <span className="truncate">{route.label}</span>
                      </NavLink>
                    )
                  })}
              </div>
            </div>
          ) : null
        )}
      </nav>

      {/* User Card */}
      <div className="p-4 border-t border-sidebar-border space-y-3">
        <div className="flex items-center gap-3">
          <Avatar className="h-8 w-8">
            <AvatarFallback className="text-xs">
              {activeSession.displayName
                .split(' ')
                .map((n) => n[0])
                .join('')
                .toUpperCase()}
            </AvatarFallback>
          </Avatar>
          <div className="flex-1 min-w-0">
            <p className="text-sm font-medium truncate">{activeSession.displayName}</p>
            <p className="text-xs text-sidebar-foreground/70 truncate">
              {activeSession.roles.length > 0 ? activeSession.roles.join(', ') : 'no app roles'}
            </p>
          </div>
        </div>
        <Separator className="bg-sidebar-border" />
        {authMode !== 'none' ? (
          <Button
            variant="ghost"
            size="sm"
            className="w-full justify-start text-sidebar-foreground hover:bg-sidebar-muted"
            onClick={() => void handleSignOut()}
          >
            <LogOut className="w-4 h-4 mr-2" />
            Sign out
          </Button>
        ) : null}
      </div>
    </div>
  )
}

export function Sidebar() {
  return (
    <>
      {/* Desktop Sidebar */}
      <aside className="hidden md:flex w-[260px] flex-shrink-0 border-r border-sidebar-border">
        <SidebarContent />
      </aside>

      {/* Mobile Sheet */}
      <Sheet>
        <SheetTrigger asChild>
          <Button variant="ghost" size="icon" className="md:hidden">
            <Menu className="h-5 w-5" />
          </Button>
        </SheetTrigger>
        <SheetContent side="left" className="w-[260px] p-0">
          <SidebarContent />
        </SheetContent>
      </Sheet>
    </>
  )
}
