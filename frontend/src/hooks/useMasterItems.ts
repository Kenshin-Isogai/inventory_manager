import useSWR from 'swr'

import { fetchMasterItems } from '../lib/additionalApi'

export function useMasterItems(enabled = true) {
  return useSWR(enabled ? 'master-items' : null, fetchMasterItems)
}
