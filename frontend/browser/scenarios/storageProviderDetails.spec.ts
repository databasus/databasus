import { expect } from '@playwright/test';

import { DICTIONARIES } from '../../src/shared/i18n/dictionaries';
import { applicationFixtures } from '../applicationFixtures';
import { assertAccountReflow } from '../assertAccountReflow';
import { test } from '../browserTest';
import { enlargeText } from '../enlargeText';
import { apiResponse } from '../HttpFixture';
import { openAccount } from '../openAccount';
import { openWorkspaceScreen } from '../openWorkspaceScreen';
import { storageProviders } from '../storageProviderFixtures';
import { visualContexts } from '../visualContexts';

test.use({
  // eslint-disable-next-line no-empty-pattern -- Playwright requires an object pattern for fixture dependencies
  httpFixtures: async ({}, provideFixtures) => {
    await provideFixtures(applicationFixtures());
  },
});

for (const context of visualContexts) {
  for (const provider of storageProviders) {
    test(`reads and edits saved ${provider.variant ?? provider.settings.type} storage in ${context.theme} ${context.locale} ${context.width}`, async ({
      page,
      httpFixtures,
    }) => {
      const storage = {
        id: 'archive-id',
        workspaceId: 'workspace-id',
        name: 'Saved archive',
        settings: provider.settings,
      };
      httpFixtures.splice(
        0,
        httpFixtures.length,
        ...applicationFixtures().filter((fixture) => fixture.path !== '/storages'),
      );
      httpFixtures.push(
        apiResponse({ method: 'get', path: '/storages', status: 200, body: { items: [storage] } }),
        apiResponse({ method: 'get', path: '/storages/{storageId}', status: 200, body: storage }),
      );
      const dictionary = DICTIONARIES[context.locale];
      await page.setViewportSize({ width: context.width, height: 1000 });
      await openAccount({ page, ...context });
      await openWorkspaceScreen({ page, locale: context.locale, tab: 'storages' });
      if (context.width === 320)
        await page.getByRole('button', { name: storage.name, exact: true }).click();
      await expect(
        page.getByRole('button', { name: dictionary.storages.actions.testConnection, exact: true }),
      ).toBeVisible();

      const editName = page
        .getByRole('button', { name: dictionary.common.actions.edit, exact: true })
        .first();

      await editName.click();

      const name = page.getByRole('textbox', { name: dictionary.common.fields.name, exact: true });

      await name.fill('Unsaved archive name');

      const cancelName = page.getByRole('button', {
        name: dictionary.common.actions.cancel,
        exact: true,
      });

      await expect(cancelName).toBeVisible();
      await cancelName.click();

      await expect(name).toBeHidden();

      await editName.click();

      await expect(name).toHaveValue(storage.name);

      await cancelName.click();

      await page
        .getByRole('button', { name: dictionary.common.actions.edit, exact: true })
        .last()
        .click();
      const testConnection = page.getByRole('button', {
        name: dictionary.storages.actions.testConnection,
        exact: true,
      });
      await expect(testConnection).toBeVisible();
      if (provider.settings.type === 'SFTP') {
        await expect(
          page.getByLabel(dictionary.common.fields.password, { exact: true }),
        ).not.toHaveAttribute('aria-invalid', 'true');
        await expect(
          page.getByRole('textbox', { name: dictionary.storages.fields.privateKey, exact: true }),
        ).not.toHaveAttribute('aria-invalid', 'true');
      }
      if (provider.settings.type === 'S3') {
        const storageClass = page.getByRole('combobox', {
          name: dictionary.storages.fields.storageClass,
          exact: true,
        });
        await expect(storageClass).toHaveValue('STANDARD_IA');
        expect.soft((await storageClass.boundingBox())!.width).toBeGreaterThan(100);
      }

      if (context.width === 320) {
        await enlargeText(page.getByRole('main'));
        if (provider.settings.type === 'AZURE_BLOB') {
          const field =
            provider.settings.authMethod === 'CONNECTION_STRING'
              ? dictionary.storages.fields.connection
              : dictionary.storages.fields.accountKey;
          const credential = page
            .locator('input[type="password"]')
            .and(page.getByLabel(field, { exact: true }));
          expect((await credential.boundingBox())!.width).toBeGreaterThan(120);
        }

        await assertAccountReflow(page.getByRole('main'));
      }
      await testConnection.focus();
      await expect(testConnection).toBeFocused();
    });
  }
}
