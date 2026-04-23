import { useInventoryOverview } from '../hooks/useInventoryOverview'
import { SectionCard } from '../components/SectionCard'

export function InventoryPage() {
  const { data } = useInventoryOverview()

  return (
    <SectionCard title="Inventory" subtitle="Current local balances with reserved and available quantities.">
      <table className="data-table">
        <thead>
          <tr>
            <th>Item</th>
            <th>Description</th>
            <th>Manufacturer</th>
            <th>Category</th>
            <th>Location</th>
            <th>On hand</th>
            <th>Reserved</th>
            <th>Available</th>
          </tr>
        </thead>
        <tbody>
          {data?.balances.map((row) => (
            <tr key={`${row.itemId}-${row.locationCode}`}>
              <td>{row.itemNumber}</td>
              <td>{row.description}</td>
              <td>{row.manufacturer}</td>
              <td>{row.category}</td>
              <td>{row.locationCode}</td>
              <td>{row.onHandQuantity}</td>
              <td>{row.reservedQuantity}</td>
              <td>{row.availableQuantity}</td>
            </tr>
          ))}
        </tbody>
      </table>
    </SectionCard>
  )
}
