import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card'

export function PendingPage() {
  return (
    <div className="min-h-screen flex items-center justify-center bg-gradient-to-br from-primary/5 via-background to-accent/5 p-4">
      <Card className="w-full max-w-md">
        <CardHeader className="space-y-2">
          <CardTitle className="text-2xl">Pending Approval</CardTitle>
          <CardDescription>
            The identity is known, but app access is still waiting for admin approval.
          </CardDescription>
        </CardHeader>
        <CardContent>
          <p className="text-sm text-muted-foreground">
            An admin needs to assign app roles before this account can enter the operator, inventory, procurement, or admin areas.
          </p>
        </CardContent>
      </Card>
    </div>
  )
}
