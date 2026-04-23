import { useSWRConfig } from 'swr'

import { SectionCard } from '../components/SectionCard'
import { useProcurementRequests } from '../hooks/useProcurementRequests'
import { sendMockProcurementWebhook } from '../lib/mockApi'

export function InspectorArrivalsPage() {
  const { data } = useProcurementRequests()
  const { mutate } = useSWRConfig()
  const arrivalRows = data?.rows.filter((row) => ['submitted', 'ordered', 'partially_received'].includes(row.normalizedStatus)) ?? []

  return (
    <SectionCard title="Arrivals / Inspection" subtitle="Receiving inspectors can confirm inbound requests without full procurement access.">
      <table className="data-table">
        <thead>
          <tr>
            <th>Batch</th>
            <th>Supplier</th>
            <th>Status</th>
            <th>Items</th>
            <th>Action</th>
          </tr>
        </thead>
        <tbody>
          {arrivalRows.map((row) => (
            <tr key={row.id}>
              <td>{row.batchNumber}</td>
              <td>{row.supplierName}</td>
              <td>{row.normalizedStatus}</td>
              <td>{row.requestedItems}</td>
              <td>
                <button
                  type="button"
                  className="primary-button"
                  onClick={async () => {
                    await sendMockProcurementWebhook({
                      eventType: 'procurement.status_changed',
                      requestId: row.id,
                      normalizedStatus: 'received',
                      rawStatus: 'external_receipt_completed',
                    })
                    await Promise.all([
                      mutate('procurement-requests'),
                      mutate(['procurement-request-detail', row.id]),
                      mutate('procurement-webhook-events'),
                    ])
                  }}
                >
                  Mark Received
                </button>
              </td>
            </tr>
          ))}
        </tbody>
      </table>
    </SectionCard>
  )
}
