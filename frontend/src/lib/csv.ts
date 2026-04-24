export function downloadTextFile(fileName: string, content: string, type = 'text/csv;charset=utf-8') {
  const blob = new Blob([content], { type })
  const url = URL.createObjectURL(blob)
  const anchor = document.createElement('a')
  anchor.href = url
  anchor.download = fileName
  anchor.click()
  URL.revokeObjectURL(url)
}

export function parseCSV(text: string): Record<string, string>[] {
  const rows: string[][] = []
  let field = ''
  let row: string[] = []
  let inQuotes = false
  const normalized = text.replace(/^\uFEFF/, '')

  for (let index = 0; index < normalized.length; index += 1) {
    const char = normalized[index]
    if (char === '"') {
      if (inQuotes && normalized[index + 1] === '"') {
        field += '"'
        index += 1
      } else {
        inQuotes = !inQuotes
      }
      continue
    }
    if (char === ',' && !inQuotes) {
      row.push(field.trim())
      field = ''
      continue
    }
    if ((char === '\n' || char === '\r') && !inQuotes) {
      if (char === '\r' && normalized[index + 1] === '\n') {
        index += 1
      }
      row.push(field.trim())
      field = ''
      if (row.some((value) => value.length > 0)) {
        rows.push(row)
      }
      row = []
      continue
    }
    field += char
  }

  row.push(field.trim())
  if (row.some((value) => value.length > 0)) {
    rows.push(row)
  }

  if (rows.length === 0) {
    return []
  }

  const headers = rows[0].map((header) => header.trim().toLowerCase().replace(/\s+/g, '_'))
  return rows.slice(1).map((values) =>
    headers.reduce<Record<string, string>>((record, header, index) => {
      record[header] = values[index] ?? ''
      return record
    }, {}),
  )
}

export function toPositiveInt(value: string, fallback = 0) {
  const parsed = Number.parseInt(value, 10)
  return Number.isFinite(parsed) && parsed > 0 ? parsed : fallback
}

export function toInt(value: string, fallback = 0) {
  const parsed = Number.parseInt(value, 10)
  return Number.isFinite(parsed) ? parsed : fallback
}
