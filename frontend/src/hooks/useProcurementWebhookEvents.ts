import useSWR from 'swr'

import { fetchProcurementWebhookEvents } from '../lib/mockApi'

export function useProcurementWebhookEvents() {
  return useSWR('procurement-webhook-events', fetchProcurementWebhookEvents)
}
