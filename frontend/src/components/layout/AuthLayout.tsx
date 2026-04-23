import type { PropsWithChildren } from 'react'
import { Link } from 'react-router-dom'
import { Card } from '@/components/ui/card'
import { EnvironmentNotice } from '@/components/EnvironmentNotice'

type AuthLayoutProps = PropsWithChildren<{
  title?: string
  subtitle?: string
}>

export function AuthLayout({ title = 'Inventory Manager', subtitle, children }: AuthLayoutProps) {
  return (
    <div className="min-h-screen bg-gradient-to-br from-primary/10 via-background to-accent/10 flex items-center justify-center p-4">
      <div className="w-full max-w-md space-y-6">
        {/* Brand */}
        <div className="flex flex-col items-center space-y-2 text-center">
          <Link to="/" className="flex items-center gap-2">
            <div className="w-10 h-10 rounded-lg bg-primary/20 flex items-center justify-center text-lg font-bold text-primary">
              IM
            </div>
            <span className="text-2xl font-bold">{title}</span>
          </Link>
          {subtitle && <p className="text-sm text-muted-foreground">{subtitle}</p>}
        </div>

        {/* Content Card */}
        <Card className="p-6 sm:p-8">{children}</Card>

        {/* Footer */}
        <div className="text-center text-xs text-muted-foreground">
          <EnvironmentNotice />
        </div>
      </div>
    </div>
  )
}
