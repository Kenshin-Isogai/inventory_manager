import useSWR from 'swr'

import { fetchInventoryOverview } from '../lib/mockApi'

export function useInventoryOverview() {
  return useSWR('inventory-overview', fetchInventoryOverview)
}
