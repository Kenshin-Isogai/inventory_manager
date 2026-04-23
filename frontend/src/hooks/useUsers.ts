import useSWR from 'swr'

import { fetchUsers } from '../lib/mockApi'

export function useUsers() {
  return useSWR('users', fetchUsers)
}
