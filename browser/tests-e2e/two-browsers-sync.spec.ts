// Full-stack E2E: two browser contexts (two "devices") share one user
// account. One writes a record, the other sees it arrive via the relay.
//
// Requirements:
//   - relay running at $APPUNVS_RELAY_BASE (default http://localhost:8080)
//   - Vite dev server running for browser/ (default http://localhost:5173)
//   - Redis behind the relay
//
// Run:
//   npm run dev            # in one terminal
//   /tmp/relay-server      # in another
//   npm run test:play      # here
import { test, expect, type Page } from '@playwright/test';

const RELAY_BASE = process.env.APPUNVS_RELAY_BASE ?? 'http://localhost:8080';

interface DeviceBoot {
  deviceId: string;
  deviceToken: string;
}

interface AccountBoot {
  userId: string;
  sessionToken: string;
  email: string;
}

async function signup(): Promise<AccountBoot> {
  const email = `pw-${Date.now()}-${Math.floor(Math.random() * 1e6)}@example.com`;
  const resp = await fetch(`${RELAY_BASE}/auth/signup`, {
    method: 'POST',
    headers: { 'content-type': 'application/json' },
    body: JSON.stringify({ email, password: 'hunter22' })
  });
  if (!resp.ok) throw new Error(`signup: ${resp.status}`);
  const body = (await resp.json()) as { user_id: string; session_token: string };
  return { userId: body.user_id, sessionToken: body.session_token, email };
}

async function registerDevice(session: string, deviceId: string): Promise<DeviceBoot> {
  const resp = await fetch(`${RELAY_BASE}/auth/register`, {
    method: 'POST',
    headers: {
      'content-type': 'application/json',
      Authorization: `Bearer ${session}`
    },
    body: JSON.stringify({ device_id: deviceId, platform: 'browser' })
  });
  if (!resp.ok) throw new Error(`register: ${resp.status}`);
  const body = (await resp.json()) as { token: string };
  return { deviceId, deviceToken: body.token };
}

async function bootPage(page: Page, account: AccountBoot, dev: DeviceBoot) {
  // Seed localStorage before the app boots so the layout skips login/register.
  await page.addInitScript(
    ([sess, uid, email, devId, devTok]) => {
      localStorage.setItem('appunvs.session_token', sess);
      localStorage.setItem('appunvs.user_id', uid);
      localStorage.setItem('appunvs.email', email);
      localStorage.setItem('appunvs.device_id', devId);
      localStorage.setItem('appunvs.token', devTok);
    },
    [account.sessionToken, account.userId, account.email, dev.deviceId, dev.deviceToken]
  );
  await page.goto('/');
  // Wait until the app boots into the dashboard (/).
  await expect(page.locator('dt', { hasText: 'conn' }).locator('+ dd')).toHaveText(
    'connected',
    { timeout: 10_000 }
  );
}

test('two browsers share the same namespace and see each other\'s records', async ({ browser }) => {
  const account = await signup();
  const devA = await registerDevice(account.sessionToken, 'pw-device-A');
  const devB = await registerDevice(account.sessionToken, 'pw-device-B');

  const ctxA = await browser.newContext();
  const ctxB = await browser.newContext();
  const pageA = await ctxA.newPage();
  const pageB = await ctxB.newPage();

  await bootPage(pageA, account, devA);
  await bootPage(pageB, account, devB);

  // Both pages should start with zero records.
  await expect(pageA.locator('h2')).toHaveText(/Records \(0\)/);
  await expect(pageB.locator('h2')).toHaveText(/Records \(0\)/);

  // Write on A.
  const marker = `pw-marker-${Date.now()}`;
  await pageA.locator('input[placeholder="data"]').fill(marker);
  await pageA.locator('button[type="submit"]', { hasText: 'add' }).click();

  // A sees it locally.
  await expect(pageA.locator('tbody tr td', { hasText: marker })).toBeVisible({
    timeout: 5_000
  });

  // B sees it arrive via the relay.
  await expect(pageB.locator('tbody tr td', { hasText: marker })).toBeVisible({
    timeout: 10_000
  });

  // B's record count should match A's.
  await expect(pageB.locator('h2')).toHaveText(/Records \(1\)/);

  await ctxA.close();
  await ctxB.close();
});
