import { useNavigate } from 'react-router-dom'

import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card'
import { Button } from '@/components/ui/button'
import { useAuthSession } from '@/hooks/useAuthSession'

export function RejectedPage() {
  const navigate = useNavigate()
  const { data: session } = useAuthSession()

  return (
    <div className="min-h-screen flex items-center justify-center bg-gradient-to-br from-primary/5 via-background to-accent/5 p-4">
      <Card className="w-full max-w-md">
        <CardHeader className="space-y-2">
          <CardTitle className="text-2xl">Rejected</CardTitle>
          <CardDescription>
            The current app user exists but does not have an approved role assignment.
          </CardDescription>
        </CardHeader>
        <CardContent className="space-y-4">
          <p className="text-sm text-muted-foreground">
            {session?.user.rejectionReason || 'No rejection reason was recorded.'}
          </p>
          <Button
            onClick={() => navigate('/auth/register')}
            className="w-full"
          >
            Re-submit Registration
          </Button>
        </CardContent>
      </Card>
    </div>
  )
}
