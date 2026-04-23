import { SectionCard } from '../components/SectionCard'

export function PendingPage() {
  return (
    <div className="auth-page">
      <SectionCard title="Pending Approval" subtitle="The identity is known, but app access is still waiting for admin approval.">
        <p className="muted-copy">An admin needs to assign app roles before this account can enter the operator, inventory, procurement, or admin areas.</p>
      </SectionCard>
    </div>
  )
}
