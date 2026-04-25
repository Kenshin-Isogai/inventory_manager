import { useMemo } from 'react'
import { format, parseISO, isAfter } from 'date-fns'
import { AlertCircle, Calendar, ArrowRight, CheckCircle2, Clock } from 'lucide-react'
import { Card, CardContent, CardHeader, CardTitle, CardDescription } from '@/components/ui/card'
import { Badge } from '@/components/ui/badge'
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from '@/components/ui/table'
import { useShortageTimeline } from '@/hooks/useShortageTimeline'
import { ItemInfoPopover } from '@/components/ItemInfoPopover'

interface ShortageTimelineViewProps {
  device: string
  scope: string
}

export function ShortageTimelineView({ device, scope }: ShortageTimelineViewProps) {
  const { data, isLoading, error } = useShortageTimeline(device, scope)

  const timelineData = useMemo(() => {
    if (!data) return null
    const startAt = data.plannedStartAt ? parseISO(data.plannedStartAt) : null
    
    const rows = data.rows.map(row => {
      const delayed = row.delayedArrivals.map(d => ({
        ...d,
        date: parseISO(d.expectedDate)
      }))
      
      return {
        ...row,
        delayed,
        isCritical: row.shortageAtStart > 0 || delayed.some(d => startAt && isAfter(d.date, startAt))
      }
    })

    const criticalCount = rows.filter(r => r.isCritical).length

    return { ...data, rows, startAt, criticalCount }
  }, [data])

  if (!device || !scope) return null
  if (isLoading) return <div className="p-8 text-center text-muted-foreground">Loading timeline...</div>
  if (error) return <div className="p-8 text-center text-destructive">Error loading timeline data.</div>
  if (!timelineData || timelineData.rows.length === 0) return null

  const { rows, startAt, criticalCount } = timelineData

  return (
    <Card className="border-orange-200 bg-orange-50/30">
      <CardHeader className="pb-2">
        <div className="flex items-center justify-between">
          <div className="space-y-1">
            <CardTitle className="flex items-center gap-2 text-orange-800">
              <Clock className="w-5 h-5" />
              Shortage Timeline Risk Analysis
            </CardTitle>
            <CardDescription className="text-orange-700">
              Analysis of item availability relative to scope start date: 
              <span className="font-bold ml-1">
                {startAt ? format(startAt, 'PPP') : 'Not Set'}
              </span>
            </CardDescription>
          </div>
          <Badge variant={criticalCount > 0 ? "destructive" : "outline"} className="px-3 py-1">
            {criticalCount} Critical Risks
          </Badge>
        </div>
      </CardHeader>
      <CardContent>
        <div className="overflow-x-auto">
          <Table>
            <TableHeader>
              <TableRow className="border-orange-200 hover:bg-transparent">
                <TableHead className="text-orange-900">Item</TableHead>
                <TableHead className="text-right text-orange-900">Required</TableHead>
                <TableHead className="text-right text-orange-900">Avail. by Start</TableHead>
                <TableHead className="text-right text-orange-900">Shortage at Start</TableHead>
                <TableHead className="text-orange-900">Delayed Arrivals (After Start)</TableHead>
                <TableHead className="text-orange-900">Status</TableHead>
              </TableRow>
            </TableHeader>
            <TableBody>
              {rows.map((row) => (
                <TableRow key={row.itemId} className="border-orange-100 hover:bg-orange-100/50">
                  <TableCell>
                    <ItemInfoPopover
                      itemNumber={row.itemNumber}
                      description={row.description}
                      manufacturer={row.manufacturer}
                    />
                  </TableCell>
                  <TableCell className="text-right tabular-nums">{row.requiredQuantity}</TableCell>
                  <TableCell className="text-right tabular-nums font-medium text-green-700">
                    {row.availableByStart}
                  </TableCell>
                  <TableCell className="text-right tabular-nums">
                    {row.shortageAtStart > 0 ? (
                      <span className="text-destructive font-bold">{row.shortageAtStart}</span>
                    ) : (
                      <span className="text-muted-foreground">0</span>
                    )}
                  </TableCell>
                  <TableCell>
                    <div className="space-y-1">
                      {row.delayedArrivals.length > 0 ? (
                        row.delayedArrivals.map((d, idx) => (
                          <div key={idx} className="flex items-center gap-2 text-xs">
                            <Calendar className="w-3 h-3 text-muted-foreground" />
                            <span className="font-mono text-blue-700">{format(parseISO(d.expectedDate), 'MM/dd')}</span>
                            <ArrowRight className="w-2 h-2 text-muted-foreground" />
                            <Badge variant="outline" className="h-4 px-1 text-[10px] font-normal">
                              {d.quantity} units
                            </Badge>
                            <span className="text-[10px] text-muted-foreground truncate max-w-[80px]">
                              PO: {d.purchaseOrderNumber}
                            </span>
                          </div>
                        ))
                      ) : (
                        <span className="text-xs text-muted-foreground">No delayed arrivals tracked</span>
                      )}
                    </div>
                  </TableCell>
                  <TableCell>
                    {row.isCritical ? (
                      <Badge variant="destructive" className="gap-1">
                        <AlertCircle className="w-3 h-3" />
                        Critical
                      </Badge>
                    ) : (
                      <Badge variant="secondary" className="gap-1 bg-green-100 text-green-800 hover:bg-green-100">
                        <CheckCircle2 className="w-3 h-3" />
                        Manageable
                      </Badge>
                    )}
                  </TableCell>
                </TableRow>
              ))}
            </TableBody>
          </Table>
        </div>
      </CardContent>
    </Card>
  )
}
