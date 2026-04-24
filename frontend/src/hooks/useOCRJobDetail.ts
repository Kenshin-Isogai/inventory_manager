import useSWR from 'swr'

import { fetchOCRJobDetail } from '../lib/mockApi'

export function useOCRJobDetail(id?: string, enabled = true) {
  return useSWR(enabled && id ? ['ocr-job-detail', id] : null, ([, currentID]) => fetchOCRJobDetail(currentID))
}
