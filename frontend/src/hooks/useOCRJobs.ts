import useSWR from 'swr'

import { fetchOCRJobs } from '../lib/mockApi'

export function useOCRJobs(createdBy?: string) {
  return useSWR(
    createdBy ? ['ocr-jobs', createdBy] : 'ocr-jobs',
    () => fetchOCRJobs(createdBy),
  )
}
