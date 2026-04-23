import useSWR from 'swr'

import { fetchCurrentSession } from '../lib/mockApi'

export function useAuthSession() {
  return useSWR('auth-session', fetchCurrentSession)
}
