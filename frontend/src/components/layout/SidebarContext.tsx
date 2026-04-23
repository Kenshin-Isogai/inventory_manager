/* eslint-disable react-refresh/only-export-components */
import { createContext, useContext, useState, useCallback } from 'react'
import type { ReactNode } from 'react'

type SidebarState = {
  collapsed: boolean
  mobileOpen: boolean
  toggle: () => void
  setMobileOpen: (open: boolean) => void
}

const SidebarContext = createContext<SidebarState>({
  collapsed: false,
  mobileOpen: false,
  toggle: () => {},
  setMobileOpen: () => {},
})

export function SidebarProvider({ children }: { children: ReactNode }) {
  const [collapsed, setCollapsed] = useState(false)
  const [mobileOpen, setMobileOpen] = useState(false)

  const toggle = useCallback(() => setCollapsed((prev) => !prev), [])

  return (
    <SidebarContext.Provider value={{ collapsed, mobileOpen, toggle, setMobileOpen }}>
      {children}
    </SidebarContext.Provider>
  )
}

export function useSidebar() {
  return useContext(SidebarContext)
}
