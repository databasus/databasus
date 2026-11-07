import { expect } from '@playwright/test';

import { DICTIONARIES } from '../../src/shared/i18n/dictionaries';
import { accountFixtures } from '../accountFixtures';
import { assertEnlargedTextReflow } from '../assertEnlargedTextReflow';
import { test } from '../browserTest';
import { openAccount } from '../openAccount';
import { visualContexts } from '../visualContexts';

test.use({
  // eslint-disable-next-line no-empty-pattern -- Playwright requires an object pattern for fixture dependencies
  httpFixtures: async ({}, provideFixtures) => {
    await provideFixtures(accountFixtures());
  },
});

for (const context of visualContexts) {
  test(`offers startup retry and the first-owner claim in ${context.theme} ${context.locale} ${context.width}`, async ({
    page,
    httpFixtures,
  }) => {
    const dictionary = DICTIONARIES[context.locale];
    const options = httpFixtures.find((fixture) => fixture.path === '/sign-in-options')!;
    options.status = 500;
    options.body = {
      type: 'urn:databasus:problem:internal',
      title: 'Internal',
      detail: 'Internal',
      status: 500,
      code: 'internal',
    };
    await page.setViewportSize({ width: context.width, height: 1000 });
    await openAccount({ page, ...context, isSignedIn: false });
    const retry = page.getByRole('button', { name: dictionary.common.actions.retry, exact: true });
    await expect(retry).toBeVisible();
    await assertEnlargedTextReflow(page);

    options.status = 200;
    options.body = { hasAnyUser: false };
    await retry.focus();
    await page.keyboard.press('Enter');
    await expect(
      page.getByText(dictionary.users.signUp.administersInstance, { exact: true }),
    ).toBeVisible();
    await assertEnlargedTextReflow(page);
  });

  test(`keeps workspace creation available when there are no workspaces in ${context.theme} ${context.locale} ${context.width}`, async ({
    page,
    httpFixtures,
  }) => {
    const dictionary = DICTIONARIES[context.locale];
    httpFixtures.find((fixture) => fixture.path === '/workspaces')!.body = { items: [] };
    await page.setViewportSize({ width: context.width, height: 1000 });
    await openAccount({ page, ...context });
    const create = page.getByRole('main').getByRole('button', {
      name: dictionary.app.workspaceSelection.create,
      exact: true,
    });
    await expect(create).toBeVisible();
    await assertEnlargedTextReflow(page);

    await create.click();
    const dialog = page.getByRole('dialog', {
      name: dictionary.workspaces.create.title,
      exact: true,
    });
    await expect(dialog.getByRole('textbox')).toHaveValue(dictionary.workspaces.create.defaultName);
    await assertEnlargedTextReflow(page);

    await dialog.getByRole('textbox').focus();
    await page.keyboard.press('Escape');
    await expect(dialog).toBeHidden();
    await expect(create).toBeFocused();
  });
}
