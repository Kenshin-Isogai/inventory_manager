import { useState } from 'react'
import { useSWRConfig } from 'swr'

import { SectionCard } from '../components/SectionCard'
import { sendVerificationEmail, refreshFirebaseUser } from '../lib/firebaseAuth'

export function VerifyEmailPage() {
  const { mutate } = useSWRConfig()
  const [message, setMessage] = useState('')

  return (
    <div className="auth-page">
      <SectionCard title="Verify Email" subtitle="Identity Platform sign-in succeeded, but backend access stays blocked until the email is verified.">
        <div className="button-row">
          <button
            type="button"
            className="primary-button"
            onClick={async () => {
              await sendVerificationEmail()
              setMessage('Verification email sent.')
            }}
          >
            Send Verification Email
          </button>
          <button
            type="button"
            className="secondary-button"
            onClick={async () => {
              await refreshFirebaseUser()
              await mutate('auth-session')
              setMessage('Verification status refreshed.')
            }}
          >
            I Verified My Email
          </button>
        </div>
        {message ? <p className="muted-copy">{message}</p> : null}
      </SectionCard>
    </div>
  )
}
