import { Navigate, Route, Routes } from 'react-router-dom'

import { AppShell } from './components/AppShell'
import { AuthGate } from './components/AuthGate'
import { AdminPage } from './pages/AdminPage'
import { AdminRolesPage } from './pages/AdminRolesPage'
import { AdminUsersPage } from './pages/AdminUsersPage'
import { InspectorArrivalsPage } from './pages/InspectorArrivalsPage'
import { InventoryEventsPage } from './pages/InventoryEventsPage'
import { InventoryPage } from './pages/InventoryPage'
import { LoginPage } from './pages/LoginPage'
import { OCRQueuePage } from './pages/OCRQueuePage'
import { OperatorImportsPage } from './pages/OperatorImportsPage'
import { OperatorDashboardPage } from './pages/OperatorDashboardPage'
import { PendingPage } from './pages/PendingPage'
import { ProcurementPage } from './pages/ProcurementPage'
import { RegisterPage } from './pages/RegisterPage'
import { RejectedPage } from './pages/RejectedPage'
import { ReservationsPage } from './pages/ReservationsPage'
import { ShortagesPage } from './pages/ShortagesPage'
import { VerifyEmailPage } from './pages/VerifyEmailPage'
import './App.css'

function App() {
  return (
    <Routes>
      <Route path="/auth/login" element={<LoginPage />} />
      <Route path="/auth/verify-email" element={<VerifyEmailPage />} />
      <Route path="/auth/register" element={<RegisterPage />} />
      <Route path="/auth/pending" element={<PendingPage />} />
      <Route path="/auth/rejected" element={<RejectedPage />} />
      <Route element={<AppShell />}>
        <Route element={<AuthGate app="operator" />}>
          <Route path="/operator/dashboard" element={<OperatorDashboardPage />} />
          <Route path="/operator/reservations" element={<ReservationsPage />} />
          <Route path="/operator/shortages" element={<ShortagesPage />} />
          <Route path="/operator/imports/upload" element={<OperatorImportsPage mode="upload" />} />
          <Route path="/operator/imports/history" element={<OperatorImportsPage mode="history" />} />
        </Route>
        <Route element={<AuthGate app="inventory" />}>
          <Route path="/inventory/stock" element={<InventoryPage />} />
          <Route path="/inventory/events" element={<InventoryEventsPage />} />
        </Route>
        <Route element={<AuthGate app="procurement" />}>
          <Route path="/procurement/requests" element={<ProcurementPage />} />
          <Route path="/procurement/ocr" element={<OCRQueuePage />} />
        </Route>
        <Route element={<AuthGate app="inspector" />}>
          <Route path="/inspector/arrivals" element={<InspectorArrivalsPage />} />
        </Route>
        <Route element={<AuthGate app="admin" />}>
          <Route path="/admin/users" element={<AdminUsersPage />} />
          <Route path="/admin/roles" element={<AdminRolesPage />} />
          <Route path="/admin/master-data" element={<AdminPage />} />
        </Route>
        <Route path="/" element={<Navigate to="/operator/dashboard" replace />} />
        <Route path="*" element={<Navigate to="/operator/dashboard" replace />} />
      </Route>
    </Routes>
  )
}

export default App
