import useSWR from 'swr'

import { fetchShortages } from '../lib/mockApi'

export function useShortages(device?: string, scope?: string) {
  return useSWR(['shortages', device ?? '', scope ?? ''], ([, currentDevice, currentScope]) =>
    fetchShortages(currentDevice, currentScope),
  )
}
