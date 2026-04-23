import useSWR from 'swr'

import { fetchBootstrap } from '../lib/mockApi'

export function useBootstrap() {
  return useSWR('bootstrap', fetchBootstrap)
}
