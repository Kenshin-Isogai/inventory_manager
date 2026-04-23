import useSWR from 'swr'

import { fetchDevices } from '../lib/mockApi'

export function useDevices() {
  return useSWR('devices', fetchDevices)
}
