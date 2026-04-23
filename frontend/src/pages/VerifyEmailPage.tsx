import { useState } from 'react'
import { useSWRConfig } from 'swr'

import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card'
import { Button } from '@/components/ui/button'
import { sendVerificationEmail, refreshFirebaseUser } from '@/lib/firebaseAuth'

export function VerifyEmailPage() {
  const { mutate } = useSWRConfig()
  const [message, setMessage] = useState('')

  return (
    <div className="min-h-screen flex items-center justify-center bg-gradient-to-br from-primary/5 via-background to-accent/5 p-4">
      <Card className="w-full max-w-md">
        <CardHeader className="space-y-2">
          <CardTitle className="text-2xl">Verify Email</CardTitle>
          <CardDescription>
            Identity Platform sign-in succeeded, but backend access stays blocked until the email is verified.
          </CardDescription>
        </CardHeader>
        <CardContent className="space-y-4">
          <div className="flex flex-col gap-3 sm:flex-row">
            <Button
              onClick={async () => {
                await sendVerificationEmail()
                setMessage('Verification email sent.')
              }}
              className="flex-1"
            >
              Send Verification Email
            </Button>
            <Button
              variant="outline"
              onClick={async () => {
                await refreshFirebaseUser()
                await mutate('auth-session')
                setMessage('Verification status refreshed.')
              }}
              className="flex-1"
            >
              I Verified My Email
            </Button>
          </div>
          {message && <p className="text-sm text-muted-foreground text-center">{message}</p>}
        </CardContent>
      </Card>
    </div>
  )
}
