import { useDashboard } from '../hooks/useDashboard'
import { SectionCard } from '../components/SectionCard'

export function OperatorDashboardPage() {
  const { data } = useDashboard()

  return (
    <div className="page-grid">
      <SectionCard title="Operator Dashboard" subtitle="Requirements, shortages, and pending follow-up.">
        <div className="metric-grid">
          {data?.metrics.map((metric) => (
            <article key={metric.label} className="metric-card">
              <span>{metric.label}</span>
              <strong>{metric.value}</strong>
              <small>{metric.delta}</small>
            </article>
          ))}
        </div>
      </SectionCard>

      <SectionCard title="Alerts" subtitle="Projection-based operational alerts.">
        <ul className="list">
          {data?.alerts.map((alert) => <li key={alert}>{alert}</li>)}
        </ul>
      </SectionCard>
    </div>
  )
}
