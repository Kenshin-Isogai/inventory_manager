import useSWR from 'swr'

import { fetchReservations } from '../lib/mockApi'

export function useReservations(device?: string, scope?: string) {
  return useSWR(['reservations', device ?? '', scope ?? ''], ([, currentDevice, currentScope]) =>
    fetchReservations(currentDevice, currentScope),
  )
}
