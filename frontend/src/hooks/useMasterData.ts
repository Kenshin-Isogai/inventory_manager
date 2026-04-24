import useSWR from 'swr'

import { fetchMasterData } from '../lib/mockApi'

export function useMasterData(enabled = true) {
  return useSWR(enabled ? 'master-data' : null, fetchMasterData)
}
