import { expect } from '@playwright/test';

import { DICTIONARIES } from '../../src/shared/i18n/dictionaries';
import { applicationFixtures } from '../applicationFixtures';
import { assertAccountReflow } from '../assertAccountReflow';
import { test } from '../browserTest';
import { enlargeText } from '../enlargeText';
import { apiResponse } from '../HttpFixture';
import { notifierProviders } from '../notifierProviderFixtures';
import { openAccount } from '../openAccount';
import { openWorkspaceScreen } from '../openWorkspaceScreen';
import { visualContexts } from '../visualContexts';

test.use({
  // eslint-disable-next-line no-empty-pattern -- Playwright requires an object pattern for fixture dependencies
  httpFixtures: async ({}, provideFixtures) => {
    await provideFixtures(applicationFixtures());
  },
});

for (const context of visualContexts) {
  for (const provider of notifierProviders) {
    test(`reads and edits saved ${provider.variant ?? provider.settings.type} notifier in ${context.theme} ${context.locale} ${context.width}`, async ({
      page,
      httpFixtures,
    }) => {
      const notifier = {
        id: 'notifier-id',
        workspaceId: 'workspace-id',
        name: 'Saved destination',
        settings: provider.settings,
        lastSendError: null,
      };
      httpFixtures.splice(
        0,
        httpFixtures.length,
        ...applicationFixtures().filter(
          (fixture) => !['/notifiers', '/notifiers/{notifierId}'].includes(fixture.path),
        ),
      );
      httpFixtures.push(
        apiResponse({
          method: 'get',
          path: '/notifiers',
          status: 200,
          body: { items: [notifier] },
        }),
        apiResponse({
          method: 'get',
          path: '/notifiers/{notifierId}',
          status: 200,
          body: notifier,
        }),
      );
      const dictionary = DICTIONARIES[context.locale];
      await page.setViewportSize({ width: context.width, height: 1000 });
      await openAccount({ page, ...context });
      await openWorkspaceScreen({ page, locale: context.locale, tab: 'notifiers' });
      if (context.width === 320)
        await page.getByRole('button', { name: notifier.name, exact: true }).click();
      await expect(
        page.getByRole('button', { name: dictionary.notifiers.testNotification.send, exact: true }),
      ).toBeVisible();

      await page
        .getByRole('button', { name: dictionary.common.actions.edit, exact: true })
        .last()
        .click();
      const cancelAction = page.getByRole('button', {
        name: dictionary.common.actions.cancel,
        exact: true,
      });
      await expect(cancelAction).toBeVisible();

      if (provider.settings.type === 'TELEGRAM') {
        await expect(
          page.getByRole('textbox', { name: dictionary.notifiers.telegram.botApiUrl, exact: true }),
        ).toHaveAccessibleDescription(dictionary.notifiers.telegram.botApiUrlTooltip);
        await expect(
          page.getByRole('textbox', { name: dictionary.notifiers.telegram.proxyUrl, exact: true }),
        ).toHaveAccessibleDescription(dictionary.notifiers.telegram.proxyUrlTooltip);
      }

      if (context.width === 320) {
        await enlargeText(page.getByRole('main'));

        await assertAccountReflow(page.getByRole('main'));
      }
      await cancelAction.focus();
      await expect(cancelAction).toBeFocused();
    });
  }
}
