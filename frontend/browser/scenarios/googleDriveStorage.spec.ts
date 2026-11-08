import AxeBuilder from '@axe-core/playwright';
import { expect, type BrowserContext, type Page } from '@playwright/test';

import type { components } from '../../src/shared/api/schema.gen';
import { applicationFixtures } from '../applicationFixtures';
import { test } from '../browserTest';
import { apiResponse } from '../HttpFixture';

const secret = 'synthetic-drive-client-secret';
const tokens = {
  access_token: 'synthetic-drive-access',
  refresh_token: 'synthetic-drive-refresh',
  token_type: 'Bearer',
  expires_in: 3600,
};
const storage: components['schemas']['Storage'] = {
  id: 'existing-storage',
  workspaceId: 'workspace-id',
  name: 'Existing archive',
  settings: { type: 'LOCAL' },
};
const database: components['schemas']['Database'] = {
  id: 'logical-id',
  workspaceId: 'workspace-id',
  name: 'Orders',
  notifierIds: [],
  settings: {
    type: 'POSTGRES_LOGICAL',
    host: 'postgres.example.com',
    port: 5432,
    username: 'backup',
    database: 'orders',
    sslMode: 'disable',
    sshTunnel: { isEnabled: false },
    includeSchemas: [],
    excludeTables: [],
    excludeTableData: [],
    cpuCount: 1,
    isSkipUserMappings: false,
  },
};
const config: components['schemas']['BackupConfig'] = {
  cleanup: { pendingFileCount: 0, failedFileCount: 0 },
  databaseId: database.id,
  storageId: storage.id,
  isBackupsEnabled: true,
  encryption: 'NONE',
  settings: {
    type: 'LOGICAL',
    schedule: { type: 'CRON', cronExpression: '0 2 * * *' },
    retention: {
      policy: 'COUNT',
      count: 5,
      period: 'WEEK',
      gfsHours: 0,
      gfsDays: 0,
      gfsWeeks: 0,
      gfsMonths: 0,
      gfsYears: 0,
    },
    isRetryIfFailed: false,
    maxFailedTriesCount: 1,
    notificationTriggers: [],
  },
};

test.use({
  httpFixtures: [
    [
      ...applicationFixtures().filter(
        (fixture) => !['/databases', '/storages', '/backup-configs'].includes(fixture.path),
      ),
      apiResponse({ method: 'get', path: '/databases', status: 200, body: { items: [database] } }),
      apiResponse({ method: 'get', path: '/databases/{databaseId}', status: 200, body: database }),
      apiResponse({ method: 'get', path: '/storages', status: 200, body: { items: [storage] } }),
      apiResponse({ method: 'get', path: '/storages/{storageId}', status: 200, body: storage }),
      apiResponse({
        method: 'get',
        path: '/backup-configs',
        status: 200,
        body: { items: [config] },
      }),
      apiResponse({
        method: 'get',
        path: '/backup-configs/{databaseId}',
        status: 200,
        body: config,
      }),
      apiResponse({
        method: 'get',
        path: '/logical-backups',
        status: 200,
        body: { totalUsageMb: 0, items: [] },
      }),
      apiResponse({
        method: 'get',
        path: '/healthcheck-configs/{databaseId}',
        status: 200,
        body: {
          databaseId: database.id,
          isHealthcheckEnabled: false,
          isNotificationEnabled: false,
          intervalMinutes: 1,
          failedAttemptsBeforeUnavailable: 3,
          attemptRetentionDays: 7,
        },
      }),
      apiResponse({
        method: 'get',
        path: '/verification-configs/{databaseId}',
        status: 200,
        body: {
          databaseId: database.id,
          isAutomaticVerificationEnabled: false,
          scheduleType: 'AFTER_BACKUP',
          schedule: { type: 'DAILY', timeOfDay: '00:00' },
          notificationTriggers: [],
        },
      }),
      apiResponse({
        method: 'post',
        path: '/storage-tests',
        status: 200,
        body: { isConnected: true },
      }),
      apiResponse({
        method: 'post',
        path: '/storages',
        status: 201,
        body: {
          ...storage,
          id: 'created-drive',
          name: 'Drive draft',
          settings: { type: 'GOOGLE_DRIVE', clientId: 'public-client' },
        },
      }),
      { method: 'POST', path: 'https://oauth2.googleapis.com/token', status: 200, body: tokens },
    ],
    { scope: 'test' },
  ],
});

