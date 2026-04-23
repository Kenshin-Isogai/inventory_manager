import { useState } from 'react'
import { useNavigate } from 'react-router-dom'
import { useSWRConfig } from 'swr'

import { SectionCard } from '../components/SectionCard'
import { LOCAL_LOGIN_PROFILES, setStoredToken } from '../lib/auth'
import { isFirebaseAuthConfigured, signInWithIdentityPlatform, signUpWithIdentityPlatform, sendVerificationEmail } from '../lib/firebaseAuth'

export function LoginPage() {
  const navigate = useNavigate()
  const { mutate } = useSWRConfig()
  const [manualToken, setManualToken] = useState('')
  const [email, setEmail] = useState('')
  const [password, setPassword] = useState('')
  const [error, setError] = useState('')

  async function handleLogin(token: string) {
    setStoredToken(token)
    await mutate('auth-session')
    navigate('/operator/dashboard', { replace: true })
  }

  return (
    <div className="auth-page">
      <SectionCard title="Login" subtitle="Identity Platform email/password is the primary flow. Local tokens stay available only for development.">
        {isFirebaseAuthConfigured() ? (
          <form
            className="stack-form"
            onSubmit={async (event) => {
              event.preventDefault()
              setError('')
              try {
                await signInWithIdentityPlatform(email, password)
                await mutate('auth-session')
                navigate('/operator/dashboard', { replace: true })
              } catch (caught) {
                setError(caught instanceof Error ? caught.message : 'Failed to sign in')
              }
            }}
          >
            <label>
              <span>Email</span>
              <input value={email} onChange={(event) => setEmail(event.target.value)} />
            </label>
            <label>
              <span>Password</span>
              <input type="password" value={password} onChange={(event) => setPassword(event.target.value)} />
            </label>
            <div className="button-row">
              <button type="submit" className="primary-button">
                Sign In
              </button>
              <button
                type="button"
                className="secondary-button"
                onClick={async () => {
                  setError('')
                  try {
                    await signUpWithIdentityPlatform(email, password)
                    await sendVerificationEmail()
                    await mutate('auth-session')
                    navigate('/auth/verify-email', { replace: true })
                  } catch (caught) {
                    setError(caught instanceof Error ? caught.message : 'Failed to create account')
                  }
                }}
              >
                Create Account
              </button>
            </div>
            {error ? <p className="muted-copy">{error}</p> : null}
          </form>
        ) : null}
        {import.meta.env.DEV ? (
          <>
            <div className="auth-grid">
              {LOCAL_LOGIN_PROFILES.map((profile) => (
                <button key={profile.token} type="button" className="primary-button" onClick={() => void handleLogin(profile.token)}>
                  {profile.label}: {profile.description}
                </button>
              ))}
            </div>
            <form
              className="stack-form"
              onSubmit={(event) => {
                event.preventDefault()
                void handleLogin(manualToken)
              }}
            >
              <label>
                <span>Manual token</span>
                <input
                  value={manualToken}
                  onChange={(event) => setManualToken(event.target.value)}
                  placeholder="local:new.user@example.local|New User"
                />
              </label>
              <button type="submit" className="secondary-button">
                Continue with token
              </button>
            </form>
          </>
        ) : null}
      </SectionCard>
    </div>
  )
}
