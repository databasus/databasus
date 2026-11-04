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

  for (const provider of [
    'email',
    'telegram',
    'webhook',
    'slack',
    'discord',
    'teams',
    'mattermost',
  ] as const) {
    test(`keeps ${provider} notifier controls accessible with ${theme} theme and enlarged narrow text`, async ({
      page,
    }) => {
      await openAccount({ page, locale, theme });
      await openWorkspaceScreen({ page, locale, tab: 'notifiers' });
      await page
        .getByRole('button', { name: dictionary.notifiers.list.addNotifier, exact: true })
        .click();

      const dialog = page.getByRole('dialog', {
        name: dictionary.notifiers.list.addNotifier,
        exact: true,
      });
      const type = dialog.getByRole('combobox', {
        name: dictionary.common.fields.type,
        exact: true,
      });

      if (provider !== 'telegram') {
        await type.focus();
        await type.press('ArrowDown');
        await page
          .getByRole('option', { name: dictionary.notifiers.types[provider], exact: true })
          .click();
      }

      const providerFieldLabels = {
        email: [
          dictionary.notifiers.email.targetEmail,
          dictionary.notifiers.email.smtpHost,
          dictionary.notifiers.email.smtpPort,
          dictionary.notifiers.email.smtpUser,
          dictionary.notifiers.email.smtpPassword,
          dictionary.notifiers.email.from,
        ],
        telegram: [dictionary.notifiers.fields.botToken, dictionary.notifiers.fields.targetChatId],
        webhook: [
          dictionary.notifiers.fields.webhookUrl,
          dictionary.notifiers.webhook.method,
          dictionary.notifiers.webhook.sendOn,
        ],
        slack: [dictionary.notifiers.fields.botToken, dictionary.notifiers.fields.targetChatId],
        discord: [dictionary.notifiers.discord.channelWebhookUrl],
        teams: [dictionary.notifiers.teams.powerAutomateUrl],
        mattermost: [
          dictionary.notifiers.mattermost.incomingWebhookUrl,
          dictionary.notifiers.mattermost.connectVia,
        ],
      }[provider];
      const getProviderControl = (label: string) =>
        dialog
          .getByLabel(label, { exact: true })
          .and(dialog.locator('input:not([type="radio"]), textarea'));
      const retainedField = getProviderControl(providerFieldLabels[0]);

      for (const label of providerFieldLabels) {
        await expect(getProviderControl(label)).toBeVisible();
      }

      await retainedField.fill('retained-provider-input');

      await dialog
        .getByRole('textbox', { name: dictionary.common.fields.name, exact: true })
        .fill('Retained notifier draft');
      await dialog.evaluate(async (element) => {
        await Promise.allSettled(
          element.getAnimations({ subtree: true }).map((animation) => animation.finished),
        );
      });

      const accessibility = await new AxeBuilder({ page })
        .disableRules(['color-contrast'])
        .include('.ant-modal-content')
        .withTags(['wcag2a', 'wcag2aa'])
        .analyze();

      expect(accessibility.violations).toEqual([]);

      await page.setViewportSize({ width: 320, height: 1000 });
      await enlargeText(dialog);

      await assertAccountReflow(dialog);
      await expect(
        dialog.getByRole('textbox', { name: dictionary.common.fields.name, exact: true }),
      ).toHaveValue('Retained notifier draft');
      await expect(retainedField).toHaveValue('retained-provider-input');
    });
  }
}

registerResourceAccessScenarios('notifiers');

registerResourceScrollScenarios('notifiers');
