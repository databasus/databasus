import { expect } from '@playwright/test';

import type { components } from '../../src/shared/api/schema.gen';
import { DICTIONARIES } from '../../src/shared/i18n/dictionaries';
import { createTestWalStreamStatistics } from '../../src/shared/lib/test/buildWalStreamStatistics';
import { applicationFixtures } from '../applicationFixtures';
import { assertAccountReflow } from '../assertAccountReflow';
import { assertEnlargedTextReflow } from '../assertEnlargedTextReflow';
import { assertScreenshot } from '../assertScreenshot';
import { test } from '../browserTest';
import { enlargeText } from '../enlargeText';
import { apiResponse } from '../HttpFixture';
import { openAccount } from '../openAccount';

const database: components['schemas']['Database'] = {
  id: 'physical-id',
  workspaceId: 'workspace-id',
  name: 'WAL source',
  notifierIds: [],
  settings: {
    type: 'POSTGRES_PHYSICAL',
    host: 'postgres.example.com',
    port: 5432,
    username: 'replication',
    sslMode: 'require',
    backupType: 'FULL_INCREMENTAL_WAL_STREAM',
    sshTunnel: { isEnabled: false },
    version: '18',
  },
};
const chartCopy = DICTIONARIES.en.backups.physical.monitoring.charts;
const statistics = createTestWalStreamStatistics();
statistics.series.points = Array.from({ length: 12 }, (_, index) => ({
  startedAt: new Date(Date.parse('2026-10-18T11:48:00Z') + index * 60_000).toISOString(),
  sourceWriteBytesPerSecond: 10485760 + index * 1000,
  archiveBytesPerSecond: 10000000,
  archivedStoredBytesPerSecond: 3000000,
  maxSlotLagBytes: 2097152,
  maxUnarchivedWalBytes: 20000000,
  maxLocalQueueBytes: 0,
  localQueuePauseBytes: 536870912,
  restarts: 0,
  pauses: 0,
  sessionStarts: 0,
  highestConcurrentUploads: 2,
  segmentWriteAvgMillis: 100,
  segmentWriteMaxMillis: 200,
}));
test.use({
  httpFixtures: [
    [
      ...applicationFixtures().filter((fixture) => fixture.path !== '/databases'),
      apiResponse({ method: 'get', path: '/databases', status: 200, body: { items: [database] } }),
      apiResponse({ method: 'get', path: '/databases/{databaseId}', status: 200, body: database }),
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
        path: '/physical-backups',
        status: 200,
        body: { items: [], totalUsageMb: 0 },
      }),
      apiResponse({
        method: 'get',
        path: '/backup-configs/{databaseId}',
        status: 200,
        body: {
          cleanup: { pendingFileCount: 0, failedFileCount: 0 },
          databaseId: database.id,
          isBackupsEnabled: true,
          storageId: null,
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
        path: '/wal-streams/{databaseId}/statistics',
        status: 200,
        body: statistics,
      }),
      apiResponse({
        method: 'get',
        path: '/wal-streams/{databaseId}',
        status: 200,
        body: { isStreamerRunning: true, current: null },
      }),
    ],
    { scope: 'test' },
  ],
});

for (const theme of ['light', 'dark'] as const) {
  test(`renders WAL plots with positive measured dimensions and deterministic series in the ${theme} theme`, async ({
    page,
  }) => {
    await page.clock.setFixedTime(new Date('2026-10-18T12:00:00Z'));
    await page.addInitScript((selectedTheme) => {
      localStorage.setItem('databasus_user_token', 'synthetic-session');
      localStorage.setItem('databasus_user_id', 'owner-id');
      localStorage.setItem('databasus-locale', 'en');
      localStorage.setItem('databasus-theme', selectedTheme);
    }, theme);

    await page.goto('/');
    await page.getByText('Monitoring', { exact: true }).click();

    const plot = page.locator('.recharts-surface').first();
    await expect(plot).toBeVisible();
    const box = await plot.boundingBox();
    expect(box!.width).toBeGreaterThan(0);
    expect(box!.height).toBeGreaterThan(0);
    expect(await page.locator('.recharts-line-curve').count()).toBeGreaterThan(0);

    const keepingUpChart = page
      .locator('section,div')
      .filter({ has: page.getByText(chartCopy.keepingUpTitle, { exact: true }) })
      .filter({ has: page.locator('.recharts-surface') })
      .last();
    await expect(keepingUpChart.getByText(chartCopy.databaseWrites, { exact: true })).toBeVisible();
    await expect(keepingUpChart.getByText(chartCopy.savedToStorage, { exact: true })).toBeVisible();

    const drawnCurves = keepingUpChart.locator('.recharts-line-curve');
    await expect(drawnCurves).toHaveCount(2);
    for (const drawnCurve of await drawnCurves.all()) {
      const curveBox = await drawnCurve.boundingBox();
      expect(curveBox!.width).toBeGreaterThan(0);
      expect(await drawnCurve.getAttribute('d')).toMatch(/^M[\d.]+,[\d.]+/);
    }

    const axisTicks = keepingUpChart.locator('.recharts-cartesian-axis-tick-value');
    const timeTicks = axisTicks.filter({ hasText: /^\d{1,2}:\d{2}/ });
    const amountTicks = axisTicks.filter({ hasText: /^\d[\d.,]*\s*[KMGT]?i?B/ });
    await expect(timeTicks.first()).toBeVisible();
    await expect(amountTicks.first()).toBeVisible();
    await assertEnlargedTextReflow(page);
    await assertScreenshot(page, `wal-overview-${theme}.png`);
    const timeline = page
      .getByText(DICTIONARIES.en.backups.physical.monitoring.restoreTimeline.title, { exact: true })
      .locator('..')
      .locator('..');
    await timeline.locator('div.absolute').first().hover();
    await expect(page.getByRole('tooltip')).toContainText(
      DICTIONARIES.en.backups.physical.monitoring.restoreTimeline.restorable,
    );
    await assertEnlargedTextReflow(page);

    await page.mouse.move(0, 0);
    const technical = page.getByRole('heading', {
      name: DICTIONARIES.en.backups.physical.monitoring.technicalDetails.title,
      exact: true,
    });
    await technical.evaluate((heading) => heading.scrollIntoView({ block: 'start' }));
    await expect(technical).toBeVisible();
    await assertEnlargedTextReflow(page);
  });
}

