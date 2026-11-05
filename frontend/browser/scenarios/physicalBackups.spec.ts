import AxeBuilder from '@axe-core/playwright';
import { expect, type Locator, type Page } from '@playwright/test';

import type { components } from '../../src/shared/api/schema.gen';
import { DICTIONARIES } from '../../src/shared/i18n/dictionaries';
import { applicationFixtures, workspace } from '../applicationFixtures';
import { assertAccountReflow } from '../assertAccountReflow';
import { test } from '../browserTest';
import { apiResponse } from '../HttpFixture';
import { openAccount } from '../openAccount';
import { openWorkspaceScreen } from '../openWorkspaceScreen';

type PhysicalBackup = components['schemas']['PhysicalBackupListRow'];

const dictionary = DICTIONARIES.en;

// More presses than the dialog has controls, so the focus goes around at least once.
const FOCUS_TRAP_TAB_PRESSES = 12;
const listCopy = dictionary.backups.physical.list;

const database: components['schemas']['Database'] = {
  id: 'orders-id',
  workspaceId: workspace.id,
  name: 'Orders',
  notifierIds: [],
  settings: {
    type: 'POSTGRES_PHYSICAL',
    backupType: 'FULL_INCREMENTAL_WAL_STREAM',
    host: 'postgres.example.com',
    port: 5432,
    username: 'backup',
    sslMode: 'disable',
    sshTunnel: { isEnabled: false },
    version: '18',
  } as components['schemas']['Database']['settings'],
};

const buildBackup = (change: Partial<PhysicalBackup>): PhysicalBackup => ({
  id: 'full-backup',
  type: 'FULL',
  status: 'COMPLETED',
  timelineId: 1,
  startLsn: '0/10000000',
  stopLsn: '0/20000000',
  hasGapBefore: false,
  sizeMb: 2048,
  createdAt: '2026-10-18T10:00:00Z',
  completedAt: '2026-10-18T10:05:00Z',
  ...change,
});

const listedBackups: PhysicalBackup[] = [
  buildBackup({}),
  buildBackup({ id: 'incremental-backup', type: 'INCREMENTAL', sizeMb: 64 }),
  buildBackup({
    id: 'wal-range',
    type: 'WAL',
    fromWalFilename: '000000010000000000000010',
    toWalFilename: '0000000100000000000000FF',
    walSegmentsCount: 240,
    sizeMb: 512,
  }),
  buildBackup({ id: 'running-backup', status: 'IN_PROGRESS', completedAt: undefined }),
  buildBackup({ id: 'failed-backup', status: 'ERROR', failMessage: 'The source went away.' }),
  buildBackup({ id: 'broken-backup', status: 'CHAIN_BROKEN' }),
  buildBackup({ id: 'canceled-backup', status: 'CANCELED' }),
];

test.use({
  httpFixtures: [
    [
      ...applicationFixtures().filter(
        (fixture) => !['/databases', '/backup-configs', '/storages'].includes(fixture.path),
      ),
      apiResponse({ method: 'get', path: '/databases', status: 200, body: { items: [database] } }),
      apiResponse({ method: 'get', path: '/databases/{databaseId}', status: 200, body: database }),
      apiResponse({ method: 'get', path: '/storages', status: 200, body: { items: [] } }),
      apiResponse({ method: 'get', path: '/backup-configs', status: 200, body: { items: [] } }),
      apiResponse({
        method: 'get',
        path: '/physical-backups',
        status: 200,
        body: { items: listedBackups, totalUsageMb: 2048 },
      }),
      apiResponse({
        method: 'get',
        path: '/backup-configs/{databaseId}',
        status: 200,
        body: {
          cleanup: { pendingFileCount: 0, failedFileCount: 0 },
          databaseId: database.id,
          isBackupsEnabled: true,
          storageId: 'storage-id',
          encryption: 'NONE',
          settings: {
            type: 'PHYSICAL',
            fullBackupSchedule: { type: 'HOURLY' },
            incrementalBackupSchedule: null,
            retention: {
              policy: 'FULL_BACKUPS',
              chainsCount: 0,
              fullBackups: {
                policy: 'LAST_N',
                count: 5,
                gfsHours: 0,
                gfsDays: 0,
                gfsWeeks: 0,
                gfsMonths: 0,
                gfsYears: 0,
              },
            },
            walLagThresholdBytes: 0,
            notificationTriggers: [],
          },
        },
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
        method: 'post',
        path: '/physical-restore-tokens',
        status: 201,
        body: { token: 'restore-token', expiresAt: '2026-10-18T13:00:00Z' },
      }),
    ],
    { scope: 'test' },
  ],
});

