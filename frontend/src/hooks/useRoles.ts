import useSWR from 'swr'

import { fetchRoles } from '../lib/mockApi'

export function useRoles() {
  return useSWR('roles', fetchRoles)
}
