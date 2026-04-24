import useSWR from 'swr'
import { fetchEnhancedShortages } from '../lib/additionalApi'

export function useEnhancedShortages(device?: string, scope?: string, coverageRule?: string) {
  return useSWR(
    ['enhanced-shortages', device ?? '', scope ?? '', coverageRule ?? ''],
    ([, d, s, cr]) => fetchEnhancedShortages(d || undefined, s || undefined, cr || undefined),
  )
}
