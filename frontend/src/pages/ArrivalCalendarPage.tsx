import { useState, useMemo } from 'react'
import { Card, CardContent, CardHeader, CardTitle, CardDescription } from '../components/ui/card'
import { Badge } from '../components/ui/badge'
import { Button } from '../components/ui/button'
import { ItemInfoPopover } from '../components/ItemInfoPopover'
import { useArrivalCalendar } from '../hooks/useArrivalCalendar'
import type { ArrivalCalendarDay } from '../types'

function getYearMonth(offset: number): string {
  const d = new Date()
  d.setMonth(d.getMonth() + offset)
  return `${d.getFullYear()}-${String(d.getMonth() + 1).padStart(2, '0')}`
}

function formatYearMonth(ym: string): string {
  const [y, m] = ym.split('-')
  const months = ['Jan', 'Feb', 'Mar', 'Apr', 'May', 'Jun', 'Jul', 'Aug', 'Sep', 'Oct', 'Nov', 'Dec']
  return `${months[parseInt(m, 10) - 1]} ${y}`
}

function buildCalendarGrid(ym: string, days: ArrivalCalendarDay[]): (ArrivalCalendarDay | null)[][] {
  const [year, month] = ym.split('-').map(Number)
  const firstDay = new Date(year, month - 1, 1).getDay()
  const daysInMonth = new Date(year, month, 0).getDate()
  const dayMap = new Map(days.map((d) => [d.date, d]))

  const grid: (ArrivalCalendarDay | null)[][] = []
  let week: (ArrivalCalendarDay | null)[] = Array(firstDay).fill(null)

  for (let day = 1; day <= daysInMonth; day++) {
    const dateStr = `${year}-${String(month).padStart(2, '0')}-${String(day).padStart(2, '0')}`
    week.push(dayMap.get(dateStr) ?? { date: dateStr, items: [] })
    if (week.length === 7) {
      grid.push(week)
      week = []
    }
  }
  if (week.length > 0) {
    while (week.length < 7) week.push(null)
    grid.push(week)
  }
  return grid
}

export function ArrivalCalendarPage() {
  const [monthOffset, setMonthOffset] = useState(0)
  const [selectedDay, setSelectedDay] = useState<ArrivalCalendarDay | null>(null)
  const yearMonth = useMemo(() => getYearMonth(monthOffset), [monthOffset])
  const { data } = useArrivalCalendar(yearMonth)

  const grid = useMemo(
    () => buildCalendarGrid(yearMonth, data?.days ?? []),
    [yearMonth, data],
  )

  return (
    <div className="space-y-6 p-6">
      <div className="space-y-2">
        <h1 className="text-3xl font-bold">Arrival Calendar</h1>
        <p className="text-muted-foreground">Expected arrivals by date from purchase order lines.</p>
      </div>

      <Card>
        <CardHeader>
          <div className="flex items-center justify-between">
            <CardTitle>{formatYearMonth(yearMonth)}</CardTitle>
            <div className="flex gap-2">
              <Button variant="outline" size="sm" onClick={() => setMonthOffset((p) => p - 1)}>
                Prev
              </Button>
              <Button variant="outline" size="sm" onClick={() => setMonthOffset(0)}>
                Today
              </Button>
              <Button variant="outline" size="sm" onClick={() => setMonthOffset((p) => p + 1)}>
                Next
              </Button>
            </div>
          </div>
        </CardHeader>
        <CardContent>
          <div className="grid grid-cols-7 gap-px bg-border rounded overflow-hidden">
            {['Sun', 'Mon', 'Tue', 'Wed', 'Thu', 'Fri', 'Sat'].map((d) => (
              <div key={d} className="bg-muted px-2 py-1 text-center text-xs font-medium text-muted-foreground">
                {d}
              </div>
            ))}
            {grid.flat().map((cell, i) => {
              if (!cell) {
                return <div key={`empty-${i}`} className="bg-background min-h-[5rem]" />
              }
              const dayNum = parseInt(cell.date.split('-')[2], 10)
              const hasItems = cell.items.length > 0
              const isSelected = selectedDay?.date === cell.date
              return (
                <div
                  key={cell.date}
                  className={`bg-background min-h-[5rem] p-1.5 cursor-pointer transition-colors ${
                    isSelected ? 'ring-2 ring-primary ring-inset' : ''
                  } ${hasItems ? 'hover:bg-muted/50' : ''}`}
                  onClick={() => hasItems && setSelectedDay(cell)}
                >
                  <div className="flex items-center justify-between">
                    <span className="text-sm tabular-nums">{dayNum}</span>
                    {hasItems && (
                      <Badge variant="secondary" className="text-[10px] px-1 py-0">
                        {cell.items.length}
                      </Badge>
                    )}
                  </div>
                  {cell.items.slice(0, 2).map((item, j) => (
                    <div key={j} className="text-[10px] text-muted-foreground truncate mt-0.5">
                      {item.itemNumber} x{item.quantity}
                    </div>
                  ))}
                  {cell.items.length > 2 && (
                    <div className="text-[10px] text-muted-foreground">+{cell.items.length - 2} more</div>
                  )}
                </div>
              )
            })}
          </div>
        </CardContent>
      </Card>

      {selectedDay && selectedDay.items.length > 0 && (
        <Card>
          <CardHeader>
            <CardTitle>Arrivals on {selectedDay.date}</CardTitle>
            <CardDescription>{selectedDay.items.length} item(s) expected</CardDescription>
          </CardHeader>
          <CardContent>
            <div className="space-y-3">
              {selectedDay.items.map((item, i) => (
                <div key={i} className="flex items-start justify-between border-b pb-2 last:border-0">
                  <div className="space-y-0.5">
                    <ItemInfoPopover
                      itemNumber={item.itemNumber}
                      description={item.description}
                      manufacturer={item.manufacturer}
                    />
                    <p className="text-xs text-muted-foreground">
                      {item.manufacturer} &middot; {item.description}
                    </p>
                    <p className="text-xs text-muted-foreground">
                      PO: {item.purchaseOrderNumber}
                      {item.quotationNumber && ` | Quote: ${item.quotationNumber}`}
                      {item.supplierName && ` | ${item.supplierName}`}
                    </p>
                  </div>
                  <Badge variant="outline" className="tabular-nums">
                    x{item.quantity}
                  </Badge>
                </div>
              ))}
            </div>
          </CardContent>
        </Card>
      )}
    </div>
  )
}
