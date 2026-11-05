import { expect, type Page } from '@playwright/test';

import type { components } from '../../src/shared/api/schema.gen';
import { applicationFixtures, workspace } from '../applicationFixtures';
import { test } from '../browserTest';
import { apiResponse } from '../HttpFixture';
import { openAccount } from '../openAccount';
import { openWorkspaceScreen } from '../openWorkspaceScreen';
import { assertSingleScrollOwner } from '../resourceScrollScenarios';

type Database = components['schemas']['Database'];
type DatabaseCreate = components['schemas']['DatabaseCreate'];
type DatabaseUpdate = components['schemas']['DatabaseUpdate'];

const TYPED_PASSWORD = 'typed-password-value';
const TYPED_SSH_PASSWORD = 'typed-ssh-password-value';

const storage: components['schemas']['Storage'] = {
  id: 'archive-id',
  workspaceId: workspace.id,
  name: 'Archive',
  settings: { type: 'LOCAL' },
};

const savedDatabase: Database = {
  id: 'orders-id',
  workspaceId: workspace.id,
  name: 'Orders',
  notifierIds: [],
  settings: {
    type: 'POSTGRES_LOGICAL',
    host: 'postgres.example.com',
    port: 5432,
    username: 'backup',
    database: 'orders',
    sslMode: 'require',
    sshTunnel: { isEnabled: false },
    includeSchemas: [],
    excludeTables: [],
    excludeTableData: [],
    cpuCount: 1,
    isSkipUserMappings: false,
  },
};

const backupConfig: components['schemas']['BackupConfig'] = {
  cleanup: { pendingFileCount: 0, failedFileCount: 0 },
  databaseId: savedDatabase.id,
  storageId: storage.id,
  isBackupsEnabled: false,
  encryption: 'NONE',
  settings: {
    type: 'LOGICAL',
    schedule: { type: 'HOURLY' },
    retention: {
      policy: 'COUNT',
      count: 100,
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
      apiResponse({ method: 'get', path: '/storages', status: 200, body: { items: [storage] } }),
      apiResponse({ method: 'get', path: '/backup-configs', status: 200, body: { items: [] } }),
      apiResponse({
        method: 'get',
        path: '/backup-configs/{databaseId}',
        status: 200,
        body: backupConfig,
      }),
      apiResponse({
        method: 'put',
        path: '/backup-configs/{databaseId}',
        status: 200,
        body: backupConfig,
      }),
      apiResponse({
        method: 'post',
        path: '/database-tests',
        status: 200,
        body: { isConnected: true },
      }),
      apiResponse({
        method: 'post',
        path: '/limited-user-checks',
        status: 200,
        body: { isAdvisable: false, privileges: [] },
      }),
      apiResponse({
        method: 'get',
        path: '/healthcheck-configs/{databaseId}',
        status: 200,
        body: {
          databaseId: savedDatabase.id,
          isHealthcheckEnabled: false,
          isNotificationEnabled: false,
          intervalMinutes: 1,
          failedAttemptsBeforeUnavailable: 3,
          attemptRetentionDays: 7,
        },
      }),
      apiResponse({
        method: 'get',
        path: '/healthcheck-attempts',
        status: 200,
        body: { items: [] },
      }),
      apiResponse({ method: 'get', path: '/logical-backups', status: 200, body: { items: [] } }),
      apiResponse({
        method: 'post',
        path: '/logical-backups',
        status: 202,
        body: {
          id: 'backup-id',
          databaseId: savedDatabase.id,
          storageId: storage.id,
          status: 'IN_PROGRESS',
          sizeMb: 0,
          rawDatabaseSizeMb: 0,
          durationMs: 0,
          encryption: 'NONE',
          verificationStatus: 'NOT_VERIFIED',
          createdAt: '2026-11-01T10:00:00Z',
        },
      }),
      apiResponse({
        method: 'get',
        path: '/verification-configs/{databaseId}',
        status: 200,
        body: {
          databaseId: savedDatabase.id,
          isAutomaticVerificationEnabled: false,
          scheduleType: 'AFTER_BACKUP',
          schedule: { type: 'DAILY' },
          notificationTriggers: [],
        },
      }),
      apiResponse({ method: 'get', path: '/storages/{storageId}', status: 200, body: storage }),
    ],
    { scope: 'test' },
  ],
});

// Answers as the server does: it stores what a request sends and never returns a secret.
const serveDatabases = async (page: Page, initialDatabases: Database[]) => {
  const storedDatabases = new Map(initialDatabases.map((database) => [database.id, database]));
  const sentBodies: string[] = [];

  const withoutSecrets = (settings: Database['settings']): Database['settings'] => ({
    ...settings,
    password: undefined,
    sshTunnel: { ...settings.sshTunnel, password: undefined },
  });

  await page.route(/\/api\/v1\/databases(\?.*)?$/, async (route) => {
    if (route.request().method() === 'GET') {
      await route.fulfill({ json: { items: [...storedDatabases.values()] } });
      return;
    }

    sentBodies.push(route.request().postData() ?? '');

    const created = route.request().postDataJSON() as DatabaseCreate;
    const createdDatabase: Database = {
      ...created,
      id: savedDatabase.id,
      notifierIds: created.notifierIds ?? [],
      settings: withoutSecrets(created.settings),
    };
    storedDatabases.set(createdDatabase.id, createdDatabase);

    await route.fulfill({ status: 201, json: createdDatabase });
  });

  await page.route(/\/api\/v1\/databases\/[^/?]+$/, async (route) => {
    const databaseId = new URL(route.request().url()).pathname.split('/').pop()!;
    const storedDatabase = storedDatabases.get(databaseId)!;

    if (route.request().method() === 'GET') {
      await route.fulfill({ json: storedDatabase });
      return;
    }

    sentBodies.push(route.request().postData() ?? '');

    const update = route.request().postDataJSON() as DatabaseUpdate;
    const updatedDatabase: Database = {
      ...storedDatabase,
      name: update.name,
      settings: withoutSecrets(update.settings),
    };
    storedDatabases.set(databaseId, updatedDatabase);

    await route.fulfill({ json: updatedDatabase });
  });

  return { sentBodies };
};