const installGoogle = async (context: BrowserContext, isRefused = false) => {
  const authorizationUrls: string[] = [];
  const relayedMessages: unknown[] = [];
  await context.exposeFunction('recordStorageAuthorizationMessage', (message: unknown) =>
    relayedMessages.push(message),
  );
  const tokenHeaders: Record<string, string>[] = [];
  context.on('request', (request) => {
    if (request.url() === 'https://oauth2.googleapis.com/token')
      tokenHeaders.push(request.headers());
  });

  await context.addInitScript(() => {
    const messages: unknown[] = [];
    Object.assign(window, { storageAuthorizationMessages: messages });
    const postMessage = BroadcastChannel.prototype.postMessage;
    BroadcastChannel.prototype.postMessage = function (message) {
      messages.push(message);
      void (
        window as unknown as {
          recordStorageAuthorizationMessage: (message: unknown) => Promise<void>;
        }
      ).recordStorageAuthorizationMessage(message);
      return postMessage.call(this, message);
    };
  });

  await context.route(
    (url) => url.origin === 'https://accounts.google.com' && url.pathname === '/o/oauth2/v2/auth',
    async (route) => {
      expect(route.request().method()).toBe('GET');
      const authorization = new URL(route.request().url());
      authorizationUrls.push(authorization.href);
      const state = authorization.searchParams.get('state')!;
      expect(state).toMatch(/^[a-f0-9]{64}$/);
      const callback = new URL(authorization.searchParams.get('redirect_uri')!);
      callback.searchParams.set('state', state);
      callback.searchParams.set(
        isRefused ? 'error' : 'code',
        isRefused ? 'access_denied' : 'synthetic-code',
      );

      await route.fulfill({
        contentType: 'text/html',
        body: `<button id="continue">Continue</button><script>
      window.opener = null;
      document.querySelector('#continue').onclick = () => location.href = ${JSON.stringify(callback.href)};
    </script>`,
      });
    },
  );

  return { authorizationUrls, tokenHeaders, relayedMessages };
};

const openNestedEditor = async (page: Page) => {
  await page.addInitScript(() => {
    localStorage.setItem('databasus_user_token', 'synthetic-session');
    localStorage.setItem('databasus_user_id', 'owner-id');
    localStorage.setItem('databasus-locale', 'en');
    localStorage.setItem('databasus-theme', 'light');
  });

  await page.goto('/');
  await page.getByText('Config', { exact: true }).click();
  await page.getByText('Backup config', { exact: true }).locator('..').locator('img').click();
  await page.getByPlaceholder('0 2 * * *', { exact: true }).fill('15 3 * * *');
  const storagePicker = page.getByRole('combobox', { name: 'Storage', exact: true });
  await storagePicker.click();
  await page.getByText('Create new storage', { exact: true }).click();

  const editor = page.getByRole('dialog', { name: 'Add storage' });
  await expect(editor.getByRole('button', { name: 'Close', exact: true })).toBeFocused();

  await editor.getByRole('combobox', { name: 'Type', exact: true }).focus();
  await page.keyboard.press('ArrowDown');
  await page.getByRole('option', { name: 'Google Drive', exact: true }).click();
  await editor.getByPlaceholder('My Storage', { exact: true }).fill('Drive draft');
  await editor.getByLabel('Client ID', { exact: true }).fill('public-client');
  await editor.getByLabel('Client Secret', { exact: true }).fill(secret);

  return { editor, storagePicker };
};

