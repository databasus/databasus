import { expect } from '@playwright/test';

import { DICTIONARIES } from '../../src/shared/i18n/dictionaries';
import { assertAccountReflow } from '../assertAccountReflow';
import { assertEnlargedTextReflow } from '../assertEnlargedTextReflow';
import { test } from '../browserTest';
import { backup, databaseFixtures } from '../databaseFixtures';
import { enlargeText } from '../enlargeText';
import { openAccount } from '../openAccount';
import { openWorkspaceScreen } from '../openWorkspaceScreen';
import { verifyBackupFilters } from '../verifyBackupFilters';
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
      await page.clock.setFixedTime(new Date('2026-10-18T12:00:00Z'));
      await page.setViewportSize({ width: context.width, height: 1000 });
      await openAccount({ page, ...context });
      await openWorkspaceScreen({ page, locale: context.locale, tab: 'databases' });
      if (context.width === 320) await page.getByRole('link', { name: /^Orders/ }).click();
    });

    test('keeps backup status selection and calendar keyboard dismissal on the list', async ({
      page,
    }) => {
      const dictionary = DICTIONARIES[context.locale];
      await expect(
        page
          .getByRole('button', {
            name: dictionary.backups.logical.list.actions.restore,
            exact: true,
          })
          .filter({ visible: true }),
      ).toBeVisible();
      await assertEnlargedTextReflow(page);

      await page
        .getByRole('button', { name: dictionary.backups.filters.toggle, exact: true })
        .click();
      await verifyBackupFilters({ page, locale: context.locale, backupKind: 'logical' });
      if (context.width === 320) {
        await enlargeText(page.getByRole('main'));

        await assertAccountReflow(page.getByRole('main'));
      }
    });

    test('opens the current backup confirmations and literal failure details', async ({ page }) => {
      const dictionary = DICTIONARIES[context.locale];
      const deletion = page
        .getByRole('button', { name: dictionary.backups.list.actions.deleteBackup, exact: true })
        .filter({ visible: true });
      await deletion.focus();
      await page.keyboard.press('Enter');
      const confirmation = page.getByRole('dialog', {
        name: dictionary.common.confirmation.title,
        exact: true,
      });
      await expect(confirmation).toContainText(dictionary.backups.logical.list.deleteConfirmation);
      await assertEnlargedTextReflow(page);

      await confirmation
        .getByRole('button', { name: dictionary.common.actions.cancel, exact: true })
        .focus();
      await page.keyboard.press('Escape');
      await expect(confirmation).toBeHidden();
      await expect(deletion).toBeFocused();
      await page
        .getByRole('button', {
          name: dictionary.backups.logical.list.actions.verifyRestore,
          exact: true,
        })
        .filter({ visible: true })
        .click();
      await expect(page.getByRole('dialog')).toBeVisible();
      await assertEnlargedTextReflow(page);

      await page.keyboard.press('Escape');
      await page
        .getByRole('button', { name: dictionary.status.logicalBackup.failed })
        .filter({ visible: true })
        .click();
      const error = page.getByRole('dialog', {
        name: dictionary.backups.list.errorDetailsTitle,
        exact: true,
      });
      await expect(error).toContainText('pg_dump: permission denied <literal>');
      await assertEnlargedTextReflow(page);

      await page.keyboard.press('Escape');
      await expect(error).toBeHidden();
    });

    for (const status of ['IN_PROGRESS', 'COMPLETED', 'FAILED', 'CANCELED'] as const) {
      test(`keeps ${status} restore actions and nested feedback in the restore dialog`, async ({
        page,
        httpFixtures,
      }) => {
        const dictionary = DICTIONARIES[context.locale];
        httpFixtures.find((fixture) => fixture.path === '/restores')!.body = {
          items: [
            {
              id: 'restore-id',
              backupId: backup.id,
              status,
              durationMs: 3100,
              createdAt: '2026-10-18T11:59:00Z',
              failMessage:
                status === 'FAILED'
                  ? 'pg_restore: must be owner of extension <literal>\n' +
                    'long_identifier_'.repeat(80)
                  : undefined,
            },
          ],
        };
        await page
          .getByRole('button', {
            name: dictionary.backups.logical.list.actions.restore,
            exact: true,
          })
          .filter({ visible: true })
          .click();
        const restore = page.getByRole('dialog', {
          name: dictionary.backups.list.restoreTitle,
          exact: true,
        });
        await expect(
          restore.getByText(
            dictionary.status.restore[
              status === 'IN_PROGRESS'
                ? 'inProgress'
                : status === 'COMPLETED'
                  ? 'completed'
                  : status === 'FAILED'
                    ? 'failed'
                    : 'canceled'
            ],
            { exact: true },
          ),
        ).toBeVisible();
        await assertEnlargedTextReflow(page);

        if (status === 'IN_PROGRESS') {
          await restore
            .getByRole('button', { name: dictionary.restores.card.cancelTooltip, exact: true })
            .click();
          const cancel = page.getByRole('dialog', {
            name: dictionary.common.confirmation.title,
            exact: true,
          });
          await expect(cancel).toContainText(dictionary.restores.cancelConfirmation.question);
          await assertEnlargedTextReflow(page);

          await cancel
            .getByRole('button', { name: dictionary.common.actions.cancel, exact: true })
            .focus();
          await page.keyboard.press('Escape');
          await expect(cancel).toBeHidden();
        } else if (status === 'FAILED') {
          await restore.getByRole('button', { name: dictionary.status.restore.failed }).click();
          const error = page.getByRole('dialog', {
            name: dictionary.restores.errorDetails.title,
            exact: true,
          });
          await expect(error).toContainText('pg_restore: must be owner of extension <literal>');
          await assertEnlargedTextReflow(page);

          await error.getByRole('button').first().focus();
          await page.keyboard.press('Escape');
          await expect(error).toBeHidden();
        } else {
          await restore
            .getByRole('button', { name: dictionary.restores.targetDatabase.select, exact: false })
            .click();
          await expect(
            restore.getByLabel(dictionary.common.fields.host, { exact: true }),
          ).toBeVisible();
          await assertEnlargedTextReflow(page);
        }
        if (context.width === 320) {
          await enlargeText(restore);

          await assertAccountReflow(restore);
        }
      });
    }
  });
}
