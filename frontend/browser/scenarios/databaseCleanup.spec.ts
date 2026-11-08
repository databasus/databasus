import { expect } from '@playwright/test';

import type { components } from '../../src/shared/api/schema.gen';
import { DICTIONARIES } from '../../src/shared/i18n/dictionaries';
import { applicationFixtures, workspace } from '../applicationFixtures';
import { assertAccountReflow } from '../assertAccountReflow';
import { assertEnlargedTextReflow } from '../assertEnlargedTextReflow';
import { test } from '../browserTest';
import { enlargeText } from '../enlargeText';
import { apiResponse } from '../HttpFixture';
import { openAccount } from '../openAccount';
import { openWorkspaceScreen } from '../openWorkspaceScreen';

type Database = components['schemas']['Database'];
type BackupConfig = components['schemas']['BackupConfig'];

const logicalDatabase: Database = {
  id: 'cleanup-database',
  workspaceId: workspace.id,
  name: 'Cleanup database',
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
const physicalDatabase: Database = {
  ...logicalDatabase,
  settings: {
    type: 'POSTGRES_PHYSICAL',
    backupType: 'FULL',
    host: 'postgres.example.com',
    port: 5432,
    username: 'backup',
    sslMode: 'disable',
    sshTunnel: { isEnabled: false },
  },
};
const retention = {
  policy: 'COUNT' as const,
  count: 5,
  period: 'WEEK' as const,
  gfsHours: 0,
  gfsDays: 0,
  gfsWeeks: 0,
  gfsMonths: 0,
  gfsYears: 0,
};
const logicalConfig: BackupConfig = {
  databaseId: logicalDatabase.id,
  storageId: null,
  isBackupsEnabled: false,
  encryption: 'NONE',
  cleanup: { pendingFileCount: 1234, failedFileCount: 2 },
  settings: {
    type: 'LOGICAL',
    schedule: { type: 'HOURLY' },
    retention,
    isRetryIfFailed: true,
    maxFailedTriesCount: 3,
    notificationTriggers: [],
  },
};
const physicalConfig: BackupConfig = {
  ...logicalConfig,
  settings: {
    type: 'PHYSICAL',
    fullBackupSchedule: { type: 'HOURLY' },
    incrementalBackupSchedule: null,
    retention: {
      policy: 'FULL_BACKUPS',
      chainsCount: 0,
      fullBackups: { ...retention, policy: 'LAST_N' },
    },
    walLagThresholdBytes: 0,
    notificationTriggers: [],
  },
};

for (const method of ['logical', 'physical'] as const) {
  const database = method === 'logical' ? logicalDatabase : physicalDatabase;
  const config = method === 'logical' ? logicalConfig : physicalConfig;
  test.describe(`${method} cleanup`, () => {
    test.use({
      httpFixtures: [
        [
          ...applicationFixtures().filter(
            (fixture) => !['/databases', '/backup-configs'].includes(fixture.path),
          ),
          apiResponse({
            method: 'get',
            path: '/databases',
            status: 200,
            body: { items: [database] },
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
            body: { items: [config] },
          }),
          apiResponse({
            method: 'get',
            path: '/logical-backups',
            status: 200,
            body: { totalUsageMb: 0, items: [] },
          }),
          apiResponse({
            method: 'get',
            path: '/physical-backups',
            status: 200,
            body: { items: [], totalUsageMb: 0 },
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
              schedule: { type: 'HOURLY' },
              notificationTriggers: [],
            },
          }),
        ],
        { scope: 'test' },
      ],
    });

    for (const theme of ['light', 'dark']) {
      test(`keeps cleanup beside cleared Storage with disabled backups in ${theme} theme`, async ({
        page,
      }) => {
        let isReadFailing = false;
        let isComplete = false;
        await page.route(/\/api\/v1\/backup-configs\/[^/?]+$/, (route) =>
          route.fulfill(
            isReadFailing
              ? {
                  status: 500,
                  contentType: 'application/problem+json',
                  json: {
                    type: 'about:blank',
                    title: 'Internal server error',
                    status: 500,
                    detail: 'Internal server error',
                    code: 'internal',
                  },
                }
              : {
                  json: {
                    ...config,
                    cleanup: isComplete
                      ? { pendingFileCount: 0, failedFileCount: 0 }
                      : config.cleanup,
                  },
                },
          ),
        );
        await page.setViewportSize({ width: 320, height: 900 });
        await openAccount({ page, theme });
        await openWorkspaceScreen({ page, locale: 'en', tab: 'databases' });
        await page.getByRole('link', { name: /^Cleanup database/ }).click();
        await page.getByRole('tab', { name: 'Config', exact: true }).click();
        const notice = page.getByRole('status').filter({ hasText: 'Deleting old files' });
        await expect(notice).toContainText('1,234');
        await expect(notice).toContainText('Deletion will be retried for files: 2.');
        const precedingRow = notice.locator('xpath=preceding-sibling::*[1]');
        await expect(precedingRow).toContainText('Storage');
        await expect(page.getByText('Storage switch history', { exact: true })).toHaveCount(0);
        await enlargeText(notice.locator('xpath=ancestor::div[.//button][1]'));
        await assertEnlargedTextReflow(page);

        await assertAccountReflow(notice);
        if (method === 'logical')
          await expect.soft(notice).toHaveScreenshot(`cleanup-${method}-${theme}.png`);

        await page
          .getByText(DICTIONARIES.en.databases.config.sections.backupConfig, { exact: true })
          .locator('xpath=..')
          .locator('img[src="/icons/pen-gray.svg"]')
          .click();
        await expect(page.getByRole('button', { name: 'Save', exact: true })).toBeVisible();
        const retentionInput = page.getByRole('spinbutton', {
          name: method === 'logical' ? 'Most recent backups' : 'Most recent full backups',
          exact: true,
        });
        await retentionInput.fill('37');

        await expect(notice.locator('xpath=preceding-sibling::*[1]')).toContainText('Storage');
        await assertEnlargedTextReflow(page);

        await assertAccountReflow(notice);

        isReadFailing = true;
        await expect(notice.getByRole('button', { name: 'Retry status' })).toBeVisible();
        await enlargeText(notice);
        await assertEnlargedTextReflow(page);

        await assertAccountReflow(notice);
        await expect(notice).toContainText('1,234');
        await expect(retentionInput).toHaveValue('37');

        await page
          .getByRole('button', { name: DICTIONARIES.en.app.navigation.menu, exact: true })
          .click();
        await page.getByRole('button', { name: /Language:/ }).click();
        await page
          .getByRole('menuitemradio', { name: DICTIONARIES.ru.language.nativeName, exact: true })
          .click();
        await page
          .getByRole('button', { name: DICTIONARIES.ru.common.actions.close, exact: true })
          .click();
        await expect(notice).toHaveCount(0);
        const translatedNotice = page
          .getByRole('status')
          .filter({ hasText: DICTIONARIES.ru.backups.cleanup.pending.split('{{count}}')[0] });
        await expect(
          translatedNotice.getByRole('button', {
            name: DICTIONARIES.ru.backups.cleanup.retryStatus,
          }),
        ).toBeVisible();
        await expect(translatedNotice).toContainText(DICTIONARIES.ru.errors.internal);
        isReadFailing = false;
        isComplete = true;
        await translatedNotice
          .getByRole('button', { name: DICTIONARIES.ru.backups.cleanup.retryStatus })
          .click();
        await expect(translatedNotice).toHaveCount(0);
        await expect(
          page.getByRole('spinbutton', {
            name:
              method === 'logical'
                ? DICTIONARIES.ru.backups.logical.config.retention.count
                : DICTIONARIES.ru.backups.physical.config.fullBackupsCount,
            exact: true,
          }),
        ).toHaveValue('37');
        await expect(
          page.getByRole('button', { name: DICTIONARIES.ru.common.actions.save, exact: true }),
        ).toBeEnabled();
      });
    }
  });
}
