import { expect } from '@playwright/test';

import { DICTIONARIES } from '../../src/shared/i18n/dictionaries';
import { accountFixtures } from '../accountFixtures';
import { assertAccountReflow } from '../assertAccountReflow';
import { assertEnlargedTextReflow } from '../assertEnlargedTextReflow';
import { test } from '../browserTest';
import { enlargeText } from '../enlargeText';
import { apiResponse } from '../HttpFixture';
import { openAccount } from '../openAccount';

test.use({
  // eslint-disable-next-line no-empty-pattern -- Playwright requires an object pattern for fixture dependencies
  httpFixtures: async ({}, provideFixtures) => {
    await provideFixtures(accountFixtures());
  },
});

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
  await assertEnlargedTextReflow(page);

  await assertAccountReflow(page.locator('body'));
});

for (const refusal of [
  { code: 'email_not_found', key: 'emailNotFound', field: 'email', isEmailConfigured: true },
  {
    code: 'password_incorrect',
    key: 'passwordIncorrect',
    field: 'password',
    isEmailConfigured: true,
  },
  { code: 'registration_incomplete', key: 'registrationIncomplete', isEmailConfigured: true },
  {
    code: 'password_not_set_google_github',
    key: 'passwordNotSetGoogleGithub',
    isEmailConfigured: true,
  },
  { code: 'password_not_set', key: 'passwordNotSet', isEmailConfigured: true },
  { code: 'password_not_set', key: 'passwordNotSet', isEmailConfigured: false },
] as const) {
  test(`reflows ${refusal.code} with email delivery ${refusal.isEmailConfigured} at 320px and 200% text`, async ({
    page,
    httpFixtures,
  }) => {
    await page.setViewportSize({ width: 320, height: 900 });
    await page.addInitScript((isEmailConfigured) => {
      window.__RUNTIME_CONFIG__ = {
        GITHUB_CLIENT_ID: 'synthetic-github-client',
        GOOGLE_CLIENT_ID: 'synthetic-google-client',
        IS_EMAIL_CONFIGURED: String(isEmailConfigured),
      };
    }, refusal.isEmailConfigured);
    Object.assign(
      httpFixtures.find((fixture) => fixture.path === '/sessions')!,
      apiResponse({
        method: 'post',
        path: '/sessions',
        status: 401,
        body: {
          type: `urn:databasus:problem:${refusal.code}`,
          title: refusal.code,
          detail: refusal.code,
          status: 401,
          code: refusal.code,
        },
      }),
    );
    await openAccount({ page, locale: 'ru', theme: 'dark', isSignedIn: false });
    const dictionary = DICTIONARIES.ru;
    const email = page.getByRole('textbox', {
      name: dictionary.users.fields.yourEmail,
      exact: true,
    });
    const password = page.getByLabel(dictionary.common.fields.password, { exact: true });
    await email.fill('owner@example.com');
    await password.fill('synthetic-password');

    await page.getByRole('button', { name: dictionary.users.signIn.submit, exact: true }).click();

    const feedback = page.getByText(dictionary.errors[refusal.key], { exact: true });
    await expect(feedback).toBeVisible();
    await expect(email).toHaveValue('owner@example.com');
    await expect(password).toHaveValue('synthetic-password');
    if ('field' in refusal) {
      const input = refusal.field === 'email' ? email : password;
      await expect(input).toHaveAttribute('aria-invalid', 'true');
      const feedbackId = await feedback.getAttribute('id');
      expect(feedbackId).toBeTruthy();
      expect((await input.getAttribute('aria-describedby'))?.split(' ')).toContain(feedbackId);
      await expect(page.getByRole('alert')).toHaveCount(0);
    } else {
      await expect(page.getByRole('alert')).toContainText(dictionary.errors[refusal.key]);
    }
    if (refusal.code === 'password_not_set') {
      await expect(
        page.getByText(
          refusal.isEmailConfigured
            ? dictionary.users.signIn.passwordRecoveryHint
            : dictionary.users.signIn.contactAdministratorHint,
          { exact: true },
        ),
      ).toBeVisible();
      await expect(
        page.getByRole('button', { name: dictionary.users.signIn.setPassword, exact: true }),
      ).toHaveCount(refusal.isEmailConfigured ? 1 : 0);
    }

    await enlargeText(page.locator('body'));
    await assertEnlargedTextReflow(page);
    await assertAccountReflow(page.locator('body'));

    await email.fill('corrected@example.com');

    await expect(feedback).toHaveCount(0);
    await expect(page.getByRole('alert')).toHaveCount(0);
    await expect(password).toHaveValue('synthetic-password');
  });
}
