import useSWR from 'swr'

import { fetchProcurementBudgetCategories } from '../lib/mockApi'

export function useProcurementBudgetCategories(projectId?: string, enabled = true) {
  return useSWR(enabled ? ['procurement-budget-categories', projectId ?? ''] : null, ([, currentProjectId]) =>
    fetchProcurementBudgetCategories(currentProjectId),
  )
}