// The dialog takes the focus at the end of its opening animation, and its keys work only after that.
const waitForFocusInDialog = (dialog: Locator) => expect(dialog.locator('*:focus')).toHaveCount(1);

const openPhysicalList = async (page: Page, theme: string) => {
  await openAccount({ page, locale: 'en', theme });
  await openWorkspaceScreen({ page, locale: 'en', tab: 'databases' });
};

for (const theme of ['light', 'dark']) {
  test(`keeps the physical backup list accessible with every badge in the ${theme} theme`, async ({
    page,
  }) => {
    await page.setViewportSize({ width: 1280, height: 900 });
    await openPhysicalList(page, theme);

    const table = page.locator('.ant-table-wrapper');
    await expect(
      table.getByText(dictionary.backups.physical.types.full, { exact: true }),
    ).toHaveCount(5);
    await expect(
      table.getByText(dictionary.backups.physical.types.incremental, { exact: true }),
    ).toBeVisible();
    await expect(
      table.getByText(dictionary.backups.physical.types.wal, { exact: true }),
    ).toBeVisible();

    const accessibility = await new AxeBuilder({ page })
      .disableRules(['color-contrast'])
      .include('.ant-table-wrapper')
      .withTags(['wcag2a', 'wcag2aa'])
      .analyze();

    expect(accessibility.violations).toEqual([]);
  });
}

test('shows the physical backup list as cards that fit a narrow viewport', async ({ page }) => {
  await page.setViewportSize({ width: 320, height: 800 });
  await openPhysicalList(page, 'light');
  await page.getByRole('button', { name: /^Orders/ }).click();

  const listTitle = page.getByRole('heading', { name: dictionary.backups.list.title });
  await expect(page.getByRole('button', { name: listCopy.backUpNow })).toBeVisible();
  await expect(page.locator('.ant-table-wrapper')).toBeHidden();

  const listRegion = listTitle.locator('xpath=../..');
  await expect(
    listRegion.getByText('64 MB', { exact: true }).filter({ visible: true }),
  ).toBeVisible();
  await expect(
    listRegion
      .getByRole('button', { name: listCopy.restoreFromThisBackup })
      .filter({ visible: true })
      .first(),
  ).toBeVisible();

  await assertAccountReflow(listRegion);
});

test('keeps focus inside the restore dialog and returns it to the row action on close', async ({
  page,
  context,
}) => {
  await context.grantPermissions(['clipboard-read', 'clipboard-write']);
  await page.setViewportSize({ width: 1280, height: 900 });
  await openPhysicalList(page, 'light');

  const restoreAction = page
    .locator('.ant-table-wrapper')
    .getByRole('button', { name: listCopy.restoreFromThisBackup })
    .first();
  await restoreAction.focus();
  await page.keyboard.press('Enter');

  const dialog = page.getByRole('dialog');
  await expect(dialog.locator('pre')).toContainText('restore-token');
  await waitForFocusInDialog(dialog);

  for (let tabPress = 0; tabPress < FOCUS_TRAP_TAB_PRESSES; tabPress++) {
    await page.keyboard.press('Tab');

    expect(await dialog.evaluate((element) => element.contains(document.activeElement))).toBe(true);
  }

  const copyAction = dialog.getByRole('button', { name: dictionary.common.actions.copy });
  await copyAction.focus();
  await page.keyboard.press('Enter');
  await expect(
    dialog.getByRole('button', { name: dictionary.backups.physical.restore.copied }),
  ).toBeVisible();

  await page.keyboard.press('Escape');

  await expect(dialog).toBeHidden();
  await expect(restoreAction).toBeFocused();
});

