import { lazy, Suspense } from 'react'
import { Navigate, Route, Routes } from 'react-router-dom'

import { MainLayout } from './components/layout/MainLayout'
import { AuthLayout } from './components/layout/AuthLayout'
import { AuthGate } from './components/AuthGate'

// Auth Pages
import { LoginPage } from './pages/LoginPage'
import { VerifyEmailPage } from './pages/VerifyEmailPage'
import { RegisterPage } from './pages/RegisterPage'
import { PendingPage } from './pages/PendingPage'
import { RejectedPage } from './pages/RejectedPage'

// Portal
import { PortalPage } from './pages/PortalPage'

// Lazy loaded pages
const OperatorRequirementsPage = lazy(() =>
  import('./pages/OperatorDashboardPage').then((m) => ({ default: m.OperatorDashboardPage }))
)
const ReservationsPage = lazy(() => import('./pages/ReservationsPage').then((m) => ({ default: m.ReservationsPage })))
const ShortagesPage = lazy(() => import('./pages/ShortagesPage').then((m) => ({ default: m.ShortagesPage })))
const OperatorImportsPage = lazy(() => import('./pages/OperatorImportsPage').then((m) => ({ default: m.OperatorImportsPage })))

const InventoryItemsPage = lazy(() =>
  import('./pages/InventoryPage').then((m) => ({ default: m.InventoryPage }))
)
const InventoryLocationsPage = lazy(() =>
  import('./pages/InventoryPage').then((m) => ({ default: m.InventoryPage }))
)
const InventoryEventsPage = lazy(() =>
  import('./pages/InventoryEventsPage').then((m) => ({ default: m.InventoryEventsPage }))
)

const ProcurementRequestsPage = lazy(() =>
  import('./pages/ProcurementPage').then((m) => ({ default: m.ProcurementPage }))
)
const ProcurementRequestDetailPage = lazy(() =>
  import('./pages/ProcurementPage').then((m) => ({ default: m.ProcurementPage }))
)
const OCRQueuePage = lazy(() => import('./pages/OCRQueuePage').then((m) => ({ default: m.OCRQueuePage })))
const ProcurementDraftPage = lazy(() =>
  import('./pages/ProcurementPage').then((m) => ({ default: m.ProcurementPage }))
)

const InspectorArrivalsPage = lazy(() =>
  import('./pages/InspectorArrivalsPage').then((m) => ({ default: m.InspectorArrivalsPage }))
)

const AdminUsersPage = lazy(() =>
  import('./pages/AdminUsersPage').then((m) => ({ default: m.AdminUsersPage }))
)
const AdminRolesPage = lazy(() =>
  import('./pages/AdminRolesPage').then((m) => ({ default: m.AdminRolesPage }))
)
const AdminMasterPage = lazy(() => import('./pages/AdminPage').then((m) => ({ default: m.AdminPage })))

// Additional spec 042401 pages
const ScopeOverviewPage = lazy(() =>
  import('./pages/ScopeOverviewPage').then((m) => ({ default: m.ScopeOverviewPage }))
)
const ItemFlowPage = lazy(() =>
  import('./pages/ItemFlowPage').then((m) => ({ default: m.ItemFlowPage }))
)
const ArrivalCalendarPage = lazy(() =>
  import('./pages/ArrivalCalendarPage').then((m) => ({ default: m.ArrivalCalendarPage }))
)

// Loading fallback
function PageLoader() {
  return (
    <div className="flex items-center justify-center h-96">
      <div className="animate-spin">Loading...</div>
    </div>
  )
}

function AuthLayoutWrapper({ children }: { children: React.ReactNode }) {
  return (
    <AuthLayout>
      <Suspense fallback={<PageLoader />}>{children}</Suspense>
    </AuthLayout>
  )
}

