import { expect } from '@playwright/test';

import type { components } from '../../src/shared/api/schema.gen';
import { applicationFixtures, workspace } from '../applicationFixtures';
import { test } from '../browserTest';
import { enlargeText } from '../enlargeText';
import { apiResponse } from '../HttpFixture';
import { openAccount } from '../openAccount';
import { openWorkspaceScreen } from '../openWorkspaceScreen';

for (const kind of ['logical', 'physical', 'storage'] as const) {
  test.describe(`${kind} backup usage`, () => {
    const storage: components['schemas']['Storage'] = {
      id: 'archive-id',
      workspaceId: workspace.id,
      name: 'Backup archive',
      settings: { type: 'LOCAL' },
    };
    const database: components['schemas']['Database'] = {
      id: 'orders-id',
      workspaceId: workspace.id,
      name: 'Orders',
      notifierIds: [],
      settings:
        kind === 'physical'
          ? {
              type: 'POSTGRES_PHYSICAL',
              host: 'postgres.example.com',
              port: 5432,
              username: 'backup',
              sslMode: 'disable',
              backupType: 'FULL_INCREMENTAL_WAL_STREAM',
              sshTunnel: { isEnabled: false },
              version: '18',
            }
          : {
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
      settings:
        kind === 'physical'
          ? {
              type: 'PHYSICAL',
              fullBackupSchedule: { type: 'HOURLY' },
              incrementalBackupSchedule: null,
              retention: { policy: 'CHAINS', chainsCount: 2, fullBackups: null },
              walLagThresholdBytes: 0,
              notificationTriggers: [],
            }
          : {
              type: 'LOGICAL',
              schedule: { type: 'HOURLY' },
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

    const logicalBackup: components['schemas']['LogicalBackup'] = {
      id: 'logical-backup',
      databaseId: database.id,
      storageId: storage.id,
      status: 'COMPLETED',
      sizeMb: 10752,
      rawDatabaseSizeMb: 0,
      durationMs: 3000,
      encryption: 'NONE',
      verificationStatus: 'NOT_VERIFIED',
      createdAt: '2026-10-27T10:00:00Z',
    };
    const physicalBackup: components['schemas']['PhysicalBackupListRow'] = {
      id: 'physical-backup',
      type: 'FULL',
      status: 'COMPLETED',
      timelineId: 1,
      startLsn: '0/1000000',
      stopLsn: '0/2000000',
      hasGapBefore: false,
      sizeMb: 10752,
      createdAt: '2026-10-27T10:00:00Z',
      completedAt: '2026-10-27T10:00:03Z',
    };

    test.use({
      httpFixtures: [
        [
          ...applicationFixtures().filter(
            (fixture) =>
              ![
                '/databases',
                '/backup-configs',
                '/storages',
                '/storages/{storageId}/backup-usage',
              ].includes(fixture.path),
          ),
          apiResponse({
            method: 'get',
            path: '/databases',
            status: 200,
            body: { items: kind === 'storage' ? [] : [database] },
          }),
          apiResponse({
            method: 'get',
            path: '/databases/{databaseId}',
            status: 200,
            body: database,
          }),
          apiResponse({
            method: 'get',
            path: '/backup-configs',
            status: 200,
            body: { items: kind === 'storage' ? [] : [config] },
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
            path: '/storages',
            status: 200,
            body: { items: [storage] },
          }),
          apiResponse({ method: 'get', path: '/storages/{storageId}', status: 200, body: storage }),
          apiResponse({
            method: 'get',
            path: '/storages/{storageId}/backup-usage',
            status: 200,
            body: { totalUsageMb: 10752 },
          }),
          apiResponse({
            method: 'get',
            path: '/logical-backups',
            status: 200,
            body: { items: [logicalBackup], totalUsageMb: 10752 },
          }),
          apiResponse({
            method: 'get',
            path: '/physical-backups',
            status: 200,
            body: { items: [physicalBackup], totalUsageMb: 10752 },
          }),
        ],
        { scope: 'test' },
      ],
    });

    for (const theme of ['light', 'dark']) {
      for (const layout of ['desktop', 'mobile', 'enlarged'] as const) {
        test(`places compact usage with ${theme} theme and ${layout} text`, async ({ page }) => {
          const isMobile = layout === 'mobile';
          await page.setViewportSize({ width: isMobile ? 320 : 1280, height: 900 });
          await openAccount({ page, theme, locale: 'en' });
          await openWorkspaceScreen({
            page,
            locale: 'en',
            tab: kind === 'storage' ? 'storages' : 'databases',
          });

          if (isMobile)
            await page
              .getByRole(kind === 'storage' ? 'button' : 'link', {
                name: kind === 'storage' ? /^Backup archive/ : /^Orders/,
              })
              .click();

          const total =
            kind === 'storage' || isMobile
              ? page.getByText('Total usage: 10.5 GB', { exact: true })
              : page.getByRole('columnheader', { name: /^Size.*10.5 GB/ });
          await expect(total).toBeVisible();

          if (layout === 'enlarged') await enlargeText(page.locator('#root'));

          await expect(total).toBeVisible();

          if (kind === 'storage') {
            const name = await page.getByText(storage.name, { exact: true }).last().boundingBox();
            const usage = await total.boundingBox();
            const settings = await page
              .getByText('Storage settings', { exact: true })
              .boundingBox();

            expect(usage!.y).toBeGreaterThanOrEqual(name!.y + name!.height);
            expect(usage!.y + usage!.height).toBeLessThanOrEqual(settings!.y);
          } else if (isMobile) {
            const firstCard = page.locator('.mb-2.rounded-lg').filter({
              has: page.getByText('10.5 GB', { exact: true }),
            });
            await expect(firstCard).toHaveCount(1);
            await expect(firstCard).toBeVisible();
            await expect(total.filter({ visible: true })).toHaveCount(1);

            const usage = await total.boundingBox();
            const card = await firstCard.boundingBox();

            expect(usage!.y + usage!.height).toBeLessThanOrEqual(card!.y);
          } else {
            const geometry = await total.evaluate((header) => {
              const value = header.querySelector<HTMLElement>('.text-xs')!;
              const label = Array.from(value.parentElement!.childNodes).find(
                (node) => node.nodeType === Node.TEXT_NODE && node.textContent?.includes('Size'),
              )!;
              const range = document.createRange();
              range.selectNodeContents(label);
              const labelBox = range.getBoundingClientRect();
              const valueBox = value.getBoundingClientRect();

              return {
                labelCenter: labelBox.y + labelBox.height / 2,
                valueCenter: valueBox.y + valueBox.height / 2,
                labelRight: labelBox.right,
                valueLeft: valueBox.left,
                weight: getComputedStyle(value).fontWeight,
              };
            });

            expect(Math.abs(geometry.labelCenter - geometry.valueCenter)).toBeLessThan(4);
            expect(geometry.valueLeft).toBeGreaterThanOrEqual(geometry.labelRight);
            expect(geometry.weight).toBe('400');
            await expect(page.getByText('Total usage: 10.5 GB', { exact: true })).toBeHidden();
          }
        });
      }
    }
  });
}