test('returns focus to the restore button when the point-in-time dialog closes', async ({
  page,
}) => {
  await page.setViewportSize({ width: 1280, height: 900 });
  await openPhysicalList(page, 'light');

  const restoreButton = page.getByRole('button', { name: listCopy.restore, exact: true });
  await restoreButton.focus();
  await page.keyboard.press('Enter');

  const dialog = page.getByRole('dialog');
  await expect(dialog.locator('pre')).toContainText('restore-token');
  await waitForFocusInDialog(dialog);

  await page.keyboard.press('Escape');

  await expect(dialog).toBeHidden();
  await expect(restoreButton).toBeFocused();
});

test('returns focus to the status badge when the error details dialog closes', async ({ page }) => {
  await page.setViewportSize({ width: 1280, height: 900 });
  await openPhysicalList(page, 'light');

  const failedStatus = page
    .locator('.ant-table-wrapper')
    .getByRole('button', { name: dictionary.status.physicalBackup.error });
  await failedStatus.focus();
  await page.keyboard.press('Enter');

  const dialog = page.getByRole('dialog');
  await expect(dialog.getByText('The source went away.')).toBeVisible();
  await waitForFocusInDialog(dialog);

  await page.keyboard.press('Escape');

  await expect(dialog).toBeHidden();
  await expect(failedStatus).toBeFocused();
});

test('keeps the scroll position and the loaded rows after a manual backup', async ({ page }) => {
  const buildPage = (firstRow: number) =>
    Array.from({ length: 30 }, (_, rowIndex) =>
      buildBackup({ id: `backup-${firstRow + rowIndex}`, sizeMb: firstRow + rowIndex + 1 }),
    );
  let startRequests = 0;
  await page.route(/\/api\/v1\/physical-backups(\?.*)?$/, (route) => {
    if (route.request().method() === 'POST') {
      startRequests += 1;

      return route.fulfill({ status: 202 });
    }

    const isNextPage = new URL(route.request().url()).searchParams.has('after');

    return route.fulfill({
      json: isNextPage
        ? { items: buildPage(30), totalUsageMb: 2048 }
        : { items: buildPage(0), totalUsageMb: 2048, nextCursor: 'first-page-cursor' },
    });
  });
  await page.setViewportSize({ width: 1280, height: 700 });
  await openPhysicalList(page, 'light');

  const table = page.locator('.ant-table-wrapper');
  const rows = table.locator('tr[data-row-key]');
  await expect(rows).toHaveCount(30);

  const scrollListTo = (scrollTop: number | null) =>
    table.evaluate((tableElement, requestedTop) => {
      let scrollOwner: Element | null = tableElement;

      while (scrollOwner && scrollOwner.scrollHeight <= scrollOwner.clientHeight) {
        scrollOwner = scrollOwner.parentElement;
      }

      if (requestedTop !== null) scrollOwner!.scrollTop = requestedTop;

      return scrollOwner!.scrollTop;
    }, scrollTop);

  await scrollListTo(100_000);
  await expect(rows).toHaveCount(60);

  const positionBeforeStart = await scrollListTo(300);
  expect(positionBeforeStart).toBeGreaterThan(0);

  const laterPoll = page.waitForResponse(
    (response) =>
      response.url().includes('/physical-backups') && response.request().method() === 'GET',
  );
  // A real click would scroll the button into view, and this check is about the list's own effect.
  await page.getByRole('button', { name: listCopy.backUpNow }).dispatchEvent('click');
  await expect.poll(() => startRequests).toBe(1);
  await laterPoll;
  await page.waitForResponse(
    (response) =>
      response.url().includes('/physical-backups') && response.request().method() === 'GET',
  );

  await expect(rows).toHaveCount(60);
  expect(
    await table.evaluate((tableElement) => tableElement.querySelector('.ant-spin-spinning')),
  ).toBeNull();
  expect(await scrollListTo(null)).toBe(positionBeforeStart);
});
