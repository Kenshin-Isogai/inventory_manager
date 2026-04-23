import useSWR from 'swr'

import { fetchOCRJobs } from '../lib/mockApi'

export function useOCRJobs() {
  return useSWR('ocr-jobs', fetchOCRJobs)
}
