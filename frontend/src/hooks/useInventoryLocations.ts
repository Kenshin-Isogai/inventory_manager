import useSWR from 'swr'

import { fetchInventoryLocations } from '../lib/mockApi'

export function useInventoryLocations() {
  return useSWR('inventory-locations', fetchInventoryLocations)
}
