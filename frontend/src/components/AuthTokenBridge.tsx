import { useEffect } from 'react'
import { useSWRConfig } from 'swr'

import { clearStoredToken, setStoredToken } from '../lib/auth'
import { isFirebaseAuthConfigured, signOutFromIdentityPlatform, subscribeToIDTokenChange } from '../lib/firebaseAuth'

export function AuthTokenBridge() {
  const { mutate } = useSWRConfig()

  useEffect(() => {
    if (!isFirebaseAuthConfigured()) {
      return () => undefined
    }
    return subscribeToIDTokenChange(async (user) => {
      if (!user) {
        clearStoredToken()
        await mutate('auth-session')
        return
      }
      const token = await user.getIdToken()
      setStoredToken(token)
      await mutate('auth-session')
      if (!user.emailVerified) {
        return
      }
    })
  }, [mutate])

  useEffect(() => {
    const handler = () => {
      void signOutFromIdentityPlatform()
    }
    window.addEventListener('app-signout', handler)
    return () => window.removeEventListener('app-signout', handler)
  }, [])

  return null
}
