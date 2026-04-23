import useSWR from 'swr'

import { fetchProcurementBudgetCategories } from '../lib/mockApi'

export function useProcurementBudgetCategories(projectId?: string) {
  return useSWR(['procurement-budget-categories', projectId ?? ''], ([, currentProjectId]) =>
    fetchProcurementBudgetCategories(currentProjectId),
  )
}
