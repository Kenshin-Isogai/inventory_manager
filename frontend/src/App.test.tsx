import '@testing-library/jest-dom/vitest'
import { render, screen } from '@testing-library/react'
import { BrowserRouter } from 'react-router-dom'
import { describe, expect, it } from 'vitest'

import App from './App'
import { clearStoredToken } from './lib/auth'

describe('App', () => {
  it('renders the operator shell by default', async () => {
    clearStoredToken()
    render(
      <BrowserRouter>
        <App />
      </BrowserRouter>,
    )

    expect(await screen.findByText('Inventory Operations Skeleton')).toBeInTheDocument()
    expect(screen.getByText('Operator Dashboard')).toBeInTheDocument()
  })
})
