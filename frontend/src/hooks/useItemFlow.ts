import useSWR from 'swr'
import { fetchItemFlow } from '../lib/additionalApi'

export function useItemFlow(itemId?: string) {
  return useSWR(itemId ? ['item-flow', itemId] : null, ([, id]) => fetchItemFlow(id))
}
