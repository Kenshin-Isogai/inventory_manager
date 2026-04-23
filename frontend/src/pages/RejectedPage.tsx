import { Link } from 'react-router-dom'

import { SectionCard } from '../components/SectionCard'
import { useAuthSession } from '../hooks/useAuthSession'

export function RejectedPage() {
  const { data: session } = useAuthSession()
  return (
    <div className="auth-page">
      <SectionCard title="Rejected" subtitle="The current app user exists but does not have an approved role assignment.">
        <p className="muted-copy">{session?.user.rejectionReason || 'No rejection reason was recorded.'}</p>
        <Link to="/auth/register" className="inline-link">
          Re-submit registration
        </Link>
      </SectionCard>
    </div>
  )
}
