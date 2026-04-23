import useSWR from 'swr'

import { fetchProcurementRequestDetail } from '../lib/mockApi'

export function useProcurementRequestDetail(id?: string) {
  return useSWR(id ? ['procurement-request-detail', id] : null, ([, currentId]) =>
    fetchProcurementRequestDetail(currentId),
  )
}
