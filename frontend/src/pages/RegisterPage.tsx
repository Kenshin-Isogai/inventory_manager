import { useEffect, useState } from 'react'
import { useNavigate } from 'react-router-dom'
import { useSWRConfig } from 'swr'

import { SectionCard } from '../components/SectionCard'
import { useAuthSession } from '../hooks/useAuthSession'
import { registerUser } from '../lib/mockApi'

export function RegisterPage() {
  const navigate = useNavigate()
  const { data: session } = useAuthSession()
  const { mutate } = useSWRConfig()
  const [email, setEmail] = useState('')
  const [displayName, setDisplayName] = useState('')
  const [error, setError] = useState('')

  useEffect(() => {
    setEmail(session?.user.email ?? '')
    setDisplayName(session?.user.displayName ?? '')
  }, [session?.user.displayName, session?.user.email])

  return (
    <div className="auth-page">
      <SectionCard title="Registration" subtitle="Create or re-submit an app user record after identity is established.">
        <form
          className="stack-form"
          onSubmit={async (event) => {
            event.preventDefault()
            try {
              await registerUser({ email, displayName })
              await mutate('auth-session')
              await mutate('users')
              navigate('/auth/pending', { replace: true })
            } catch (caught) {
              setError(caught instanceof Error ? caught.message : 'Failed to submit registration')
            }
          }}
        >
          <label>
            <span>Email</span>
            <input value={email} onChange={(event) => setEmail(event.target.value)} />
          </label>
          <label>
            <span>Display Name</span>
            <input value={displayName} onChange={(event) => setDisplayName(event.target.value)} />
          </label>
          <button type="submit" className="primary-button">
            Submit Registration
          </button>
          {error ? <p className="muted-copy">{error}</p> : null}
        </form>
      </SectionCard>
    </div>
  )
}
