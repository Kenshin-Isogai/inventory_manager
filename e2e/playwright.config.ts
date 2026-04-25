import { defineConfig, devices } from '@playwright/test';

/**
 * Playwright config for inventory_manager E2E tests.
 *
 * Prerequisites:
 *   1. docker compose up -d db
 *   2. cd backend && go run ./cmd/migrate up && go run ./cmd/server &
 *   3. cd frontend && npm install && npm run dev &
 *   4. npx playwright install
 *   5. npx playwright test
 */
export default defineConfig({
  testDir: './tests',
  fullyParallel: false, // sequential to avoid test data conflicts
  forbidOnly: !!process.env.CI,
  retries: process.env.CI ? 2 : 0,
  workers: 1,
  reporter: 'html',
  timeout: 30_000,

  use: {
    baseURL: process.env.FRONTEND_URL || 'http://localhost:5173',
    trace: 'on-first-retry',
    screenshot: 'only-on-failure',
  },

  projects: [
    {
      name: 'chromium',
      use: { ...devices['Desktop Chrome'] },
    },
  ],

  webServer: [
    {
      command: 'cd ../backend && go run ./cmd/server',
      port: 8080,
      reuseExistingServer: true,
      timeout: 60_000,
      env: {
        APP_ENV: 'test',
        APP_MODE: 'test',
        AUTH_MODE: 'none',
        RBAC_MODE: 'none',
        DATABASE_URL:
          process.env.DATABASE_URL ||
          'postgres://postgres:postgres@localhost:5433/inventory_manager_test?sslmode=disable',
      },
    },
    {
      command: 'cd ../frontend && npm run dev',
      port: 5173,
      reuseExistingServer: true,
      timeout: 60_000,
    },
  ],
});
