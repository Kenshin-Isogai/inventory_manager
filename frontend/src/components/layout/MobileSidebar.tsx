import { Link, NavLink, useNavigate } from 'react-router-dom'
import { LogOut } from 'lucide-react'
import { useAuthSession } from '@/hooks/useAuthSession'
import { useBootstrap } from '@/hooks/useBootstrap'
import { canAccessApp, clearStoredToken, localFallbackSession, resolveSessionUser, defaultPathForApp } from '@/lib/auth'
import { isFirebaseAuthConfigured } from '@/lib/firebaseAuth'
import { routes, APP_SECTION_LABELS } from '@/lib/routes'
import type { AppSection } from '@/types'
import { cn } from '@/lib/utils'
import { Button } from '@/components/ui/button'
import { Avatar, AvatarFallback } from '@/components/ui/avatar'
import { Separator } from '@/components/ui/separator'
import { Sheet, SheetContent } from '@/components/ui/sheet'
import { useSidebar } from './SidebarContext'
import { useSWRConfig } from 'swr'

const apps: AppSection[] = ['operator', 'inventory', 'procurement', 'inspector', 'admin']

export function MobileSidebar() {
  const { mobileOpen, setMobileOpen } = useSidebar()
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
    setMobileOpen(false)
    navigate('/auth/login', { replace: true })
  }

  const initials = activeSession.displayName
    .split(' ')
    .map((n) => n[0])
    .join('')
    .toUpperCase()

  return (
    <Sheet open={mobileOpen} onOpenChange={setMobileOpen}>
      <SheetContent side="left" className="w-[280px] p-0 bg-sidebar text-sidebar-foreground border-sidebar-border">
        {/* Brand */}
        <div className="px-4 py-4 border-b border-sidebar-border">
          <Link
            to={defaultPathForApp(activeSession.defaultApp)}
            className="flex items-center gap-2 hover:text-sidebar-accent transition-colors"
            onClick={() => setMobileOpen(false)}
          >
            <div className="w-8 h-8 rounded-lg bg-sidebar-accent/20 flex items-center justify-center text-xs font-bold text-sidebar-accent">
              IM
            </div>
            <span className="text-base font-semibold">Inventory Manager</span>
          </Link>
        </div>

        {/* Navigation */}
        <nav className="flex-1 overflow-y-auto py-3 px-2 space-y-5">
          {apps.map((app) =>
            canAccessApp(activeSession.roles, app) ? (
              <div key={app} className="space-y-1">
                <p className="px-3 pb-1 text-[10px] font-semibold text-sidebar-accent uppercase tracking-[0.15em]">
                  {APP_SECTION_LABELS[app]}
                </p>
                <div className="space-y-0.5">
                  {routes
                    .filter((route) => route.app === app)
                    .map((route) => {
                      const Icon = route.icon
                      return (
                        <NavLink
                          key={route.path}
                          to={route.path}
                          onClick={() => setMobileOpen(false)}
                          className={({ isActive }) =>
                            cn(
                              'flex items-center gap-3 px-3 py-2 rounded-md text-sm transition-all duration-150 relative',
                              isActive
                                ? 'bg-sidebar-muted text-sidebar-accent font-medium'
                                : 'text-sidebar-foreground/80 hover:bg-sidebar-muted/50 hover:text-sidebar-foreground'
                            )
                          }
                        >
                          {({ isActive }) => (
                            <>
                              {isActive && (
                                <span className="absolute left-0 top-1 bottom-1 w-[3px] bg-sidebar-accent rounded-r" />
                              )}
                              {Icon && <Icon className="w-4 h-4 flex-shrink-0" />}
                              <span className="truncate">{route.label}</span>
                            </>
                          )}
                        </NavLink>
                      )
                    })}
                </div>
              </div>
            ) : null
          )}
        </nav>

        {/* User Card */}
        <div className="border-t border-sidebar-border p-3 space-y-2">
          <div className="flex items-center gap-3 px-1">
            <Avatar className="h-8 w-8">
              <AvatarFallback className="text-xs bg-sidebar-muted text-sidebar-accent">{initials}</AvatarFallback>
            </Avatar>
            <div className="flex-1 min-w-0">
              <p className="text-sm font-medium truncate">{activeSession.displayName}</p>
              <p className="text-[11px] text-sidebar-foreground/60 truncate">
                {activeSession.roles.length > 0 ? activeSession.roles.join(', ') : 'no app roles'}
              </p>
            </div>
          </div>
          {authMode !== 'none' && (
            <>
              <Separator className="bg-sidebar-border" />
              <Button
                variant="ghost"
                size="sm"
                className="w-full justify-start text-sidebar-foreground/70 hover:text-sidebar-foreground hover:bg-sidebar-muted h-8 text-xs"
                onClick={() => void handleSignOut()}
              >
                <LogOut className="w-3.5 h-3.5 mr-2" />
                Sign out
              </Button>
            </>
          )}
        </div>
      </SheetContent>
    </Sheet>
  )
}
