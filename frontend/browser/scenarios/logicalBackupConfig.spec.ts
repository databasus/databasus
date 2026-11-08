import { expect, type Page } from '@playwright/test';

import type { components } from '../../src/shared/api/schema.gen';
import { DICTIONARIES } from '../../src/shared/i18n/dictionaries';
import { applicationFixtures, workspace } from '../applicationFixtures';
import { assertEnlargedTextReflow } from '../assertEnlargedTextReflow';
import { test } from '../browserTest';
import { apiResponse } from '../HttpFixture';
import { openAccount } from '../openAccount';
import { openWorkspaceScreen } from '../openWorkspaceScreen';
import { assertSingleScrollOwner } from '../resourceScrollScenarios';

type BackupConfig = components['schemas']['BackupConfig'];

const dictionary = DICTIONARIES.ru;

const database: components['schemas']['Database'] = {
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
    sslMode: 'disable',
    sshTunnel: { isEnabled: false },
    includeSchemas: [],
    excludeTables: [],
    excludeTableData: [],
    cpuCount: 1,
    isSkipUserMappings: false,
  },
};

const secondDatabase: components['schemas']['Database'] = {
  ...database,
  id: 'invoices-id',
  name: 'Invoices',
};

const storedConfig: BackupConfig = {
  cleanup: { pendingFileCount: 0, failedFileCount: 0 },
  databaseId: database.id,
  storageId: null,
  isBackupsEnabled: true,
  encryption: 'NONE',
  settings: {
    type: 'LOGICAL',
    schedule: { type: 'DAILY', timeOfDay: '04:00' },
    retention: {
      policy: 'COUNT',
      count: 10,
      period: 'MONTH',
      gfsHours: 0,
      gfsDays: 0,
      gfsWeeks: 0,
      gfsMonths: 0,
      gfsYears: 0,
    },
    isRetryIfFailed: true,
    maxFailedTriesCount: 3,
    notificationTriggers: [],
  },
};

test.use({
  httpFixtures: [
    [
      ...applicationFixtures().filter(
        (fixture) => !['/databases', '/backup-configs', '/storages'].includes(fixture.path),
      ),
      apiResponse({
        method: 'get',
        path: '/databases',
        status: 200,
        body: { items: [database, secondDatabase] },
      }),
      apiResponse({ method: 'get', path: '/storages', status: 200, body: { items: [] } }),
      apiResponse({ method: 'get', path: '/backup-configs', status: 200, body: { items: [] } }),
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
        path: '/healthcheck-attempts',
        status: 200,
        body: { items: [] },
      }),
      apiResponse({
        method: 'get',
        path: '/verification-configs/{databaseId}',
        status: 200,
        body: {
          databaseId: database.id,
          isAutomaticVerificationEnabled: false,
          scheduleType: 'AFTER_BACKUP',
          schedule: { type: 'DAILY' },
          notificationTriggers: [],
        },
      }),
    ],
    { scope: 'test' },
  ],
});

const serveDatabasesAndStoredBackupConfig = async (page: Page) => {
  let currentConfig = storedConfig;

  await page.route(/\/api\/v1\/databases\/[^/?]+$/, (route) => {
    const databaseId = new URL(route.request().url()).pathname.split('/').pop();

    return route.fulfill({ json: databaseId === database.id ? database : secondDatabase });
  });

  await page.route(/\/api\/v1\/backup-configs\/[^/?]+$/, async (route) => {
    if (route.request().method() === 'PUT') {
      currentConfig = {
        ...currentConfig,
        ...(route.request().postDataJSON() as BackupConfig),
        databaseId: database.id,
      };
    }

    await route.fulfill({ json: currentConfig });
  });
};

const selectDatabase = (page: Page, name: RegExp) => page.getByRole('link', { name }).click();

const openConfigTab = (page: Page) =>
  page.getByRole('main').getByText(dictionary.databases.tabs.config, { exact: true }).click();

const openBackupSection = async (page: Page) => {
  const sectionTitle = page.getByText(dictionary.databases.config.sections.backupConfig, {
    exact: true,
  });

  await sectionTitle.locator('xpath=..').locator('img[src="/icons/pen-gray.svg"]').click();
};

for (const width of [320, 1280]) {
  test(`edits retention and advanced settings in Russian and reads them again when viewport width is ${width}`, async ({
    page,
  }) => {
    await serveDatabasesAndStoredBackupConfig(page);
    await page.setViewportSize({ width, height: 420 });
    await openAccount({ page, locale: 'ru' });
    await openWorkspaceScreen({ page, locale: 'ru', tab: 'databases' });

    if (width === 320) await selectDatabase(page, /^Orders/);

    await openConfigTab(page);
    await openBackupSection(page);

    const logicalConfigText = dictionary.backups.logical.config;
    const saveButton = page.getByRole('button', { name: dictionary.common.actions.save });

    await expect(saveButton).toBeDisabled();

    await page.getByLabel(logicalConfigText.retention.count, { exact: true }).fill('25');
    await page.getByRole('button', { name: logicalConfigText.advancedSettings }).click();
    await page.getByRole('switch', { name: logicalConfigText.retryIfFailed }).click();

    const detailContainer = saveButton.locator(
      'xpath=ancestor::div[contains(@class,"overflow-y-auto")][1]',
    );

    await assertSingleScrollOwner(detailContainer);
    await assertEnlargedTextReflow(page);

    await saveButton.click();
    await expect(saveButton).toBeHidden();

    await expect(
      page.getByText(logicalConfigText.retention.countValue.replace('{{count}}', '25')),
    ).toBeVisible();

    if (width === 320) {
      await page.getByRole('button', { name: /^← / }).click();
      await selectDatabase(page, /^Orders/);
      await openConfigTab(page);
    } else {
      await selectDatabase(page, /^Invoices/);
      await expect(page.getByText('Invoices', { exact: true }).last()).toBeVisible();
      await selectDatabase(page, /^Orders/);
    }

    await expect(
      page.getByText(logicalConfigText.retention.countValue.replace('{{count}}', '25')),
    ).toBeVisible();
    await expect(
      page.getByText(logicalConfigText.retryIfFailedShort).locator('xpath=..'),
    ).toContainText(dictionary.common.answers.no);
  });
}
