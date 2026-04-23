import useSWR from 'swr'

import { fetchProcurementSyncRuns } from '../lib/mockApi'

export function useProcurementSyncRuns() {
  return useSWR('procurement-sync-runs', fetchProcurementSyncRuns)
}
