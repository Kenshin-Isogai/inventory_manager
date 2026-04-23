import useSWR from 'swr'

import { fetchProcurementProjects } from '../lib/mockApi'

export function useProcurementProjects() {
  return useSWR('procurement-projects', fetchProcurementProjects)
}
