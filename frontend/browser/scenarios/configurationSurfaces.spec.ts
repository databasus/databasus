import { expect } from '@playwright/test';

import { DICTIONARIES } from '../../src/shared/i18n/dictionaries';
import { assertAccountReflow } from '../assertAccountReflow';
import { assertEnlargedTextReflow } from '../assertEnlargedTextReflow';
import { test } from '../browserTest';
import { backupConfig, databaseFixtures } from '../databaseFixtures';
import { enlargeText } from '../enlargeText';
import { apiResponse } from '../HttpFixture';
import { openAccount } from '../openAccount';
import { openWorkspaceScreen } from '../openWorkspaceScreen';
import { visualContexts } from '../visualContexts';

test.use({
  // eslint-disable-next-line no-empty-pattern -- Playwright requires an object pattern for fixture dependencies
  httpFixtures: async ({}, provideFixtures) => {
    await provideFixtures(databaseFixtures());
  },
});

for (const context of visualContexts) {
  test.describe(`${context.theme} ${context.locale} ${context.width}`, () => {
    test.beforeEach(async ({ page }) => {
      await page.setViewportSize({ width: context.width, height: 1000 });
      await openAccount({ page, ...context });
      await openWorkspaceScreen({ page, locale: context.locale, tab: 'databases' });
      if (context.width === 320) await page.getByRole('link', { name: /^Orders/ }).click();
      await page
        .getByRole('tab', { name: DICTIONARIES[context.locale].databases.tabs.config, exact: true })
        .click();
    });

    test('keeps all schedule modes, retention and nested storage in the existing editor', async ({
      page,
    }) => {
      const dictionary = DICTIONARIES[context.locale];
      await expect(page.getByText('postgres.example.com', { exact: true })).toBeVisible();
      await assertEnlargedTextReflow(page);

      await page
        .getByRole('button', {
          name: `${dictionary.common.actions.edit} ${dictionary.databases.config.sections.backupConfig}`,
          exact: true,
        })
        .click();
      const interval = page.getByRole('combobox', {
        name: dictionary.backups.logical.config.interval,
        exact: true,
      });
      await expect(interval).toBeVisible();
      for (const schedule of ['hourly', 'daily', 'weekly', 'monthly', 'cron'] as const) {
        await interval.focus();
        await interval.press('ArrowDown');
        await page.getByText(dictionary.schedules.types[schedule], { exact: true }).last().click();
        await assertEnlargedTextReflow(page);
      }
      const cron = page.getByRole('textbox', {
        name: dictionary.schedules.cronExpression,
        exact: true,
      });
      await cron.fill('invalid');
      await expect(page.getByText(dictionary.schedules.invalidCron, { exact: true })).toBeVisible();
      await expect(
        page.getByRole('button', { name: dictionary.common.actions.save, exact: true }),
      ).toBeDisabled();
      await assertEnlargedTextReflow(page);

      await interval.focus();
      await interval.press('ArrowDown');
      await page.getByText(dictionary.schedules.types.daily, { exact: true }).last().click();
      const policy = page.getByRole('combobox', {
        name: dictionary.backups.logical.config.retentionPolicy,
        exact: true,
      });
      for (const retention of ['timePeriod', 'gfs', 'count'] as const) {
        await policy.focus();
        await policy.press('ArrowDown');
        await page
          .getByText(dictionary.backups.logical.retentionPolicyTypes[retention], { exact: true })
          .last()
          .click();
        await assertEnlargedTextReflow(page);
      }
      const storage = page.getByRole('combobox', {
        name: dictionary.backups.config.storage,
        exact: true,
      });
      await storage.focus();
      await storage.press('ArrowDown');
      await expect(storage).toHaveAttribute('aria-expanded', 'true');
      await page
        .getByRole('button', { name: dictionary.backups.config.createNewStorage, exact: true })
        .click();
      const nested = page.getByRole('dialog', {
        name: dictionary.backups.config.addStorageTitle,
        exact: true,
      });
      await expect(
        nested.getByRole('textbox', { name: dictionary.common.fields.name, exact: true }),
      ).toBeVisible();
      await assertEnlargedTextReflow(page);

      await nested
        .getByRole('textbox', { name: dictionary.common.fields.name, exact: true })
        .focus();
      await page.keyboard.press('Escape');
      await expect(nested).toBeHidden();
      await expect(storage).toBeFocused();
      if (context.width === 320) {
        await enlargeText(page.getByRole('main'));
        await interval.scrollIntoViewIfNeeded();

        await assertAccountReflow(page.getByRole('main'));
      }
    });

    test('keeps settings actions, nested notifier and healthcheck editing reachable', async ({
      page,
    }) => {
      const dictionary = DICTIONARIES[context.locale];
      await expect(page.getByText('postgres.example.com', { exact: true })).toBeVisible();
      await assertEnlargedTextReflow(page);

      await page
        .getByRole('button', {
          name: `${dictionary.common.actions.edit} ${dictionary.databases.config.sections.notifiers}`,
          exact: true,
        })
        .click();
      const notifiers = page.getByRole('combobox', {
        name: dictionary.databases.notifiers.title,
        exact: true,
      });
      await assertEnlargedTextReflow(page);

      await notifiers.focus();
      await notifiers.press('ArrowDown');
      await assertEnlargedTextReflow(page);

      await page
        .getByRole('button', { name: dictionary.databases.notifiers.createNew, exact: true })
        .click();
      const nested = page.getByRole('dialog', {
        name: dictionary.databases.notifiers.addTitle,
        exact: true,
      });
      await expect(nested).toBeVisible();
      await assertEnlargedTextReflow(page);

      await nested
        .getByRole('button', { name: dictionary.common.actions.close, exact: true })
        .focus();
      await page.keyboard.press('Escape');
      await expect(nested).toBeHidden();
      await page
        .getByRole('button', { name: dictionary.common.actions.cancel, exact: true })
        .click();
      await page
        .getByRole('button', {
          name: `${dictionary.common.actions.edit} ${dictionary.databases.config.sections.healthcheck}`,
          exact: true,
        })
        .click();
      await expect(
        page.getByRole('switch', { name: dictionary.healthcheck.config.enable, exact: true }),
      ).toBeVisible();
      await assertEnlargedTextReflow(page);

      await page
        .getByRole('button', { name: dictionary.common.actions.cancel, exact: true })
        .click();
      for (const action of ['copy', 'delete'] as const) {
        await page
          .getByRole('button', { name: dictionary.common.actions[action], exact: true })
          .click();
        const confirmation = page.getByRole('dialog', {
          name: dictionary.common.confirmation.title,
          exact: true,
        });
        await expect(confirmation).toBeVisible();

        await confirmation
          .getByRole('button', { name: dictionary.common.actions.cancel, exact: true })
          .focus();
        await page.keyboard.press('Escape');
        await expect(confirmation).toBeHidden();
      }
    });

    test('preserves the changed storage when its warning is canceled and submits once after confirmation', async ({
      page,
      httpFixtures,
      boundary,
    }) => {
      const dictionary = DICTIONARIES[context.locale];
      httpFixtures.find((fixture) => fixture.path === '/storages')!.body = {
        items: [
          {
            id: 'storage-id',
            workspaceId: 'workspace-id',
            name: 'Archive',
            settings: { type: 'LOCAL' },
          },
          {
            id: 'replacement-storage',
            workspaceId: 'workspace-id',
            name: 'Replacement archive',
            settings: { type: 'LOCAL' },
          },
        ],
      };
      httpFixtures.push(
        apiResponse({
          method: 'put',
          path: '/backup-configs/{databaseId}',
          status: 200,
          body: { ...backupConfig, storageId: 'replacement-storage' },
        }),
      );
      await page
        .getByRole('button', {
          name: `${dictionary.common.actions.edit} ${dictionary.databases.config.sections.backupConfig}`,
          exact: true,
        })
        .click();
      const storage = page.getByRole('combobox', {
        name: dictionary.backups.config.storage,
        exact: true,
      });
      await storage.focus();
      await storage.press('ArrowDown');
      await page.getByText('Replacement archive', { exact: true }).last().click();
      const save = page.getByRole('button', { name: dictionary.common.actions.save, exact: true });
      await save.click();
      const warning = page.getByRole('dialog', {
        name: dictionary.common.confirmation.title,
        exact: true,
      });
      await expect(warning).toContainText(dictionary.backups.storageSwitch.warning);
      await assertEnlargedTextReflow(page);

      await warning
        .getByRole('button', { name: dictionary.common.actions.cancel, exact: true })
        .click();
      await expect(warning).toBeHidden();
      await expect(save).toBeEnabled();
      expect(
        boundary.requests.filter(
          (request) => request.method === 'PUT' && request.path.startsWith('/backup-configs/'),
        ),
      ).toHaveLength(0);
      await save.click();
      await warning
        .getByRole('button', { name: dictionary.backups.storageSwitch.confirm, exact: true })
        .click();
      await expect(warning).toBeHidden();
      await expect(
        page.getByRole('button', {
          name: `${dictionary.common.actions.edit} ${dictionary.databases.config.sections.backupConfig}`,
          exact: true,
        }),
      ).toBeVisible();
      expect(
        boundary.requests.filter(
          (request) => request.method === 'PUT' && request.path.startsWith('/backup-configs/'),
        ),
      ).toHaveLength(1);
      await assertEnlargedTextReflow(page);
    });
  });
}