const openConnectionSection = async (page: Page) => {
  await page.getByText('Config', { exact: true }).click();
  await page
    .getByText('Database settings', { exact: true })
    .locator('..')
    .locator('img[src="/icons/pen-gray.svg"]')
    .click();
};

for (const width of [320, 1280]) {
  test(`scrolls a long connection form in one region when viewport width is ${width}`, async ({
    page,
  }) => {
    await serveDatabases(page, [savedDatabase]);
    await page.setViewportSize({ width, height: 420 });
    await openAccount({ page });
    await openWorkspaceScreen({ page, locale: 'en', tab: 'databases' });

    if (width === 320) await page.getByRole('button', { name: /^Orders/ }).click();

    await openConnectionSection(page);
    await page.getByRole('button', { name: 'Advanced settings' }).click();
    await page.getByLabel('SSH tunnel').check();

    const testConnectionButton = page.getByRole('button', { name: 'Test connection' }).first();
    const detailContainer = testConnectionButton.locator(
      'xpath=ancestor::div[contains(@class,"overflow-y-auto")][1]',
    );

    await expect
      .poll(() =>
        detailContainer.evaluate(
          (element) => element.clientHeight > 0 && element.scrollHeight > element.clientHeight + 1,
        ),
      )
      .toBe(true);

    await testConnectionButton.scrollIntoViewIfNeeded();

    await expect(testConnectionButton).toBeInViewport();
    expect(
      await page.getByLabel('Host', { exact: true }).evaluate((input) => {
        const { left, right } = input.getBoundingClientRect();
        return left >= 0 && right <= window.innerWidth;
      }),
    ).toBe(true);
    await assertSingleScrollOwner(detailContainer);
  });
}

test('creates a database, edits it and reads it again without any secret', async ({ page }) => {
  const { sentBodies } = await serveDatabases(page, []);
  await page.setViewportSize({ width: 1280, height: 900 });
  await openAccount({ page });
  await openWorkspaceScreen({ page, locale: 'en', tab: 'databases' });

  await page.getByRole('button', { name: 'Add database' }).click();
  await page.getByPlaceholder('My favourite DB').fill('Orders');
  await page.getByRole('button', { name: 'Continue' }).click();
  await page.getByRole('button', { name: 'Continue' }).click();
  await page.getByLabel('Host', { exact: true }).fill('postgres.example.com');
  await page.getByLabel('Username', { exact: true }).fill('backup');
  await page.getByLabel('Password', { exact: true }).fill(TYPED_PASSWORD);
  await page.getByLabel('DB name', { exact: true }).fill('orders');
  await page.getByRole('button', { name: 'Advanced settings' }).click();
  await page.getByLabel('SSH tunnel').check();

  await page.getByLabel('SSH host', { exact: true }).fill('bastion.example.com');
  await page.getByLabel('SSH username', { exact: true }).fill('tunnel-user');
  await page.getByLabel('SSH password', { exact: true }).fill(TYPED_SSH_PASSWORD);
  await page.getByRole('button', { name: 'Test connection' }).click();
  await page.getByRole('button', { name: 'Continue' }).click();

  await page.getByRole('dialog').getByRole('combobox', { name: 'Storage' }).click();
  await page.getByTitle('Archive').click();
  await page.getByRole('button', { name: 'Continue' }).click();
  await page.getByRole('button', { name: 'Complete' }).click();

  await expect(page.getByRole('dialog')).toBeHidden();
  expect(sentBodies[0]).toContain(TYPED_PASSWORD);

  await openConnectionSection(page);

  await expect(page.getByLabel('Host', { exact: true })).toHaveValue('postgres.example.com');
  await expect(page.getByLabel('Password', { exact: true })).toHaveValue('');
  await expect(page.getByText('SSH credentials')).toBeVisible();

  await page.getByLabel('Username', { exact: true }).fill('replica_user');
  await page.getByRole('button', { name: 'Test connection' }).click();
  await page.getByRole('button', { name: 'Save', exact: true }).click();

  await expect(page.getByLabel('Host', { exact: true })).toBeHidden();
  await expect(page.getByText('replica_user')).toBeVisible();
  expect(sentBodies[1]).not.toContain(TYPED_PASSWORD);
  expect(sentBodies[1]).not.toContain(TYPED_SSH_PASSWORD);

  await page.reload();
  await page.getByText('Config', { exact: true }).click();

  await expect(page.getByText('replica_user')).toBeVisible();
  await expect(page.getByText(TYPED_PASSWORD)).toHaveCount(0);
  await expect(page.getByText(TYPED_SSH_PASSWORD)).toHaveCount(0);
});
