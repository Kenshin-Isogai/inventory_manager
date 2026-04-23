import useSWR from 'swr'

import { fetchOCRJobDetail } from '../lib/mockApi'

export function useOCRJobDetail(id?: string) {
  return useSWR(id ? ['ocr-job-detail', id] : null, ([, currentID]) => fetchOCRJobDetail(currentID))
}
