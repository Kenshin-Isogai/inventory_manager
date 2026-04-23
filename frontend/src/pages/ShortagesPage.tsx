import { useSearchParams } from 'react-router-dom'
import { useState } from 'react'

import { SectionCard } from '../components/SectionCard'
import { useImports } from '../hooks/useImports'
import { useShortages } from '../hooks/useShortages'
import { exportShortagesCSV } from '../lib/mockApi'

export function ShortagesPage() {
  const [searchParams] = useSearchParams()
  const device = searchParams.get('device') ?? ''
  const scope = searchParams.get('scope') ?? ''
  const { data: shortages } = useShortages(device, scope)
  const { data: imports } = useImports()
  const [exportMessage, setExportMessage] = useState('')

  async function handleExport() {
    const csv = await exportShortagesCSV(device, scope)
    const blob = new Blob([csv], { type: 'text/csv;charset=utf-8' })
    const url = URL.createObjectURL(blob)
    const link = document.createElement('a')
    link.href = url
    link.download = 'shortages.csv'
    link.click()
    URL.revokeObjectURL(url)
    setExportMessage('Exported shortage CSV.')
  }

  return (
    <div className="page-grid">
      <SectionCard title="Shortages" subtitle="Projection-based shortages filtered by the current Device / Scope context.">
        <div className="two-column">
          <button type="button" className="primary-button" onClick={() => void handleExport()}>
            Export CSV
          </button>
          <span>{exportMessage || 'Exports the currently filtered shortage projection.'}</span>
        </div>
        <table className="data-table">
          <thead>
            <tr>
              <th>Device</th>
              <th>Scope</th>
              <th>Manufacturer</th>
              <th>Item</th>
              <th>Description</th>
              <th>Short Qty</th>
            </tr>
          </thead>
          <tbody>
            {shortages?.rows.map((row) => (
              <tr key={`${row.device}-${row.scope}-${row.itemNumber}`}>
                <td>{row.device}</td>
                <td>{row.scope}</td>
                <td>{row.manufacturer}</td>
                <td>{row.itemNumber}</td>
                <td>{row.description}</td>
                <td>{row.quantity}</td>
              </tr>
            ))}
          </tbody>
        </table>
      </SectionCard>

      <SectionCard title="Import History" subtitle="CSV import jobs tracked locally for item and alias operations.">
        <table className="data-table">
          <thead>
            <tr>
              <th>ID</th>
              <th>Type</th>
              <th>Status</th>
              <th>File</th>
              <th>Summary</th>
            </tr>
          </thead>
          <tbody>
            {imports?.rows.map((row) => (
              <tr key={row.id}>
                <td>{row.id}</td>
                <td>{row.importType}</td>
                <td>{row.status}</td>
                <td>{row.fileName}</td>
                <td>{row.summary}</td>
              </tr>
            ))}
          </tbody>
        </table>
      </SectionCard>
    </div>
  )
}
