import { expect } from '@playwright/test';

import type { components } from '../../src/shared/api/schema.gen';
import { DICTIONARIES } from '../../src/shared/i18n/dictionaries';
import { applicationFixtures, smtpNotifier, workspace } from '../applicationFixtures';
import { assertAccountReflow } from '../assertAccountReflow';
import { assertEnlargedTextReflow } from '../assertEnlargedTextReflow';
import { assertScreenshot } from '../assertScreenshot';
import { test } from '../browserTest';
import { enlargeText } from '../enlargeText';
import { apiResponse } from '../HttpFixture';
import { openAccount } from '../openAccount';

const targetWorkspace = { ...workspace, id: 'target-workspace', name: 'Target archive team' };
const targetStorage: components['schemas']['Storage'] = {
  id: 'target-storage',
  workspaceId: targetWorkspace.id,
  name: 'Target archive',
  settings: { type: 'LOCAL' },
};
const database: components['schemas']['Database'] = {
  id: 'orders-id',
  workspaceId: workspace.id,
  name: 'Orders',
  notifierIds: [smtpNotifier.id],
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

test.use({
  httpFixtures: [
    [
      ...applicationFixtures().filter(
        (fixture) =>
          !['/workspaces', '/databases', '/storages', '/backup-configs'].includes(fixture.path),
      ),
      apiResponse({
        method: 'get',
        path: '/workspaces',
        status: 200,
        body: { items: [workspace, targetWorkspace] },
      }),
      apiResponse({ method: 'get', path: '/databases', status: 200, body: { items: [database] } }),
      apiResponse({ method: 'get', path: '/databases/{databaseId}', status: 200, body: database }),
      apiResponse({
        method: 'get',
        path: '/storages',
        status: 200,
        body: { items: [targetStorage] },
      }),
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
        path: '/verification-configs/{databaseId}',
        status: 200,
        body: {
          databaseId: database.id,
          isAutomaticVerificationEnabled: false,
          scheduleType: 'AFTER_BACKUP',
          schedule: { type: 'DAILY', timeOfDay: '00:00' },
          notificationTriggers: [],
        },
      }),
    ],
    { scope: 'test' },
  ],
});

for (const locale of ['fr', 'ru'] as const) {
  test(`keeps transfer choices and nested creation usable with ${locale} text enlarged at 320px`, async ({
    page,
    boundary,
  }) => {
    const dictionary = DICTIONARIES[locale];

    await openAccount({ page, locale, theme: locale === 'fr' ? 'light' : 'dark' });
    await page.getByText(dictionary.databases.tabs.config, { exact: true }).click();
    await page
      .getByRole('button', { name: dictionary.common.actions.transfer, exact: true })
      .click();
    const dialog = page.getByRole('dialog', {
      name: dictionary.databases.transfer.title,
    });

    await dialog
      .getByRole('combobox', { name: dictionary.databases.transfer.targetWorkspace, exact: true })
      .click();
    await page.getByRole('option', { name: targetWorkspace.name, exact: true }).click();
    await dialog
      .getByRole('combobox', { name: dictionary.databases.transfer.storage.title, exact: true })
      .click();
    await page.getByRole('option', { name: targetStorage.name, exact: true }).click();

    await expect(page.getByRole('listbox')).toHaveCount(0);

    await page.setViewportSize({ width: 320, height: 1000 });
    await enlargeText(dialog);
    await assertEnlargedTextReflow(page);

    if (locale === 'fr') await assertScreenshot(page, 'database-transfer-fr-narrow-200.png');
    await assertAccountReflow(dialog);
    await expect(
      dialog.getByRole('button', { name: dictionary.databases.transfer.submit, exact: true }),
    ).toBeEnabled();

    for (const creation of [
      {
        action: dictionary.databases.transfer.storage.createNew,
        title: dictionary.databases.transfer.storage.addTitle,
      },
      {
        action: dictionary.databases.notifiers.createNew,
        title: dictionary.databases.notifiers.addTitle,
      },
    ]) {
      await dialog.getByRole('button', { name: creation.action, exact: true }).click();
      const editor = page.getByRole('dialog', { name: creation.title, exact: true });

      await expect(
        editor.getByRole('textbox', { name: dictionary.common.fields.name, exact: true }),
      ).toBeVisible();

      await editor.evaluate(async (element) => {
        await Promise.allSettled(
          element.getAnimations({ subtree: true }).map((animation) => animation.finished),
        );
      });
      await enlargeText(editor);
      await assertEnlargedTextReflow(page);

      await assertAccountReflow(editor);

      await editor
        .getByRole('button', { name: dictionary.common.actions.close, exact: true })
        .click();

      await expect(editor).not.toBeVisible();
      await expect(dialog.getByText(targetWorkspace.name, { exact: true })).toBeVisible();
      await expect(dialog.getByText(targetStorage.name, { exact: true })).toBeVisible();
      await assertEnlargedTextReflow(page);

      await assertAccountReflow(dialog);
    }

    expect(boundary.requests.every((request) => request.method === 'GET')).toBe(true);
  });
}
