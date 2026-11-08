import AxeBuilder from '@axe-core/playwright';
import { expect, type Locator, type Page } from '@playwright/test';

import type { components } from '../../src/shared/api/schema.gen';
import { DICTIONARIES } from '../../src/shared/i18n/dictionaries';
import { applicationFixtures, workspace } from '../applicationFixtures';
import { assertAccountReflow } from '../assertAccountReflow';
import { assertEnlargedTextReflow } from '../assertEnlargedTextReflow';
import { assertScreenshot } from '../assertScreenshot';
import { test } from '../browserTest';
import { apiResponse } from '../HttpFixture';
import { openAccount } from '../openAccount';
import { openWorkspaceScreen } from '../openWorkspaceScreen';
import { verifyBackupFilters } from '../verifyBackupFilters';
import { visualContexts } from '../visualContexts';

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

    const table = page.getByRole('table');
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
      .include('table')
      .withTags(['wcag2a', 'wcag2aa'])
      .analyze();

    expect(accessibility.violations).toEqual([]);
  });
}

test('shows the physical backup list as cards that fit a narrow viewport', async ({ page }) => {
  await page.setViewportSize({ width: 320, height: 800 });
  await openPhysicalList(page, 'light');
  await page.getByRole('link', { name: /^Orders/ }).click();

  const listTitle = page.getByRole('heading', { name: dictionary.backups.list.title });
  await expect(page.getByRole('button', { name: listCopy.backUpNow })).toBeVisible();
  await expect(page.getByRole('table')).toBeHidden();

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
  await assertEnlargedTextReflow(page);

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
    .getByRole('table')
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
  await assertEnlargedTextReflow(page);

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
  await assertEnlargedTextReflow(page);

  await page.keyboard.press('Escape');

  await expect(dialog).toBeHidden();
  await expect(restoreButton).toBeFocused();
});

for (const context of visualContexts) {
  test(`returns focus to the status badge when the error details dialog closes in ${context.theme} ${context.locale} ${context.width}`, async ({
    page,
  }) => {
    await page.setViewportSize({ width: context.width, height: 900 });
    await openAccount({ page, ...context });
    await openWorkspaceScreen({ page, locale: context.locale, tab: 'databases' });
    if (context.width === 320) await page.getByRole('link', { name: /^Orders/ }).click();

    const failedStatus = page.getByRole('button', {
      name: DICTIONARIES[context.locale].status.physicalBackup.error,
    });
    await failedStatus.focus();
    await page.keyboard.press('Enter');

    const dialog = page.getByRole('dialog');
    await expect(dialog.getByText('The source went away.')).toBeVisible();
    await waitForFocusInDialog(dialog);
    await assertEnlargedTextReflow(page);

    await page.keyboard.press('Escape');

    await expect(dialog).toBeHidden();
    await expect(failedStatus).toBeFocused();
  });
}

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

  const table = page.getByRole('table');
  const rows = table.locator('tbody').getByRole('row');
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
    await table.evaluate(
      (tableElement) => tableElement.closest('[aria-busy]')?.getAttribute('aria-busy') === 'true',
    ),
  ).toBe(false);
  expect(await scrollListTo(null)).toBe(positionBeforeStart);
});

