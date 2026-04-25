/**
 * E2E test helpers for inventory_manager.
 * Provides API-level seed/reset functions so Playwright tests can start from a known state.
 */

const API_BASE = process.env.API_BASE_URL || 'http://localhost:8080';

export async function apiPost(path: string, body: Record<string, unknown>) {
  const res = await fetch(`${API_BASE}${path}`, {
    method: 'POST',
    headers: {
      'Content-Type': 'application/json',
      Authorization: 'Bearer local-admin-token',
    },
    body: JSON.stringify(body),
  });
  return { status: res.status, body: await res.json().catch(() => null) };
}

export async function apiGet(path: string) {
  const res = await fetch(`${API_BASE}${path}`, {
    headers: { Authorization: 'Bearer local-admin-token' },
  });
  return { status: res.status, body: await res.json().catch(() => null) };
}

export async function resetTestData() {
  const res = await fetch(`${API_BASE}/api/v1/test/reset`, {
    method: 'POST',
    headers: { Authorization: 'Bearer local-admin-token' },
  });
  if (!res.ok) {
    throw new Error(`reset failed with status ${res.status}`);
  }
}

/**
 * Seed inventory via API so the UI has data to display.
 */
export async function seedInventoryViaAPI(options?: { reset?: boolean }) {
  if (options?.reset) {
    await resetTestData();
  }

  // Receive some items into inventory
  await apiPost('/api/v1/inventory/receives', {
    itemId: 'item-er2',
    locationCode: 'TOKYO-A1',
    quantity: 50,
    note: 'E2E test seed',
  });

  await apiPost('/api/v1/inventory/receives', {
    itemId: 'item-mk44',
    locationCode: 'TOKYO-B2',
    quantity: 30,
    note: 'E2E test seed',
  });

  // Create a reservation
  await apiPost('/api/v1/operator/reservations', {
    itemId: 'item-er2',
    deviceScopeId: 'ds-er2-powerboard',
    quantity: 10,
    purpose: 'E2E test',
  });

  const { body } = await apiGet('/api/v1/operator/reservations');
  return {
    reservationId: body?.data?.rows?.[0]?.id as string | undefined,
  };
}
