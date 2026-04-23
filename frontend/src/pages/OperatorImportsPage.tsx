import type { ChangeEvent } from 'react'
import { useState } from 'react'
import { useSWRConfig } from 'swr'

import { SectionCard } from '../components/SectionCard'
import { useImports } from '../hooks/useImports'
import { importMasterDataCSV } from '../lib/mockApi'

type OperatorImportsPageProps = {
  mode: 'upload' | 'history'
}

export function OperatorImportsPage({ mode }: OperatorImportsPageProps) {
  const { data } = useImports()
  const { mutate } = useSWRConfig()
  const [message, setMessage] = useState('')

  async function handleUpload(event: ChangeEvent<HTMLInputElement>) {
    const file = event.target.files?.[0]
    if (!file) {
      return
    }
    await importMasterDataCSV('items', file)
    await mutate('imports')
    setMessage(`Staged import ${file.name}`)
    event.target.value = ''
  }

  return (
    <div className="page-grid">
      {mode === 'upload' ? (
        <SectionCard title="Imports Upload" subtitle="Operator-facing upload lane for local CSV rehearsal before cloud integration is fixed.">
          <label className="stack-form">
            <span>Upload CSV</span>
            <input type="file" accept=".csv,text/csv" onChange={(event) => void handleUpload(event)} />
          </label>
          <p className="muted-copy">
            Current local implementation reuses the import job pipeline so operators can verify upload, validation, and history wiring.
          </p>
          {message ? <p className="muted-copy">{message}</p> : null}
        </SectionCard>
      ) : null}

      <SectionCard title="Imports History" subtitle="Latest import jobs available to operators without leaving the application shell.">
        <table className="data-table">
          <thead>
            <tr>
              <th>Type</th>
              <th>Status</th>
              <th>File</th>
              <th>Summary</th>
              <th>Created</th>
            </tr>
          </thead>
          <tbody>
            {data?.rows.map((row) => (
              <tr key={row.id}>
                <td>{row.importType}</td>
                <td>{row.status}</td>
                <td>{row.fileName}</td>
                <td>{row.summary}</td>
                <td>{row.createdAt}</td>
              </tr>
            ))}
          </tbody>
        </table>
      </SectionCard>
    </div>
  )
}
