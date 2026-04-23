import { StrictMode } from 'react'
import { createRoot } from 'react-dom/client'
import { BrowserRouter } from 'react-router-dom'
import { SWRConfig } from 'swr'
import { AuthTokenBridge } from './components/AuthTokenBridge'
import './index.css'
import App from './App.tsx'

createRoot(document.getElementById('root')!).render(
  <StrictMode>
    <SWRConfig value={{ provider: () => new Map() }}>
      <BrowserRouter>
        <AuthTokenBridge />
        <App />
      </BrowserRouter>
    </SWRConfig>
  </StrictMode>,
)
