import useSWR from 'swr'

import { fetchInventoryItems } from '../lib/mockApi'

export function useInventoryItems() {
  return useSWR('inventory-items', fetchInventoryItems)
}