const assertContainment = async (page: Page, google: Awaited<ReturnType<typeof installGoogle>>) => {
  const browserState = await page.evaluate(() => ({
    local: { ...localStorage },
    session: { ...sessionStorage },
    messages: (window as unknown as { storageAuthorizationMessages: unknown[] })
      .storageAuthorizationMessages,
    url: location.href,
  }));

  const retained = JSON.stringify({
    browserState,
    authorizationUrls: google.authorizationUrls,
    messages: google.relayedMessages,
  });

  for (const credential of [secret, tokens.access_token, tokens.refresh_token, 'Drive draft']) {
    expect(retained).not.toContain(credential);
  }
};

test('authorizes the nested editor without opener and preserves the surrounding draft until explicit save', async ({
  page,
  context,
  boundary,
}) => {
  const google = await installGoogle(context);
  const { editor, storagePicker } = await openNestedEditor(page);
  const workspace = await page
    .getByRole('combobox', { name: 'Selected workspace', exact: true })
    .textContent();

  const popupPromise = page.waitForEvent('popup');
  await editor.getByRole('button', { name: 'Authorize', exact: true }).click();
  const popup = await popupPromise;
  await expect(popup.getByRole('button', { name: 'Continue' })).toBeVisible();
  expect(await popup.evaluate(() => window.opener)).toBeNull();
  const closed = popup.waitForEvent('close');
  await popup.getByRole('button', { name: 'Continue' }).click();
  await expect(editor.getByText('Authorized', { exact: true })).toBeVisible();
  await closed;

  expect(google.tokenHeaders).toHaveLength(1);
  await expect
    .poll(
      () =>
        google.relayedMessages.filter((message) => (message as { type: string }).type === 'code')
          .length,
    )
    .toBe(1);
  expect(google.tokenHeaders[0]).not.toHaveProperty('authorization');

  const state = new URL(google.authorizationUrls[0]).searchParams.get('state')!;
  await page.evaluate(async (state) => {
    const channel = new BroadcastChannel(`databasus-storage-authorization:${state}`);
    channel.postMessage({ type: 'code', state, code: 'duplicate-code' });
    channel.close();
  }, state);

  await expect(editor.getByText('Authorized', { exact: true })).toBeVisible();
  expect(
    boundary.requests.filter((request) => request.method !== 'GET' && request.path !== '/token'),
  ).toHaveLength(0);
  expect(boundary.requests.filter((request) => request.path === '/token')).toHaveLength(1);
  expect(boundary.requests.find((request) => request.path === '/token')!.body).toMatchObject({
    client_secret: secret,
    code: 'synthetic-code',
    redirect_uri: `${new URL(page.url()).origin}/storages/google-oauth`,
  });

  await assertContainment(page, google);
  await expect(page.getByPlaceholder('0 2 * * *')).toHaveValue('15 3 * * *');
  await expect(storagePicker).toHaveText('Existing archive');
  expect(
    await page.getByRole('combobox', { name: 'Selected workspace', exact: true }).textContent(),
  ).toBe(workspace);
  await expect(editor.getByPlaceholder('My Storage')).toHaveValue('Drive draft');

  await editor.getByRole('button', { name: 'Test connection' }).click();
  await editor.getByRole('button', { name: 'Save', exact: true }).click();
  await expect(editor).not.toBeVisible();
  await expect(page.getByPlaceholder('0 2 * * *')).toHaveValue('15 3 * * *');
  expect(
    await page.getByRole('combobox', { name: 'Selected workspace', exact: true }).textContent(),
  ).toBe(workspace);

  expect(
    boundary.requests.filter(
      (request) => request.method === 'POST' && request.path === '/storages',
    ),
  ).toHaveLength(1);
  expect(boundary.requests.filter((request) => request.method === 'PUT')).toHaveLength(0);
  await expect(storagePicker).toHaveText('Drive draft');
});

