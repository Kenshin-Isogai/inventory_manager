import { useState } from 'react'
import type { FormEvent } from 'react'
import { useSWRConfig } from 'swr'

import { SectionCard } from '../components/SectionCard'
import { useInventoryOverview } from '../hooks/useInventoryOverview'
import { adjustInventory } from '../lib/mockApi'

export function InventoryEventsPage() {
  const { data } = useInventoryOverview()
  const { mutate } = useSWRConfig()
  const [itemId, setItemId] = useState('item-er2')
  const [locationCode, setLocationCode] = useState('TOKYO-A1')
  const [quantityDelta, setQuantityDelta] = useState(-1)
  const [deviceScopeId, setDeviceScopeId] = useState('ds-er2-powerboard')
  const [note, setNote] = useState('Inventory adjustment')
  const [message, setMessage] = useState('')

  async function handleSubmit(event: FormEvent<HTMLFormElement>) {
    event.preventDefault()
    await adjustInventory({ itemId, locationCode, quantityDelta, deviceScopeId, note })
    await mutate('inventory-overview')
    setMessage(`Recorded adjustment ${quantityDelta} for ${itemId} at ${locationCode}.`)
  }

  return (
    <div className="page-grid">
      <SectionCard title="Inventory Events" subtitle="Receive, move, and adjust operations currently flow through the local adjustment command.">
        <form className="stack-form" onSubmit={handleSubmit}>
          <label>
            <span>Item</span>
            <select value={itemId} onChange={(event) => setItemId(event.target.value)}>
              {data?.balances.map((row) => (
                <option key={row.itemId} value={row.itemId}>
                  {row.itemNumber} / {row.description}
                </option>
              ))}
            </select>
          </label>
          <div className="two-column">
            <label>
              <span>Location</span>
              <input value={locationCode} onChange={(event) => setLocationCode(event.target.value)} />
            </label>
            <label>
              <span>Quantity Delta</span>
              <input type="number" value={quantityDelta} onChange={(event) => setQuantityDelta(Number(event.target.value) || 0)} />
            </label>
          </div>
          <div className="two-column">
            <label>
              <span>Device Scope</span>
              <input value={deviceScopeId} onChange={(event) => setDeviceScopeId(event.target.value)} />
            </label>
            <label>
              <span>Note</span>
              <input value={note} onChange={(event) => setNote(event.target.value)} />
            </label>
          </div>
          <button type="submit" className="primary-button">
            Record Inventory Event
          </button>
          {message ? <p className="muted-copy">{message}</p> : null}
        </form>
      </SectionCard>
    </div>
  )
}
