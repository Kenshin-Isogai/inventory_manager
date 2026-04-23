import useSWR from 'swr'

import { fetchReservations } from '../lib/mockApi'

export function useReservations() {
  return useSWR('reservations', fetchReservations)
}