for (const context of visualContexts) {
  test(`keeps physical backup filters and both deletion warnings in ${context.theme} ${context.locale} ${context.width}`, async ({
    page,
  }) => {
    const dictionary = DICTIONARIES[context.locale];
    await page.setViewportSize({ width: context.width, height: 1000 });
    await openAccount({ page, ...context });
    await openWorkspaceScreen({ page, locale: context.locale, tab: 'databases' });
    if (context.width === 320) await page.getByRole('link', { name: /^Orders/ }).click();
    const remove = page
      .getByRole('button', { name: dictionary.backups.list.actions.deleteBackup, exact: true })
      .filter({ visible: true });
    await expect(remove).toHaveCount(2);
    await assertEnlargedTextReflow(page);

    for (const [index, warning] of [
      dictionary.backups.physical.list.deleteFullConfirmation,
      dictionary.backups.physical.list.deleteIncrementalConfirmation,
    ].entries()) {
      await remove.nth(index).click();
      const confirmation = page.getByRole('dialog', {
        name: dictionary.common.confirmation.title,
        exact: true,
      });
      await expect(confirmation).toContainText(warning);
      await assertEnlargedTextReflow(page);

      await page.keyboard.press('Escape');
      await expect(confirmation).toBeHidden();
    }
    await page
      .getByRole('button', { name: dictionary.backups.filters.toggle, exact: true })
      .click();
    await verifyBackupFilters({ page, locale: context.locale, backupKind: 'physical' });
  });

  test(`opens script and manual restore commands for host and Docker in ${context.theme} ${context.locale} ${context.width}`, async ({
    page,
  }) => {
    const dictionary = DICTIONARIES[context.locale];
    const copy = dictionary.backups.physical.restore;
    await page.setViewportSize({ width: context.width, height: 1000 });
    await openAccount({ page, ...context });
    await openWorkspaceScreen({ page, locale: context.locale, tab: 'databases' });
    if (context.width === 320) await page.getByRole('link', { name: /^Orders/ }).click();
    await page
      .getByRole('button', { name: dictionary.backups.physical.list.restore, exact: true })
      .click();
    const restore = page.getByRole('dialog', {
      name: dictionary.backups.physical.list.pointInTimeRestoreTitle,
      exact: true,
    });
    await expect(restore.locator('pre').first()).toContainText('restore-token');
    await assertEnlargedTextReflow(page);

    await restore.getByRole('textbox', { name: copy.targetTime, exact: true }).click();
    const calendar = page.getByRole('dialog', { name: copy.targetTime, exact: true });
    await expect(calendar).toBeVisible();
    const calendarBounds = await calendar.boundingBox();
    expect
      .soft(calendarBounds?.width)
      .toBeLessThanOrEqual(Math.min(400, page.viewportSize()!.width - 16));
    await expect(
      calendar.getByRole('button', { name: dictionary.common.actions.ok }),
    ).toBeVisible();
    await assertEnlargedTextReflow(page);
    if (context.theme === 'light' && context.width === 1280)
      await assertScreenshot(page, 'physical-restore-target-calendar.png');
    await page.keyboard.press('Escape');
    await expect(restore).toBeVisible();
    for (const method of ['manual', 'script'] as const) {
      const methodChoice = restore.getByRole('radio', { name: copy.methods[method], exact: true });
      await methodChoice.focus();
      await methodChoice.press('Space');
      await expect(methodChoice).toBeChecked();
      for (const environment of [copy.environments.host, 'Docker']) {
        await restore.getByRole('tab', { name: environment, exact: true }).click();
        await assertEnlargedTextReflow(page);
      }
    }
  });
}

