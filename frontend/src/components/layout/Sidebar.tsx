import { Link, NavLink, useNavigate } from 'react-router-dom'
import { LogOut, PanelLeftClose, PanelLeft } from 'lucide-react'
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
import { Tooltip, TooltipContent, TooltipProvider, TooltipTrigger } from '@/components/ui/tooltip'
import { useSidebar } from './SidebarContext'
import { useSWRConfig } from 'swr'

const apps: AppSection[] = ['operator', 'inventory', 'procurement', 'inspector', 'admin']

export function Sidebar() {
  const { collapsed, toggle } = useSidebar()
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

  const initials = activeSession.displayName
    .split(' ')
    .map((n) => n[0])
    .join('')
    .toUpperCase()

  return (
    <TooltipProvider delayDuration={0}>
      <aside
        className={cn(
          'hidden md:flex flex-col h-full bg-sidebar text-sidebar-foreground border-r border-sidebar-border transition-all duration-300 ease-in-out flex-shrink-0',
          collapsed ? 'w-[68px]' : 'w-[260px]'
        )}
      >
        {/* Brand + Toggle */}
        <div className={cn('flex items-center border-b border-sidebar-border', collapsed ? 'px-3 py-4 justify-center' : 'px-4 py-4 justify-between')}>
          <Link
            to={defaultPathForApp(activeSession.defaultApp)}
            className="flex items-center gap-2 hover:text-sidebar-accent transition-colors"
          >
            <div className="w-8 h-8 rounded-lg bg-sidebar-accent/20 flex items-center justify-center text-xs font-bold text-sidebar-accent flex-shrink-0">
              IM
            </div>
            {!collapsed && <span className="text-base font-semibold truncate">Inventory Manager</span>}
          </Link>
          {!collapsed && (
            <Button
              variant="ghost"
              size="icon"
              className="h-7 w-7 text-sidebar-foreground/60 hover:text-sidebar-foreground hover:bg-sidebar-muted"
              onClick={toggle}
            >
              <PanelLeftClose className="h-4 w-4" />
            </Button>
          )}
        </div>

        {/* Expand button when collapsed */}
        {collapsed && (
          <div className="flex justify-center py-2">
            <Button
              variant="ghost"
              size="icon"
              className="h-7 w-7 text-sidebar-foreground/60 hover:text-sidebar-foreground hover:bg-sidebar-muted"
              onClick={toggle}
            >
              <PanelLeft className="h-4 w-4" />
            </Button>
          </div>
        )}

        {/* Navigation */}
        <nav className="flex-1 overflow-y-auto py-3 px-2 space-y-5">
          {apps.map((app) =>
            canAccessApp(activeSession.roles, app) ? (
              <div key={app} className="space-y-1">
                {!collapsed && (
                  <p className="px-3 pb-1 text-[10px] font-semibold text-sidebar-accent uppercase tracking-[0.15em]">
                    {APP_SECTION_LABELS[app]}
                  </p>
                )}
                {collapsed && (
                  <Separator className="bg-sidebar-border mx-auto w-8 my-2" />
                )}
                <div className="space-y-0.5">
                  {routes
                    .filter((route) => route.app === app)
                    .map((route) => {
                      const Icon = route.icon
                      const linkContent = (
                        <NavLink
                          key={route.path}
                          to={route.path}
                          className={({ isActive }) =>
                            cn(
                              'flex items-center gap-3 rounded-md text-sm transition-all duration-150 relative group',
                              collapsed ? 'justify-center px-2 py-2.5' : 'px-3 py-2',
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
                              {Icon && <Icon className={cn('flex-shrink-0', collapsed ? 'w-5 h-5' : 'w-4 h-4')} />}
                              {!collapsed && <span className="truncate">{route.label}</span>}
                            </>
                          )}
                        </NavLink>
                      )

                      if (collapsed) {
                        return (
                          <Tooltip key={route.path}>
                            <TooltipTrigger asChild>{linkContent}</TooltipTrigger>
                            <TooltipContent side="right" className="font-medium">
                              {route.label}
                            </TooltipContent>
                          </Tooltip>
                        )
                      }
                      return linkContent
                    })}
                </div>
              </div>
            ) : null
          )}
        </nav>

        {/* User Card */}
        <div className={cn('border-t border-sidebar-border', collapsed ? 'p-2' : 'p-3')}>
          {collapsed ? (
            <Tooltip>
              <TooltipTrigger asChild>
                <div className="flex justify-center py-1">
                  <Avatar className="h-8 w-8 cursor-default">
                    <AvatarFallback className="text-xs bg-sidebar-muted text-sidebar-accent">{initials}</AvatarFallback>
                  </Avatar>
                </div>
              </TooltipTrigger>
              <TooltipContent side="right">
                <p className="font-medium">{activeSession.displayName}</p>
                <p className="text-xs opacity-70">{activeSession.roles.join(', ') || 'no roles'}</p>
              </TooltipContent>
            </Tooltip>
          ) : (
            <div className="space-y-2">
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
          )}
        </div>
      </aside>
    </TooltipProvider>
  )
}
