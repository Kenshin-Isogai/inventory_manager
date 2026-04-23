import useSWR from 'swr'

import { fetchDeviceScopes } from '../lib/mockApi'

export function useDeviceScopes() {
  return useSWR('device-scopes', fetchDeviceScopes)
}
