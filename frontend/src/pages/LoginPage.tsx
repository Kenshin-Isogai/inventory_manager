import { useState } from 'react'
import { useNavigate } from 'react-router-dom'
import { useSWRConfig } from 'swr'

import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import { Separator } from '@/components/ui/separator'
import { LOCAL_LOGIN_PROFILES, setStoredToken } from '@/lib/auth'
import { fetchCurrentSession } from '@/lib/mockApi'
import type { AuthSessionResponse } from '@/types'
import {
  describeIdentityAuthError,
  isFirebaseAuthConfigured,
  sendVerificationEmail,
  signInWithIdentityPlatform,
  signUpWithIdentityPlatform,
  syncStoredTokenFromIdentityPlatform,
} from '@/lib/firebaseAuth'

export function LoginPage() {
  const navigate = useNavigate()
  const { mutate } = useSWRConfig()
  const [manualToken, setManualToken] = useState('')
  const [email, setEmail] = useState('')
  const [password, setPassword] = useState('')
  const [error, setError] = useState('')

  async function handleLogin(token: string) {
    setStoredToken(token)
    const session = await fetchCurrentSession()
    await mutate('auth-session', session, false)
    navigate('/app/portal', { replace: true })
  }

  async function syncSessionAndNavigate() {
    await syncStoredTokenFromIdentityPlatform()
    const session = await fetchCurrentSession() as AuthSessionResponse | undefined
    await mutate('auth-session', session, false)

    if (!session?.authenticated) {
      throw new Error('Identity Platform sign-in succeeded, but the backend rejected the token.')
    }
    if (!session.user.emailVerified) {
      navigate('/auth/verify-email', { replace: true })
      return
    }
    if (session.user.status === 'pending') {
      navigate('/auth/pending', { replace: true })
      return
    }
    if (session.user.status === 'rejected') {
      navigate('/auth/rejected', { replace: true })
      return
    }
    if (session.user.status === 'unregistered' || session.user.registrationNeeded) {
      navigate('/auth/register', { replace: true })
      return
    }
    navigate('/app/portal', { replace: true })
  }

  return (
    <div className="min-h-screen flex items-center justify-center bg-gradient-to-br from-primary/5 via-background to-accent/5 p-4">
      <Card className="w-full max-w-md">
        <CardHeader className="space-y-2">
          <CardTitle className="text-2xl">Sign In</CardTitle>
          <CardDescription>
            Identity Platform email/password is the primary flow. Local tokens stay available only for development.
          </CardDescription>
        </CardHeader>
        <CardContent className="space-y-6">
          {isFirebaseAuthConfigured() ? (
            <form
              onSubmit={async (event) => {
                event.preventDefault()
                setError('')
                try {
                  await signInWithIdentityPlatform(email, password)
                  await syncSessionAndNavigate()
                } catch (caught) {
                  setError(
                    describeIdentityAuthError(
                      caught,
                      'Failed to sign in. If the page jumps back here, backend auth may still be rejecting the token.'
                    )
                  )
                }
              }}
              className="space-y-4"
            >
              <div className="space-y-2">
                <Label htmlFor="email">Email</Label>
                <Input
                  id="email"
                  type="email"
                  value={email}
                  onChange={(event) => setEmail(event.target.value)}
                  placeholder="your@email.com"
                />
              </div>
              <div className="space-y-2">
                <Label htmlFor="password">Password</Label>
                <Input
                  id="password"
                  type="password"
                  value={password}
                  onChange={(event) => setPassword(event.target.value)}
                  placeholder="••••••••"
                />
              </div>
              <div className="flex gap-2">
                <Button type="submit" className="flex-1">
                  Sign In
                </Button>
                <Button
                  type="button"
                  variant="outline"
                  className="flex-1"
                  onClick={async () => {
                    setError('')
                    try {
                      await signUpWithIdentityPlatform(email, password)
                      await sendVerificationEmail()
                      await syncStoredTokenFromIdentityPlatform()
                      const session = await fetchCurrentSession()
                      await mutate('auth-session', session, false)
                      navigate('/auth/verify-email', { replace: true })
                    } catch (caught) {
                      setError(describeIdentityAuthError(caught, 'Failed to create account'))
                    }
                  }}
                >
                  Create Account
                </Button>
              </div>
              <p className="text-xs text-muted-foreground text-center">
                If this email already exists in Identity Platform, use `Sign In`. Verified identities will continue to app registration automatically.
              </p>
              {error && <p className="text-sm text-destructive text-center">{error}</p>}
            </form>
          ) : null}

          {import.meta.env.DEV ? (
            <>
              <Separator />
              <div className="space-y-3">
                <p className="text-sm font-medium text-muted-foreground">Development: Local Login Profiles</p>
                <div className="grid gap-2">
                  {LOCAL_LOGIN_PROFILES.map((profile) => (
                    <Button
                      key={profile.token}
                      type="button"
                      variant="outline"
                      className="justify-start text-left h-auto py-3 px-3"
                      onClick={() => void handleLogin(profile.token)}
                    >
                      <div>
                        <p className="font-medium text-sm">{profile.label}</p>
                        <p className="text-xs text-muted-foreground">{profile.description}</p>
                      </div>
                    </Button>
                  ))}
                </div>
              </div>

              <Separator />
              <form
                onSubmit={(event) => {
                  event.preventDefault()
                  void handleLogin(manualToken)
                }}
                className="space-y-3"
              >
                <p className="text-sm font-medium text-muted-foreground">Manual Token Input</p>
                <div className="space-y-2">
                  <Label htmlFor="manual-token">Token</Label>
                  <Input
                    id="manual-token"
                    value={manualToken}
                    onChange={(event) => setManualToken(event.target.value)}
                    placeholder="local:new.user@example.local|New User"
                    className="text-xs"
                  />
                </div>
                <Button type="submit" variant="outline" className="w-full">
                  Continue with Token
                </Button>
              </form>
            </>
          ) : null}
        </CardContent>
      </Card>
    </div>
  )
}
