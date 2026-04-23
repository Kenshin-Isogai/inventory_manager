import useSWR from 'swr'

import { fetchMasterData } from '../lib/mockApi'

export function useMasterData() {
  return useSWR('master-data', fetchMasterData)
}
