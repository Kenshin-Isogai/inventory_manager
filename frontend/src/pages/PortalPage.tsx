import { useNavigate } from 'react-router-dom'
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card'
import { Button } from '@/components/ui/button'
import { useAuthSession } from '@/hooks/useAuthSession'
import { useBootstrap } from '@/hooks/useBootstrap'
import { canAccessApp, defaultPathForApp, localFallbackSession, resolveSessionUser } from '@/lib/auth'
import { isFirebaseAuthConfigured } from '@/lib/firebaseAuth'
import { EnvironmentNotice } from '@/components/EnvironmentNotice'
import { ClipboardList, Package, FileText, Truck, Users } from 'lucide-react'
import type { AppSection } from '@/types'

const appSections: Array<{
  app: AppSection
  title: string
  description: string
  icon: React.ComponentType<{ className?: string }>
}> = [
  {
    app: 'operator',
    title: 'Operator',
    description: 'Manage requirements, reservations, shortages, and imports',
    icon: ClipboardList,
  },
  {
    app: 'inventory',
    title: 'Inventory',
    description: 'Track items, locations, and inventory events',
    icon: Package,
  },
  {
    app: 'procurement',
    title: 'Procurement',
    description: 'Create requests, process OCR, and manage drafts',
    icon: FileText,
  },
  {
    app: 'inspector',
    title: 'Inspector',
    description: 'Confirm arrivals and inspect received items',
    icon: Truck,
  },
  {
    app: 'admin',
    title: 'Admin',
    description: 'Manage users, roles, and master data',
    icon: Users,
  },
]

export function PortalPage() {
  const navigate = useNavigate()
  const { data: bootstrap } = useBootstrap()
  const { data: session } = useAuthSession()
  const authMode = bootstrap?.authMode ?? (isFirebaseAuthConfigured() ? 'enforced' : 'none')
  const activeSession = authMode === 'none' ? localFallbackSession : resolveSessionUser(session)

  const handleNavigate = (app: AppSection) => {
    const path = defaultPathForApp(app)
    navigate(path)
  }

  return (
    <div className="min-h-screen bg-gradient-to-br from-primary/5 via-background to-accent/5 py-12 px-4 sm:px-6 lg:px-8">
      <div className="max-w-6xl mx-auto space-y-8">
        <div className="text-center space-y-3">
          <h1 className="text-4xl font-bold tracking-tight">Inventory Management</h1>
          <p className="text-lg text-muted-foreground">
            Welcome, {activeSession.displayName}
          </p>
        </div>

        <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-4 sm:gap-6">
          {appSections.map((section) => {
            const hasAccess = canAccessApp(activeSession.roles, section.app)
            const Icon = section.icon

            return (
              <Card
                key={section.app}
                className={`transition-all ${
                  hasAccess
                    ? 'hover:shadow-lg hover:border-primary/50 cursor-pointer'
                    : 'opacity-50 pointer-events-none'
                }`}
              >
                <CardHeader>
                  <div className="space-y-2">
                    <CardTitle className="flex items-center gap-2">
                      <Icon className="w-5 h-5" />
                      {section.title}
                    </CardTitle>
                    <CardDescription>{section.description}</CardDescription>
                  </div>
                </CardHeader>
                <CardContent>
                  {hasAccess ? (
                    <Button
                      onClick={() => handleNavigate(section.app)}
                      className="w-full"
                    >
                      Access {section.title}
                    </Button>
                  ) : (
                    <Button disabled className="w-full">
                      No Access
                    </Button>
                  )}
                </CardContent>
              </Card>
            )
          })}
        </div>

        <div className="text-center text-sm text-muted-foreground space-y-2 pt-8 border-t">
          <EnvironmentNotice />
          <p className="text-xs">
            Current roles: {activeSession.roles.length > 0 ? activeSession.roles.join(', ') : 'no app roles'}
          </p>
        </div>
      </div>
    </div>
  )
}
