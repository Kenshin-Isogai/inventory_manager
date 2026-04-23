import { Outlet } from 'react-router-dom'
import { Sidebar } from './Sidebar'
import { MobileSidebar } from './MobileSidebar'
import { Header } from './Header'
import { SidebarProvider } from './SidebarContext'

export function MainLayout() {
  return (
    <SidebarProvider>
      <div className="flex h-screen bg-background overflow-hidden">
        {/* Desktop Sidebar */}
        <Sidebar />

        {/* Mobile Sidebar (Sheet overlay) */}
        <MobileSidebar />

        {/* Main Content */}
        <div className="flex-1 flex flex-col overflow-hidden min-w-0">
          <Header />
          <main className="flex-1 overflow-y-auto">
            <div className="animate-fade-in">
              <Outlet />
            </div>
          </main>
        </div>
      </div>
    </SidebarProvider>
  )
}
