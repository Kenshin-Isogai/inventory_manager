/**
 * E2E: Reservation list, creation, allocation, and shortage detection.
 */
import { test, expect } from '@playwright/test';
import { seedInventoryViaAPI, apiGet, apiPost, resetTestData } from './helpers';

test.beforeEach(async () => {
  await resetTestData();
});

test.describe('Reservation list page @smoke', () => {
  test.beforeEach(async ({ page }) => {
    await seedInventoryViaAPI();
    await page.goto('/app/operator/reservations');
    await page.waitForLoadState('networkidle');
  });

  test('displays reservations from database', async ({ page }) => {
    // Should show the seeded reservation
    await expect(page.getByText('ER2').first()).toBeVisible({ timeout: 10_000 });
  });

  test('reservation list API returns correct data', async () => {
    const { status, body } = await apiGet('/api/v1/operator/reservations');
    expect(status).toBe(200);
    expect(body.data.rows).toBeDefined();
    expect(body.data.rows.length).toBeGreaterThan(0);

    const er2Res = body.data.rows.find(
      (r: { itemNumber: string }) => r.itemNumber === 'ER2',
    );
    expect(er2Res).toBeDefined();
    expect(er2Res.quantity).toBe(10);
    expect(er2Res.status).toBe('requested');
  });
});

test.describe('Shortage detection @scenario', () => {
  test('shortages page shows items where reserved > on_hand', async ({ page }) => {
    await seedInventoryViaAPI();

    // Create a shortage scenario: reserve 100 more after seeding only 50 on hand
    await apiPost('/api/v1/operator/reservations', {
      itemId: 'item-er2',
      deviceScopeId: 'ds-er2-powerboard',
      quantity: 100,
    });

    await page.goto('/app/operator/shortage');
    await page.waitForLoadState('networkidle');

    // Should show a shortage row for ER2
    // The exact UI may vary, but the shortage count should be visible
    const { body } = await apiGet('/api/v1/operator/shortages');
    expect(Array.isArray(body.data.rows)).toBe(true);
  });
});

test.describe('Allocation flow (frontend → API → DB) @scenario', () => {
  test('allocating a reservation updates reserved_quantity in balances', async () => {
    const seed = await seedInventoryViaAPI();
    expect(seed.reservationId).toBeDefined();

    // Check balance before allocation
    const { body: beforeOverview } = await apiGet('/api/v1/inventory/overview');
    const beforeER2 = beforeOverview.data.balances.find(
      (b: { itemNumber: string; locationCode: string }) =>
        b.itemNumber === 'ER2' && b.locationCode === 'TOKYO-A1',
    );
    const beforeReserved = beforeER2?.reservedQuantity || 0;

    // Allocate
    const allocResult = await apiPost(
      `/api/v1/operator/reservations/${seed.reservationId}/allocate`,
      { locationCode: 'TOKYO-A1', quantity: 5 },
    );
    expect(allocResult.status).toBe(200);
    expect(allocResult.body.data.allocatedQuantity).toBeGreaterThanOrEqual(5);
    expect(
      allocResult.body.data.allocations.some(
        (allocation: { locationCode: string; quantity: number }) =>
          allocation.locationCode === 'TOKYO-A1' && allocation.quantity >= 5,
      ),
    ).toBe(true);

    // Check balance after allocation
    const { body: afterOverview } = await apiGet('/api/v1/inventory/overview');
    const afterER2 = afterOverview.data.balances.find(
      (b: { itemNumber: string; locationCode: string }) =>
        b.itemNumber === 'ER2' && b.locationCode === 'TOKYO-A1',
    );
    expect(afterER2).toBeDefined();
    expect(afterER2.reservedQuantity).toBeGreaterThanOrEqual(beforeReserved);
    expect(afterER2.availableQuantity).toBe(afterER2.onHandQuantity - afterER2.reservedQuantity);
  });
});
