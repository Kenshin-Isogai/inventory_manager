import useSWR from 'swr'

import { fetchOCRJobs } from '../lib/mockApi'

export function useOCRJobs(createdBy?: string, enabled = true) {
  return useSWR(
    enabled ? (createdBy ? ['ocr-jobs', createdBy] : 'ocr-jobs') : null,
    () => fetchOCRJobs(createdBy),
  )
}
