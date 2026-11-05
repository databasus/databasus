import { type Locator, type Page, expect } from '@playwright/test';

import type { components } from '../../src/shared/api/schema.gen';
import { DICTIONARIES } from '../../src/shared/i18n/dictionaries';
import { applicationFixtures, workspace } from '../applicationFixtures';
import { test } from '../browserTest';
import { apiResponse } from '../HttpFixture';
import { openAccount } from '../openAccount';
import { openWorkspaceScreen } from '../openWorkspaceScreen';
import { assertSingleScrollOwner } from '../resourceScrollScenarios';

const FIRST_PAGE_SIZE = 40;
const SECOND_PAGE_SIZE = 20;
const SECOND_PAGE_CURSOR = 'second-page';

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

const config: components['schemas']['BackupConfig'] = {
  cleanup: { pendingFileCount: 0, failedFileCount: 0 },
  databaseId: database.id,
  storageId: null,
  isBackupsEnabled: true,
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

// Each backup gets a distinct size, so its size text identifies its row on both layouts.
const buildBackup = (index: number): components['schemas']['LogicalBackup'] => ({
  id: `backup-${index}`,
  databaseId: database.id,
  storageId: 'storage-id',
  status: 'COMPLETED',
  sizeMb: 101 + index,
  rawDatabaseSizeMb: 1,
  durationMs: 1000,
  encryption: 'NONE',
  verificationStatus: 'NOT_VERIFIED',
  createdAt: new Date(Date.UTC(2026, 9, 30, 12) - index * 60_000).toISOString(),
});

test.use({
  httpFixtures: [
    [
      ...applicationFixtures().filter(
        (fixture) => !['/databases', '/backup-configs'].includes(fixture.path),
      ),
      apiResponse({ method: 'get', path: '/databases', status: 200, body: { items: [database] } }),
      apiResponse({ method: 'get', path: '/databases/{databaseId}', status: 200, body: database }),
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
      apiResponse({ method: 'get', path: '/logical-backups', status: 200, body: { items: [] } }),
      apiResponse({
        method: 'get',
        path: '/verification-configs/{databaseId}',
        status: 200,
        body: {
          databaseId: database.id,
          isAutomaticVerificationEnabled: false,
          scheduleType: 'AFTER_BACKUP',
          schedule: { type: 'HOURLY' },
          notificationTriggers: [],
        },
      }),
      apiResponse({
        method: 'post',
        path: '/databases/{databaseId}/copy',
        status: 201,
        body: { ...database, id: 'orders-copy-id', name: 'Orders copy' },
      }),
    ],
    { scope: 'test' },
  ],
});

for (const width of [320, 1280]) {
  test(`scrolls the database details in one region and loads further backups when viewport width is ${width}`, async ({
    page,
  }) => {
    const backupPageCursors: (string | null)[] = [];
    // The shared fixtures answer one path with one body, while this scenario needs a cursor page.
    await page.route('**/api/v1/logical-backups?**', async (route) => {
      const cursor = new URL(route.request().url()).searchParams.get('after');
      backupPageCursors.push(cursor);
      const page =
        cursor === SECOND_PAGE_CURSOR
          ? {
              items: Array.from({ length: SECOND_PAGE_SIZE }, (_, index) =>
                buildBackup(FIRST_PAGE_SIZE + index),
              ),
            }
          : {
              items: Array.from({ length: FIRST_PAGE_SIZE }, (_, index) => buildBackup(index)),
              nextCursor: SECOND_PAGE_CURSOR,
            };

      await route.fulfill({ json: page });
    });

    await page.setViewportSize({ width, height: width === 320 ? 560 : 360 });
    await openAccount({ page });
    await openWorkspaceScreen({ page, locale: 'en', tab: 'databases' });

    if (width === 320) await page.getByRole('button', { name: /^Orders/ }).click();

    const lastFirstPageBackup = page
      .getByText(`${101 + FIRST_PAGE_SIZE - 1} MB`, { exact: true })
      .first();
    await expect(lastFirstPageBackup).toBeAttached();
    const detailContainer = lastFirstPageBackup.locator(
      'xpath=ancestor::div[contains(@class,"overflow-y-auto")][1]',
    );

    await expect
      .poll(() =>
        detailContainer.evaluate(
          (element) => element.clientHeight > 0 && element.scrollHeight > element.clientHeight + 1,
        ),
      )
      .toBe(true);
    await expect(lastFirstPageBackup).not.toBeInViewport();

    await detailContainer.evaluate((element) => element.scrollTo(0, element.scrollHeight));

    await expect(
      page.getByText(`${101 + FIRST_PAGE_SIZE + SECOND_PAGE_SIZE - 1} MB`, { exact: true }).first(),
    ).toBeAttached();
    expect(backupPageCursors).toContain(SECOND_PAGE_CURSOR);
    await expect
      .poll(() => detailContainer.evaluate((element) => element.scrollTop))
      .toBeGreaterThan(0);
    await assertSingleScrollOwner(detailContainer);
  });
}

const openDatabaseSettings = async (page: Page, theme: 'light' | 'dark') => {
  await page.setViewportSize({ width: 1280, height: 800 });
  await openAccount({ page, theme });
  await openWorkspaceScreen({ page, locale: 'en', tab: 'databases' });
  await page.getByRole('tab', { name: 'Config' }).click();
};

for (const theme of ['light', 'dark'] as const) {
  test(`shows white available status text in the ${theme} theme`, async ({ page }) => {
    await page.route('**/api/v1/databases?**', (route) =>
      route.fulfill({ json: { items: [{ ...database, healthStatus: 'AVAILABLE' }] } }),
    );
    await openAccount({ page, theme });
    await openWorkspaceScreen({ page, locale: 'en', tab: 'databases' });

    const databaseCard = page.getByRole('button', { name: /^Orders/ });
    await expect(databaseCard).toBeVisible();
    await expect(
      databaseCard.getByText(DICTIONARIES.en.status.health.available, { exact: true }),
    ).toHaveCSS('color', 'rgb(255, 255, 255)');
  });
}

test('shows the copy notice on the dark surface and removes it on its own', async ({ page }) => {
  await openDatabaseSettings(page, 'dark');

  await page.getByRole('button', { name: 'Copy', exact: true }).click();
  await page.getByRole('dialog').getByRole('button', { name: 'Copy', exact: true }).click();

  const notice = page.locator('.ant-notification-notice');
  await expect(notice.getByText('Database copied successfully!')).toBeVisible();

  const noticeBackground = await notice.evaluate(
    (element) => getComputedStyle(element).backgroundColor,
  );
  const [red, green, blue] = (noticeBackground.match(/\d+/g) ?? []).map(Number);
  expect(Math.max(red, green, blue)).toBeLessThan(128);

  // The notice stays for AntD's 4.5 seconds plus its leave animation, longer than the default wait.
  await expect(notice).toHaveCount(0, { timeout: 15_000 });
});

const getTopLeftRadius = (card: Locator) =>
  card.evaluate((element) => getComputedStyle(element).borderTopLeftRadius);

const getCardOfHeading = (page: Page, name: string) =>
  page
    .getByRole('heading', { name, level: 2 })
    .locator('xpath=ancestor::div[contains(@class, "rounded-md")][1]');

test('squares the top-left corner of the backups card only when it sits directly under the tabs', async ({
  page,
}) => {
  await page.setViewportSize({ width: 1280, height: 800 });
  await openAccount({ page, theme: 'light' });
  await openWorkspaceScreen({ page, locale: 'en', tab: 'databases' });

  const backupsCard = getCardOfHeading(page, 'Backups');

  await expect.poll(() => getTopLeftRadius(backupsCard)).toBe('0px');
});

test('squares the top-left corner of the healthcheck card when it sits above the backups card', async ({
  page,
}) => {
  await page.route('**/api/v1/healthcheck-configs/*', (route) =>
    route.fulfill({
      json: {
        databaseId: database.id,
        isHealthcheckEnabled: true,
        isNotificationEnabled: false,
        intervalMinutes: 1,
        failedAttemptsBeforeUnavailable: 3,
        attemptRetentionDays: 7,
      },
    }),
  );
  await page.setViewportSize({ width: 1280, height: 800 });
  await openAccount({ page, theme: 'light' });
  await openWorkspaceScreen({ page, locale: 'en', tab: 'databases' });

  const healthcheckCard = getCardOfHeading(page, 'Healthcheck attempts');
  const backupsCard = getCardOfHeading(page, 'Backups');

  await expect.poll(() => getTopLeftRadius(healthcheckCard)).toBe('0px');
  expect(await getTopLeftRadius(backupsCard)).not.toBe('0px');
});

test('shows a focus indicator when tabbing to a tab and an edit icon', async ({ page }) => {
  await openDatabaseSettings(page, 'light');

  const configTab = page.getByRole('tab', { name: /Config$/ });
  await page.getByRole('button', { name: /^Orders/ }).focus();
  await expect
    .poll(async () => {
      await page.keyboard.press('Tab');

      return configTab.evaluate((element) => element === document.activeElement);
    })
    .toBe(true);
  await expect(configTab.locator('xpath=..')).toHaveCSS('outline-style', 'solid');

  const editIcon = page.getByRole('button', { name: 'Edit Orders' });
  await expect
    .poll(async () => {
      await page.keyboard.press('Tab');

      return editIcon.evaluate((element) => element === document.activeElement);
    })
    .toBe(true);
  await expect(editIcon).toHaveCSS('outline-style', 'solid');

  await page.keyboard.press('Enter');

  await expect(page.getByRole('textbox', { name: 'Name' })).toBeVisible();
});
