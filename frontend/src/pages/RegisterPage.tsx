import { useState } from 'react'
import { useNavigate } from 'react-router-dom'
import { useSWRConfig } from 'swr'

import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import { useAuthSession } from '@/hooks/useAuthSession'
import { defaultPathForApp } from '@/lib/auth'
import { registerUser } from '@/lib/mockApi'

export function RegisterPage() {
  const navigate = useNavigate()
  const { data: session } = useAuthSession()
  const { mutate } = useSWRConfig()
  const [email, setEmail] = useState<string | null>(null)
  const [displayName, setDisplayName] = useState<string | null>(null)
  const [error, setError] = useState('')
  const resolvedEmail = email ?? session?.user.email ?? ''
  const resolvedDisplayName = displayName ?? session?.user.displayName ?? ''

  return (
    <div className="min-h-screen flex items-center justify-center bg-gradient-to-br from-primary/5 via-background to-accent/5 p-4">
      <Card className="w-full max-w-md">
        <CardHeader className="space-y-2">
          <CardTitle className="text-2xl">Registration</CardTitle>
          <CardDescription>
            Create or re-submit an app user record after identity is established.
          </CardDescription>
        </CardHeader>
        <CardContent>
          <form
            onSubmit={async (event) => {
              event.preventDefault()
              setError('')
              try {
                const result = await registerUser({ email: resolvedEmail, displayName: resolvedDisplayName })
                await mutate('auth-session')
                await mutate('users')
                navigate(result.status === 'active' ? defaultPathForApp('admin') : '/auth/pending', { replace: true })
              } catch (caught) {
                setError(caught instanceof Error ? caught.message : 'Failed to submit registration')
              }
            }}
            className="space-y-4"
          >
            <div className="space-y-2">
              <Label htmlFor="email">Email</Label>
              <Input
                id="email"
                type="email"
                value={resolvedEmail}
                onChange={(event) => setEmail(event.target.value)}
                placeholder="your@email.com"
              />
            </div>
            <div className="space-y-2">
              <Label htmlFor="display-name">Display Name</Label>
              <Input
                id="display-name"
                value={resolvedDisplayName}
                onChange={(event) => setDisplayName(event.target.value)}
                placeholder="Your Name"
              />
            </div>
            <Button type="submit" className="w-full">
              Submit Registration
            </Button>
            {error && <p className="text-sm text-destructive text-center">{error}</p>}
          </form>
        </CardContent>
      </Card>
    </div>
  )
}
