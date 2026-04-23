import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card'
import { Badge } from '@/components/ui/badge'
import { useDashboard } from '@/hooks/useDashboard'
import { AlertCircle } from 'lucide-react'

export function OperatorDashboardPage() {
  const { data } = useDashboard()

  return (
    <div className="space-y-6 p-6">
      <div className="space-y-2">
        <h1 className="text-3xl font-bold tracking-tight">Operator Dashboard</h1>
        <p className="text-muted-foreground">
          Requirements, shortages, and pending follow-up.
        </p>
      </div>

      <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-4 gap-4">
        {data?.metrics.map((metric) => (
          <Card key={metric.label}>
            <CardContent className="pt-6">
              <div className="space-y-2">
                <p className="text-sm text-muted-foreground">{metric.label}</p>
                <div className="flex items-baseline justify-between">
                  <p className="text-2xl font-bold">{metric.value}</p>
                  {metric.delta && (
                    <Badge variant="outline" className="text-xs">
                      {metric.delta}
                    </Badge>
                  )}
                </div>
              </div>
            </CardContent>
          </Card>
        ))}
      </div>

      {data?.alerts && data.alerts.length > 0 && (
        <Card className="border-amber-200 bg-amber-50">
          <CardHeader>
            <CardTitle className="flex items-center gap-2 text-lg">
              <AlertCircle className="w-5 h-5 text-amber-600" />
              Alerts
            </CardTitle>
            <CardDescription>Projection-based operational alerts</CardDescription>
          </CardHeader>
          <CardContent>
            <ul className="space-y-2">
              {data.alerts.map((alert, idx) => (
                <li key={idx} className="text-sm text-amber-900 flex gap-2">
                  <span className="text-amber-600">•</span>
                  <span>{alert}</span>
                </li>
              ))}
            </ul>
          </CardContent>
        </Card>
      )}
    </div>
  )
}
