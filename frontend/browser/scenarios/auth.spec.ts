import { expect } from '@playwright/test';

import { DICTIONARIES } from '../../src/shared/i18n/dictionaries';
import { accountFixtures } from '../accountFixtures';
import { assertAccountReflow } from '../assertAccountReflow';
import { test } from '../browserTest';
import { enlargeText } from '../enlargeText';
import { openAccount } from '../openAccount';

test.use({ httpFixtures: [accountFixtures(), { scope: 'test' }] });

test('preserves auth input and reflows its translated refusal with 200% text at 320px', async ({
  page,
}) => {
  await page.setViewportSize({ width: 320, height: 900 });
  await page.addInitScript(() => {
    window.__RUNTIME_CONFIG__ = {
      GITHUB_CLIENT_ID: 'synthetic-github-client',
      GOOGLE_CLIENT_ID: 'synthetic-google-client',
    };
  });

  await openAccount({ page, locale: 'es', theme: 'dark', isSignedIn: false });
  const dictionary = DICTIONARIES.es;
  const email = page.getByRole('textbox', { name: dictionary.users.fields.yourEmail, exact: true });
  const password = page.getByLabel(dictionary.common.fields.password, { exact: true });
  let navigationCount = 0;
  page.on('framenavigated', (frame) => {
    if (frame === page.mainFrame()) navigationCount++;
  });

  await email.fill('owner@example.com');
  await password.fill('synthetic-password');
  await page.getByRole('button', { name: dictionary.users.signIn.submit, exact: true }).click();

  await expect(
    page.getByText(dictionary.errors.humanVerificationFailed, { exact: true }),
  ).toBeVisible();
  await expect(email).toHaveValue('owner@example.com');
  await expect(password).toHaveValue('synthetic-password');
  expect(navigationCount).toBe(0);
  await enlargeText(page.locator('body'));

  for (const name of [
    dictionary.users.signIn.submit,
    dictionary.users.oauth.continueWithGithub,
    dictionary.users.oauth.continueWithGoogle,
  ]) {
    const action = page.getByRole('button', { name, exact: true });
    await action.scrollIntoViewIfNeeded();
    await expect(action).toBeInViewport();
  }

  await assertAccountReflow(page.locator('body'));
});
