import useSWR from 'swr'

import { fetchDashboard } from '../lib/mockApi'

export function useDashboard() {
  return useSWR('dashboard', fetchDashboard)
}
