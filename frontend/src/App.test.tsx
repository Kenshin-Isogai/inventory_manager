import '@testing-library/jest-dom/vitest'
import { render } from '@testing-library/react'
import { MemoryRouter } from 'react-router-dom'
import { describe, expect, it } from 'vitest'

import App from './App'
import { clearStoredToken } from './lib/auth'

describe('App', () => {
  it('renders without crashing', () => {
    clearStoredToken()
    const { container } = render(
      <MemoryRouter initialEntries={['/auth/login']}>
        <App />
      </MemoryRouter>,
    )

    expect(container).toBeTruthy()
  })
})
