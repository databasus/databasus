import { expect, type Page } from '@playwright/test';

import type { components } from '../../src/shared/api/schema.gen';
import { DICTIONARIES } from '../../src/shared/i18n/dictionaries';
import { applicationFixtures, workspace } from '../applicationFixtures';
import { test } from '../browserTest';
import { apiResponse } from '../HttpFixture';
import { openAccount } from '../openAccount';
import { openWorkspaceScreen } from '../openWorkspaceScreen';

type VerificationConfig = components['schemas']['VerificationConfig'];

const dictionary = DICTIONARIES.ru;
const SAVED_CRON_EXPRESSION = '0 3 * * 1';

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

const storedConfig: VerificationConfig = {
  databaseId: database.id,
  isAutomaticVerificationEnabled: true,
  scheduleType: 'INTERVAL',
  schedule: { type: 'CRON', cronExpression: '0 2 * * *' },
  notificationTriggers: [],
};

test.use({
  httpFixtures: [
    [
      ...applicationFixtures().filter(
        (fixture) => !['/databases', '/backup-configs', '/storages'].includes(fixture.path),
      ),
      apiResponse({ method: 'get', path: '/databases', status: 200, body: { items: [database] } }),
      apiResponse({ method: 'get', path: '/storages', status: 200, body: { items: [] } }),
      apiResponse({ method: 'get', path: '/backup-configs', status: 200, body: { items: [] } }),
      apiResponse({ method: 'get', path: '/logical-backups', status: 200, body: { items: [] } }),
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
    ],
    { scope: 'test' },
  ],
});

const serveDatabaseConfigScreenRoutes = async (page: Page) => {
  let currentConfig = storedConfig;

  await page.route(/\/api\/v1\/databases\/[^/?]+$/, (route) => route.fulfill({ json: database }));

  await page.route(/\/api\/v1\/backup-configs\/[^/?]+$/, (route) =>
    route.fulfill({
      json: {
        cleanup: { pendingFileCount: 0, failedFileCount: 0 },
        databaseId: database.id,
        storageId: null,
        isBackupsEnabled: false,
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
          isRetryIfFailed: false,
          maxFailedTriesCount: 1,
          notificationTriggers: [],
        },
      },
    }),
  );

  await page.route(/\/api\/v1\/verification-configs\/[^/?]+$/, async (route) => {
    if (route.request().method() === 'PUT') {
      currentConfig = {
        ...(route.request().postDataJSON() as VerificationConfig),
        databaseId: database.id,
      };
    }

    await route.fulfill({ json: currentConfig });
  });
};

const openVerificationSection = async (page: Page) => {
  await page.getByRole('main').getByText(dictionary.databases.tabs.config, { exact: true }).click();

  const sectionTitle = page.getByText(dictionary.databases.config.sections.restoreVerification, {
    exact: true,
  });

  await sectionTitle.locator('xpath=..').locator('img[src="/icons/pen-gray.svg"]').click();
};

for (const width of [320, 1280]) {
  test(`edits a verification schedule in Russian and reads it again when viewport width is ${width}`, async ({
    page,
  }) => {
    await serveDatabaseConfigScreenRoutes(page);
    await page.setViewportSize({ width, height: 720 });
    await openAccount({ page, locale: 'ru' });
    await openWorkspaceScreen({ page, locale: 'ru', tab: 'databases' });

    if (width === 320) await page.getByRole('button', { name: /^Orders/ }).click();

    await openVerificationSection(page);

    const saveButton = page.getByRole('button', { name: dictionary.common.actions.save });
    const cronInput = page.getByLabel(dictionary.schedules.cronExpression, { exact: true });

    await expect(saveButton).toBeDisabled();

    await cronInput.fill('0 0 31 2 *');
    await expect(page.getByText(dictionary.schedules.invalidCron)).toBeVisible();
    await expect(saveButton).toBeDisabled();

    await cronInput.fill(SAVED_CRON_EXPRESSION);
    await expect(saveButton).toBeEnabled();
    await saveButton.click();
    await expect(saveButton).toBeHidden();

    await expect(page.getByText(SAVED_CRON_EXPRESSION, { exact: true })).toBeVisible();
    await expect(page.getByText(dictionary.schedules.cronExpression)).toBeVisible();
  });
}
