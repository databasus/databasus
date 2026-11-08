import { expect, type Page } from '@playwright/test';

import type { components } from '../../src/shared/api/schema.gen';
import { DICTIONARIES } from '../../src/shared/i18n/dictionaries';
import { applicationFixtures, workspace } from '../applicationFixtures';
import { assertEnlargedTextReflow } from '../assertEnlargedTextReflow';
import { test } from '../browserTest';
import { apiResponse } from '../HttpFixture';
import { openAccount } from '../openAccount';
import { openWorkspaceScreen } from '../openWorkspaceScreen';

type BackupConfig = components['schemas']['BackupConfig'];

interface OptionChoice {
  controlName: string;
  optionName: string;
}

const dictionary = DICTIONARIES.ru;
const physicalConfigCopy = dictionary.backups.physical.config;

const database: components['schemas']['Database'] = {
  id: 'orders-id',
  workspaceId: workspace.id,
  name: 'Orders',
  notifierIds: [],
  settings: {
    type: 'POSTGRES_PHYSICAL',
    backupType: 'FULL_INCREMENTAL',
    host: 'postgres.example.com',
    port: 5432,
    username: 'backup',
    sslMode: 'disable',
    sshTunnel: { isEnabled: false },
  } as components['schemas']['Database']['settings'],
};

const storedConfig: BackupConfig = {
  cleanup: { pendingFileCount: 0, failedFileCount: 0 },
  databaseId: database.id,
  storageId: null,
  isBackupsEnabled: true,
  encryption: 'NONE',
  settings: {
    type: 'PHYSICAL',
    fullBackupSchedule: { type: 'HOURLY' },
    incrementalBackupSchedule: { type: 'HOURLY' },
    retention: {
      policy: 'CHAINS_AND_FULL_BACKUPS',
      chainsCount: 3,
      fullBackups: {
        policy: 'GFS',
        count: 0,
        gfsHours: 0,
        gfsDays: 7,
        gfsWeeks: 4,
        gfsMonths: 0,
        gfsYears: 0,
      },
    },
    walLagThresholdBytes: 0,
    notificationTriggers: ['BACKUP_FAILED'],
  },
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
    ],
    { scope: 'test' },
  ],
});

const serveBackupConfigRoutes = async (page: Page) => {
  const submittedConfigs: BackupConfig[] = [];
  let currentConfig = storedConfig;

  await page.route(/\/api\/v1\/databases\/[^/?]+$/, (route) => route.fulfill({ json: database }));

  await page.route(/\/api\/v1\/backup-configs\/[^/?]+$/, async (route) => {
    if (route.request().method() === 'PUT') {
      const submittedConfig = route.request().postDataJSON() as BackupConfig;

      submittedConfigs.push(submittedConfig);
      currentConfig = { ...currentConfig, ...submittedConfig, databaseId: database.id };
    }

    await route.fulfill({ json: currentConfig });
  });

  return submittedConfigs;
};

const openBackupConfigSection = async (page: Page) => {
  await page.getByRole('main').getByText(dictionary.databases.tabs.config, { exact: true }).click();

  const sectionTitle = page.getByText(dictionary.databases.config.sections.backupConfig, {
    exact: true,
  });

  await sectionTitle.locator('xpath=..').locator('img[src="/icons/pen-gray.svg"]').click();
};

const chooseOption = async (page: Page, { controlName, optionName }: OptionChoice) => {
  await page.getByRole('combobox', { name: controlName, exact: true }).click();
  await page.getByRole('option', { name: optionName, exact: true }).click();
};

for (const width of [320, 1280]) {
  test(`edits a physical retention in Russian and reads it again when viewport width is ${width}`, async ({
    page,
  }) => {
    const submittedConfigs = await serveBackupConfigRoutes(page);
    await page.setViewportSize({ width, height: 720 });
    await openAccount({ page, locale: 'ru' });
    await openWorkspaceScreen({ page, locale: 'ru', tab: 'databases' });

    if (width === 320) await page.getByRole('link', { name: /^Orders/ }).click();

    await openBackupConfigSection(page);

    const saveButton = page.getByRole('button', { name: dictionary.common.actions.save });
    const dailyInput = page.getByRole('spinbutton', {
      name: physicalConfigCopy.gfsFields.daily,
      exact: true,
    });

    await expect(saveButton).toBeDisabled();

    await dailyInput.fill('9');
    await chooseOption(page, {
      controlName: physicalConfigCopy.fullBackups,
      optionName: dictionary.backups.physical.fullBackupsPolicies.lastN,
    });

    const countInput = page.getByRole('spinbutton', {
      name: physicalConfigCopy.fullBackupsCount,
      exact: true,
    });

    await countInput.fill('6');
    await expect(countInput).toBeInViewport({ ratio: 1 });

    await chooseOption(page, {
      controlName: physicalConfigCopy.fullBackups,
      optionName: dictionary.backups.physical.fullBackupsPolicies.gfs,
    });
    await expect(dailyInput).toHaveValue('9');

    await chooseOption(page, {
      controlName: physicalConfigCopy.fullBackups,
      optionName: dictionary.backups.physical.fullBackupsPolicies.lastN,
    });
    await expect(countInput).toHaveValue('6');

    await expect(saveButton).toBeEnabled();
    await assertEnlargedTextReflow(page);

    await saveButton.click();
    await expect(saveButton).toBeHidden();

    expect(submittedConfigs).toHaveLength(1);
    expect(submittedConfigs[0].settings).toMatchObject({
      retention: {
        policy: 'CHAINS_AND_FULL_BACKUPS',
        chainsCount: 3,
        fullBackups: {
          policy: 'LAST_N',
          count: 6,
          gfsHours: 0,
          gfsDays: 0,
          gfsWeeks: 0,
          gfsMonths: 0,
          gfsYears: 0,
        },
      },
    });

    await expect(page.getByText(physicalConfigCopy.fullBackupsKept, { exact: true })).toBeVisible();
    await expect(
      page.getByText(physicalConfigCopy.fullBackupsCountValue.replace('{{count}}', '6'), {
        exact: true,
      }),
    ).toBeVisible();
  });
}
