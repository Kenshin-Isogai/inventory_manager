import useSWR from 'swr'

import { fetchProcurementRequests } from '../lib/mockApi'

export function useProcurementRequests() {
  return useSWR('procurement-requests', fetchProcurementRequests)
}
