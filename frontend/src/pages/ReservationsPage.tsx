import { SectionCard } from '../components/SectionCard'
import { useReservations } from '../hooks/useReservations'

export function ReservationsPage() {
  const { data } = useReservations()

  return (
    <SectionCard title="Reservations" subtitle="Local read model contract for reservation visibility.">
      <table className="data-table">
        <thead>
          <tr>
            <th>ID</th>
            <th>Item</th>
            <th>Description</th>
            <th>Qty</th>
            <th>Device</th>
            <th>Scope</th>
            <th>Status</th>
          </tr>
        </thead>
        <tbody>
          {data?.rows.map((row) => (
            <tr key={row.id}>
              <td>{row.id}</td>
              <td>{row.itemNumber}</td>
              <td>{row.description}</td>
              <td>{row.quantity}</td>
              <td>{row.device}</td>
              <td>{row.scope}</td>
              <td>{row.status}</td>
            </tr>
          ))}
        </tbody>
      </table>
    </SectionCard>
  )
}
