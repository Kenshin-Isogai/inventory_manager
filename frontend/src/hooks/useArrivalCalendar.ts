import useSWR from 'swr'
import { fetchArrivalCalendar } from '../lib/additionalApi'

export function useArrivalCalendar(yearMonth: string) {
  return useSWR(yearMonth ? ['arrival-calendar', yearMonth] : null, ([, ym]) =>
    fetchArrivalCalendar(ym),
  )
}