test('keeps technical values, tips and help readable with French 200-percent text at 320px', async ({
  page,
  httpFixtures,
}) => {
  const copy = DICTIONARIES.fr.backups.physical.monitoring;
  const fixture = httpFixtures.find(
    (fixture) => fixture.path === '/wal-streams/{databaseId}/statistics',
  )!;
  Object.assign(
    fixture,
    apiResponse({
      method: 'get',
      path: '/wal-streams/{databaseId}/statistics',
      status: 200,
      body: {
        ...statistics,
        health: { ...statistics.health, notices: [{ code: 'NO_SLOT_WAL_LIMIT', level: 'INFO' }] },
      },
    }),
  );
  await page.setViewportSize({ width: 320, height: 1000 });
  await openAccount({ page, locale: 'fr', theme: 'light' });
  await page.getByRole('link', { name: /WAL source/ }).click();
  await page.getByText(DICTIONARIES.fr.databases.tabs.monitoring, { exact: true }).click();
  const plot = page.locator('.recharts-surface').first();
  await expect(plot).toBeVisible();
  expect((await plot.boundingBox())!.width).toBeGreaterThan(0);
  await assertEnlargedTextReflow(page);

  const main = page.getByRole('main');
  await enlargeText(main);

  const technical = page.getByRole('heading', { name: copy.technicalDetails.title, exact: true });
  await technical.evaluate((heading) => heading.scrollIntoView({ block: 'start' }));

  const help = page.getByLabel(copy.tooltips.notReceivedYet, { exact: true }).first();
  await page.keyboard.press('Tab');
  await help.focus();
  await expect(help).toBeFocused();
  await expect(page.getByRole('tooltip')).toHaveText(copy.tooltips.notReceivedYet);
  await enlargeText(page.getByRole('tooltip'));

  await page.keyboard.press('Tab');
  const tip = page.getByText(copy.notices.noSlotWalLimit.title, { exact: true });
  await tip.evaluate((element) => element.scrollIntoView({ block: 'start' }));

  const slotStatus = page.getByText(copy.details.slotStatus, { exact: true }).locator('..');
  await expect(slotStatus.getByRole('definition')).toHaveText('reserved');
  await slotStatus.evaluate((row) => row.closest('dl')!.scrollIntoView({ block: 'start' }));

  await assertAccountReflow(main);
});

test('stops the verdict transition only while the reduced motion preference is set', async ({
  page,
}) => {
  await page.clock.setFixedTime(new Date('2026-10-18T12:00:00Z'));
  await page.addInitScript(() => {
    localStorage.setItem('databasus_user_token', 'synthetic-session');
    localStorage.setItem('databasus_user_id', 'owner-id');
    localStorage.setItem('databasus-locale', 'en');
  });
  await page.emulateMedia({ reducedMotion: 'reduce' });

  await page.goto('/');
  await page.getByText('Monitoring', { exact: true }).click();

  const verdictBanner = page.locator('div.rounded-lg.border').filter({ hasText: /./ }).first();
  await expect(page.locator('.recharts-surface').first()).toBeVisible();
  const readTransitionDuration = () =>
    verdictBanner.evaluate((banner) => getComputedStyle(banner).transitionDuration);

  await expect.poll(readTransitionDuration).toBe('0s');

  await page.emulateMedia({ reducedMotion: 'no-preference' });
  await expect.poll(readTransitionDuration).toContain('0.3s');

  await page.emulateMedia({ reducedMotion: 'reduce' });
  await expect.poll(readTransitionDuration).toBe('0s');
});
