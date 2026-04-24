import useSWR from 'swr'

import { fetchScopeSystems } from '../lib/mockApi'

export function useScopeSystems() {
  return useSWR('scope-systems', fetchScopeSystems)
}
