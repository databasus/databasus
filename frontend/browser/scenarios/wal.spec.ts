import { expect } from '@playwright/test';

import type { components } from '../../src/shared/api/schema.gen';
import { createTestWalStreamStatistics } from '../../src/shared/lib/test/buildWalStreamStatistics';
import { applicationFixtures } from '../applicationFixtures';
import { test } from '../browserTest';
import { apiResponse } from '../HttpFixture';

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

test('renders WAL plots with positive measured dimensions and deterministic series', async ({
  page,
}) => {
  await page.clock.setFixedTime(new Date('2026-10-18T12:00:00Z'));
  await page.addInitScript(() => {
    localStorage.setItem('databasus_user_token', 'synthetic-session');
    localStorage.setItem('databasus_user_id', 'owner-id');
    localStorage.setItem('databasus-locale', 'en');
    localStorage.setItem('databasus-theme', 'light');
  });
  await page.goto('/');
  await page.getByText('Monitoring', { exact: true }).click();
  const plot = page.locator('.recharts-surface').first();
  await expect(plot).toBeVisible();
  const box = await plot.boundingBox();
  expect(box!.width).toBeGreaterThan(0);
  expect(box!.height).toBeGreaterThan(0);
  expect(await page.locator('.recharts-line-curve').count()).toBeGreaterThan(0);
});
