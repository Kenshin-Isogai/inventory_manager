import useSWR from 'swr'
import { fetchShortageTimeline } from '../lib/additionalApi'

export function useShortageTimeline(device?: string, scope?: string) {
  return useSWR(
    device && scope ? ['shortage-timeline', device, scope] : null,
    ([, d, s]) => fetchShortageTimeline(d, s),
  )
}