test('keeps the prior storage selection and config draft after canceling an authorized nested editor', async ({
  page,
  context,
  boundary,
}) => {
  const google = await installGoogle(context);
  const { editor } = await openNestedEditor(page);

  const popupPromise = page.waitForEvent('popup');
  await editor.getByRole('button', { name: 'Authorize', exact: true }).click();
  const popup = await popupPromise;
  await popup.getByRole('button', { name: 'Continue' }).click();
  await expect(editor.getByText('Authorized', { exact: true })).toBeVisible();

  await editor.getByRole('button', { name: 'Close', exact: true }).click();
  await expect(page.getByPlaceholder('0 2 * * *')).toHaveValue('15 3 * * *');
  await expect(page.getByRole('combobox', { name: 'Storage', exact: true })).toBeVisible();
  await expect(page.getByRole('combobox', { name: 'Storage', exact: true })).toHaveText(
    'Existing archive',
  );

  expect(
    boundary.requests.filter((request) => request.method !== 'GET' && request.path !== '/token'),
  ).toHaveLength(0);
  await assertContainment(page, google);
});

for (const failure of ['refusal', 'missing refresh', 'canceled', 'replaced', 'blocked popup']) {
  test(`recovers from ${failure} without saving or losing the draft`, async ({
    page,
    context,
    boundary,
  }) => {
    const google = await installGoogle(context, failure === 'refusal');
    if (failure === 'missing refresh')
      await context.route('https://oauth2.googleapis.com/token', (route) =>
        route.fulfill({
          contentType: 'application/json',
          body: JSON.stringify({ access_token: 'synthetic-access-only' }),
        }),
      );
    const { editor } = await openNestedEditor(page);

    if (failure === 'blocked popup') {
      await page.evaluate(() => {
        window.open = () => null;
      });
      await editor.getByRole('button', { name: 'Authorize', exact: true }).click();
      await expect(editor.getByRole('alert')).toContainText('blocked');
    } else {
      const popupPromise = page.waitForEvent('popup');
      await editor.getByRole('button', { name: 'Authorize', exact: true }).click();
      const popup = await popupPromise;
      await expect(popup.getByRole('button', { name: 'Continue' })).toBeVisible();
      if (failure === 'canceled')
        await editor.getByRole('button', { name: 'Cancel authorization' }).click();
      if (failure === 'replaced')
        await editor.getByLabel('Client ID', { exact: true }).fill('replacement-client');
      await popup.getByRole('button', { name: 'Continue' }).click();
      if (failure === 'canceled' || failure === 'replaced') {
        await expect(popup.getByRole('alert')).toContainText('start authorization again', {
          timeout: 7000,
        });
        expect(new URL(popup.url()).search).toBe('');
        expect(google.tokenHeaders).toHaveLength(0);
        await popup.close();
      } else await expect(editor.getByRole('alert')).toBeVisible();
    }

    await expect(editor.getByPlaceholder('My Storage')).toHaveValue('Drive draft');
    await expect(editor.getByRole('button', { name: 'Test connection' })).toBeDisabled();
    expect(
      boundary.requests.filter((request) => request.method !== 'GET' && request.path !== '/token'),
    ).toHaveLength(0);
    await assertContainment(page, google);
  });
}

test('clears an orphan callback and stops waiting after its editor was closed', async ({
  context,
  boundary,
}) => {
  await installGoogle(context);
  const callback = await context.newPage();
  await callback.goto(`/storages/google-oauth?state=${'b'.repeat(64)}&code=orphan-code`);

  await expect(callback.getByRole('alert')).toContainText('start authorization again', {
    timeout: 7000,
  });
  expect(new URL(callback.url()).search).toBe('');
  expect(boundary.requests).toHaveLength(0);
  await callback.close();
});

test('keeps authorization controls accessible at a narrow width', async ({ page, context }) => {
  await installGoogle(context);
  const { editor } = await openNestedEditor(page);

  await page.setViewportSize({ width: 320, height: 900 });
  await editor.evaluate(async (element) => {
    await Promise.allSettled(
      element.getAnimations({ subtree: true }).map((animation) => animation.finished),
    );
  });

  const accessibility = await new AxeBuilder({ page })
    .disableRules(['color-contrast'])
    .include('[role="dialog"]')
    .withTags(['wcag2a', 'wcag2aa'])
    .analyze();
  expect(accessibility.violations).toEqual([]);

  const width = await editor.evaluate((element) => ({
    width: element.scrollWidth,
    viewport: innerWidth,
  }));
  expect(width.width).toBeLessThanOrEqual(width.viewport);
});