test.describe('regional point-in-time calendar', () => {
  test.use({ timezoneId: 'America/New_York' });

  test('submits the chosen UTC target when date and time are selected only by keyboard', async ({
    page,
    boundary,
  }) => {
    const copy = dictionary.backups.physical.restore;
    await openPhysicalList(page, 'light');
    const invoker = page.getByRole('button', {
      name: dictionary.backups.physical.list.restore,
      exact: true,
    });

    await invoker.focus();
    await page.keyboard.press('Enter');

    const restore = page.getByRole('dialog', {
      name: dictionary.backups.physical.list.pointInTimeRestoreTitle,
      exact: true,
    });

    await expect(restore.locator('pre').first()).toContainText('restore-token');
    await waitForFocusInDialog(restore);

    const target = restore.getByRole('textbox', { name: copy.targetTime, exact: true });

    for (
      let tabPress = 0;
      tabPress < 20 && !(await target.evaluate((input) => input === document.activeElement));
      tabPress++
    ) {
      await page.keyboard.press('Tab');
    }

    await expect(target).toBeFocused();

    await page.keyboard.press('ArrowDown');

    const calendar = page.getByRole('dialog', { name: copy.targetTime, exact: true });

    await expect(calendar.locator('*:focus')).toHaveCount(1);

    const today = calendar.getByRole('button', { name: 'Sunday, 18 October 2026', exact: true });

    for (
      let tabPress = 0;
      tabPress < 12 && !(await today.evaluate((day) => day === document.activeElement));
      tabPress++
    ) {
      await page.keyboard.press('Tab');

      await expect(calendar.locator('*:focus')).toHaveCount(1);
    }

    await expect(today).toBeFocused();

    await page.keyboard.press('ArrowLeft');

    await expect(
      calendar.getByRole('button', { name: 'Saturday, 17 October 2026', exact: true }),
    ).toBeFocused();

    await page.keyboard.press('Enter');

    await expect(calendar.getByLabel(dictionary.schedules.hours, { exact: true })).toBeFocused();

    await page.keyboard.type('1');
    await expect(calendar.getByLabel(dictionary.schedules.hours, { exact: true })).toHaveValue(
      '01',
    );
    await page.keyboard.press('ArrowRight');

    await expect(calendar.getByLabel(dictionary.schedules.minutes, { exact: true })).toBeFocused();

    await page.keyboard.type('23');

    await expect(
      calendar.getByLabel(dictionary.common.calendar.seconds, { exact: true }),
    ).toBeFocused();

    await page.keyboard.type('45');

    await expect(
      calendar.getByLabel(dictionary.schedules.timePeriod, { exact: true }),
    ).toBeFocused();

    await page.keyboard.press('p');

    await expect(calendar.getByLabel(dictionary.schedules.timePeriod, { exact: true })).toHaveValue(
      'PM',
    );

    await page.keyboard.press('Tab');

    const submit = calendar.getByRole('button', {
      name: dictionary.common.actions.ok,
      exact: true,
    });

    await expect(submit).toBeFocused();

    await page.keyboard.press('Tab');

    await expect(calendar.locator('*:focus')).toHaveCount(1);

    await page.keyboard.press('Shift+Tab');

    await expect(submit).toBeFocused();

    await page.keyboard.press('Enter');

    await expect(calendar).toBeHidden();
    await expect(target).toBeFocused();
    await expect
      .poll(
        () =>
          boundary.requests.filter((request) => request.path === '/physical-restore-tokens').at(-1)
            ?.body,
      )
      .toMatchObject({
        type: 'POINT_IN_TIME',
        databaseId: database.id,
        targetTime: '2026-10-17T17:23:45.000Z',
      });

    await page.keyboard.press('ArrowDown');

    await expect(calendar.locator('*:focus')).toHaveCount(1);

    await page.keyboard.press('Escape');

    await expect(calendar).toBeHidden();
    await expect(target).toBeFocused();
    await expect(restore).toBeVisible();

    await page.keyboard.press('Escape');

    await expect(restore).toBeHidden();
    await expect(invoker).toBeFocused();
  });

  test('submits a chosen calendar date and time as UTC and restores focus to its field', async ({
    page,
    boundary,
  }) => {
    const copy = dictionary.backups.physical.restore;
    await openAccount({ page, locale: 'en', theme: 'light' });
    await openWorkspaceScreen({ page, locale: 'en', tab: 'databases' });
    await page
      .getByRole('button', { name: dictionary.backups.physical.list.restore, exact: true })
      .click();
    const restore = page.getByRole('dialog', {
      name: dictionary.backups.physical.list.pointInTimeRestoreTitle,
      exact: true,
    });
    await expect(restore.locator('pre').first()).toContainText('restore-token');
    const target = restore.getByRole('textbox', { name: copy.targetTime, exact: true });
    await target.click();
    const calendar = page.getByRole('dialog', { name: copy.targetTime, exact: true });
    await calendar.getByRole('button', { name: 'Saturday, 17 October 2026', exact: true }).click();
    await calendar.getByLabel(dictionary.schedules.hours, { exact: true }).fill('01');
    await calendar.getByLabel(dictionary.schedules.minutes, { exact: true }).fill('23');
    await calendar.getByLabel(dictionary.common.calendar.seconds, { exact: true }).fill('45');
    await calendar.getByLabel(dictionary.schedules.timePeriod, { exact: true }).selectOption('PM');

    await calendar.getByRole('button', { name: dictionary.common.actions.ok, exact: true }).click();

    await expect(calendar).toBeHidden();
    await expect(target).toBeFocused();
    await expect
      .poll(
        () =>
          boundary.requests.filter((request) => request.path === '/physical-restore-tokens').at(-1)
            ?.body,
      )
      .toMatchObject({
        type: 'POINT_IN_TIME',
        databaseId: database.id,
        targetTime: '2026-10-17T17:23:45.000Z',
      });
    await expect(restore.locator('pre').first()).toContainText('restore-token');
  });
});
