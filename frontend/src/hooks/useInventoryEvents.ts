import useSWR from 'swr'

import { fetchInventoryEvents } from '../lib/mockApi'

export function useInventoryEvents() {
  return useSWR('inventory-events', fetchInventoryEvents)
}
