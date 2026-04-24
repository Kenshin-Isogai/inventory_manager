import useSWR from 'swr'
import { fetchScopeOverview } from '../lib/additionalApi'

export function useScopeOverview(device?: string) {
  return useSWR(['scope-overview', device ?? ''], ([, d]) => fetchScopeOverview(d || undefined))
}
