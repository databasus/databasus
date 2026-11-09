import AxeBuilder from '@axe-core/playwright';
import { expect } from '@playwright/test';

import { DICTIONARIES } from '../../src/shared/i18n/dictionaries';
import { assertAccountReflow } from '../assertAccountReflow';
import { test } from '../browserTest';
import { enlargeText } from '../enlargeText';
import { openAccount } from '../openAccount';
import { openWorkspaceScreen } from '../openWorkspaceScreen';
import { registerResourceAccessScenarios } from '../resourceAccessScenarios';
import { registerResourceScrollScenarios } from '../resourceScrollScenarios';

for (const theme of ['light', 'dark']) {
  const locale = theme === 'light' ? 'fr' : 'ru';
  const dictionary = DICTIONARIES[locale];

  for (const provider of ['local', 's3', 'nas', 'azureBlob', 'ftp', 'sftp'] as const) {
    test(`keeps ${provider} storage controls accessible with ${theme} theme and enlarged narrow text`, async ({
      page,
    }) => {
      await openAccount({ page, locale, theme });
      await openWorkspaceScreen({ page, locale, tab: 'storages' });
      const invoker = page.getByRole('button', {
        name: dictionary.storages.list.addStorage,
        exact: true,
      });
      await invoker.click();

      const dialog = page.getByRole('dialog', {
        name: dictionary.storages.list.addStorage,
        exact: true,
      });
      const type = dialog.getByRole('combobox', {
        name: dictionary.common.fields.type,
        exact: true,
      });
      await type.focus();

      await expect(type).toBeFocused();

      await type.press('ArrowDown');
      await expect(page.getByRole('option')).toHaveText(
        ['local', 's3', 'nas', 'azureBlob', 'ftp', 'sftp'].map(
          (type) => dictionary.storages.types[type as keyof typeof dictionary.storages.types],
        ),
      );

      if (provider !== 'local') {
        await page.getByText(dictionary.storages.types[provider], { exact: true }).last().click();
      } else {
        await type.press('Escape');
      }

      const providerFieldLabels = {
        local: [],
        s3: [
          dictionary.storages.fields.s3Bucket,
          dictionary.storages.fields.region,
          dictionary.storages.fields.accessKey,
          dictionary.storages.fields.secretKey,
        ],
        nas: [
          dictionary.common.fields.host,
          dictionary.common.fields.port,
          dictionary.storages.fields.share,
          dictionary.common.fields.username,
          dictionary.common.fields.password,
        ],
        azureBlob: [
          dictionary.storages.fields.accountName,
          dictionary.storages.fields.accountKey,
          dictionary.storages.fields.containerName,
        ],
        ftp: [
          dictionary.common.fields.host,
          dictionary.common.fields.port,
          dictionary.common.fields.username,
          dictionary.common.fields.password,
        ],
        sftp: [
          dictionary.common.fields.host,
          dictionary.common.fields.port,
          dictionary.common.fields.username,
          dictionary.storages.fields.authMethod,
          dictionary.storages.fields.path,
        ],
      }[provider];
      const getProviderControl = (label: string) =>
        dialog
          .getByLabel(label, { exact: true })
          .and(dialog.locator('input:not([type="radio"]), textarea, button[role="combobox"]'));
      const retainedField = providerFieldLabels.length
        ? getProviderControl(providerFieldLabels[0])
        : undefined;

      for (const label of providerFieldLabels) {
        await expect(getProviderControl(label)).toBeVisible();
      }

      if (retainedField) await retainedField.fill('retained-provider-input');
      else
        await expect(
          dialog.getByText(dictionary.storages.edit.local.diskSpaceWarning, { exact: true }),
        ).toBeVisible();

      await dialog
        .getByRole('textbox', { name: dictionary.common.fields.name, exact: true })
        .fill('Retained provider draft');
      await dialog.evaluate(async (element) => {
        await Promise.allSettled(
          element.getAnimations({ subtree: true }).map((animation) => animation.finished),
        );
      });

      const accessibility = await new AxeBuilder({ page })
        .disableRules(['color-contrast'])
        .include('[role="dialog"]')
        .withTags(['wcag2a', 'wcag2aa'])
        .analyze();

      expect(accessibility.violations).toEqual([]);

      await page.setViewportSize({ width: 320, height: 1000 });
      await enlargeText(dialog);

      await assertAccountReflow(dialog);

      await expect(
        dialog.getByRole('textbox', { name: dictionary.common.fields.name, exact: true }),
      ).toHaveValue('Retained provider draft');

      if (retainedField) await expect(retainedField).toHaveValue('retained-provider-input');
      else
        await expect(
          dialog.getByText(dictionary.storages.edit.local.diskSpaceWarning, { exact: true }),
        ).toBeVisible();

      await page.keyboard.press('Escape');

      await expect(dialog).not.toBeVisible();
      await expect(invoker).toBeFocused();
    });
  }
}

registerResourceAccessScenarios('storages');

registerResourceScrollScenarios('storages');