function App() {
  return (
    <Routes>
      {/* Root redirect */}
      <Route path="/" element={<Navigate to="/app/portal" replace />} />

      {/* Auth Routes */}
      <Route path="/auth/login" element={<AuthLayoutWrapper><LoginPage /></AuthLayoutWrapper>} />
      <Route path="/auth/verify-email" element={<AuthLayoutWrapper><VerifyEmailPage /></AuthLayoutWrapper>} />
      <Route path="/auth/register" element={<AuthLayoutWrapper><RegisterPage /></AuthLayoutWrapper>} />
      <Route path="/auth/pending" element={<AuthLayoutWrapper><PendingPage /></AuthLayoutWrapper>} />
      <Route path="/auth/rejected" element={<AuthLayoutWrapper><RejectedPage /></AuthLayoutWrapper>} />

      {/* App Routes with MainLayout */}
      <Route element={<MainLayout />}>
        {/* Portal */}
        <Route path="/app/portal" element={<PortalPage />} />

        {/* Operator Routes */}
        <Route element={<AuthGate app="operator" />}>
          <Route
            path="/app/operator/requirements"
            element={
              <Suspense fallback={<PageLoader />}>
                <OperatorRequirementsPage />
              </Suspense>
            }
          />
          <Route
            path="/app/operator/reservations"
            element={
              <Suspense fallback={<PageLoader />}>
                <ReservationsPage />
              </Suspense>
            }
          />
          <Route
            path="/app/operator/shortage"
            element={
              <Suspense fallback={<PageLoader />}>
                <ShortagesPage />
              </Suspense>
            }
          />
          <Route
            path="/app/operator/scopes"
            element={
              <Suspense fallback={<PageLoader />}>
                <ScopeOverviewPage />
              </Suspense>
            }
          />
          <Route
            path="/app/operator/items/import"
            element={
              <Suspense fallback={<PageLoader />}>
                <OperatorImportsPage />
              </Suspense>
            }
          />
        </Route>

        {/* Inventory Routes */}
        <Route element={<AuthGate app="inventory" />}>
          <Route
            path="/app/inventory/items"
            element={
              <Suspense fallback={<PageLoader />}>
                <InventoryItemsPage />
              </Suspense>
            }
          />
          <Route
            path="/app/inventory/locations"
            element={
              <Suspense fallback={<PageLoader />}>
                <InventoryLocationsPage />
              </Suspense>
            }
          />
          <Route
            path="/app/inventory/items/:id/flow"
            element={
              <Suspense fallback={<PageLoader />}>
                <ItemFlowPage />
              </Suspense>
            }
          />
          <Route
            path="/app/inventory/arrivals/calendar"
            element={
              <Suspense fallback={<PageLoader />}>
                <ArrivalCalendarPage />
              </Suspense>
            }
          />
          <Route
            path="/app/inventory/events"
            element={
              <Suspense fallback={<PageLoader />}>
                <InventoryEventsPage />
              </Suspense>
            }
          />
        </Route>

        {/* Procurement Routes */}
        <Route element={<AuthGate app="procurement" />}>
          <Route
            path="/app/procurement/requests"
            element={
              <Suspense fallback={<PageLoader />}>
                <ProcurementRequestsPage />
              </Suspense>
            }
          />
          <Route
            path="/app/procurement/requests/:id"
            element={
              <Suspense fallback={<PageLoader />}>
                <ProcurementRequestDetailPage />
              </Suspense>
            }
          />
          <Route
            path="/app/procurement/ocr-queue"
            element={
              <Suspense fallback={<PageLoader />}>
                <OCRQueuePage />
              </Suspense>
            }
          />
          <Route
            path="/app/procurement/drafts/:id"
            element={
              <Suspense fallback={<PageLoader />}>
                <ProcurementDraftPage />
              </Suspense>
            }
          />
        </Route>

        {/* Inspector Routes */}
        <Route element={<AuthGate app="inspector" />}>
          <Route
            path="/app/inspector/arrivals"
            element={
              <Suspense fallback={<PageLoader />}>
                <InspectorArrivalsPage />
              </Suspense>
            }
          />
        </Route>

        {/* Admin Routes */}
        <Route element={<AuthGate app="admin" />}>
          <Route
            path="/app/admin/users"
            element={
              <Suspense fallback={<PageLoader />}>
                <AdminUsersPage />
              </Suspense>
            }
          />
          <Route
            path="/app/admin/roles"
            element={
              <Suspense fallback={<PageLoader />}>
                <AdminRolesPage />
              </Suspense>
            }
          />
          <Route
            path="/app/admin/master"
            element={
              <Suspense fallback={<PageLoader />}>
                <AdminMasterPage />
              </Suspense>
            }
          />
        </Route>

        {/* Catch-all for /app routes */}
        <Route path="/app/*" element={<Navigate to="/app/portal" replace />} />
      </Route>

      {/* Catch-all */}
      <Route path="*" element={<Navigate to="/app/portal" replace />} />
    </Routes>
  )
}

export default App
                                                                                                                                                                                                                                                                       