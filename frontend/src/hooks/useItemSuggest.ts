import useSWR from 'swr'
import { fetchItemSuggest } from '../lib/additionalApi'

export function useItemSuggest(query: string) {
  return useSWR(
    query.length >= 2 ? ['item-suggest', query] : null,
    ([, q]) => fetchItemSuggest(q),
    { dedupingInterval: 300 },
  )
}
