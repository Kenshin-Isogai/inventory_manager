import useSWR from 'swr'

import { fetchImports } from '../lib/mockApi'

export function useImports() {
  return useSWR('imports', fetchImports)
}
