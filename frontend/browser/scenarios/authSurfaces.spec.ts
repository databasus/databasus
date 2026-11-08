import { expect } from '@playwright/test';

import { DICTIONARIES } from '../../src/shared/i18n/dictionaries';
import { accountFixtures } from '../accountFixtures';
import { assertAccountReflow } from '../assertAccountReflow';
import { assertEnlargedTextReflow } from '../assertEnlargedTextReflow';
import { test } from '../browserTest';
import { enlargeText } from '../enlargeText';
import { apiResponse } from '../HttpFixture';
import { openAccount } from '../openAccount';

for (const context of [
  { locale: 'en' as const, theme: 'light', width: 1280 },
  { locale: 'en' as const, theme: 'dark', width: 1280 },
  { locale: 'fr' as const, theme: 'light', width: 320 },
]) {
  test.describe(`${context.locale} ${context.theme} ${context.width}`, () => {
    test.use({
      // eslint-disable-next-line no-empty-pattern -- Playwright requires an object pattern for fixture dependencies
      httpFixtures: async ({}, provideFixtures) => {
        await provideFixtures(accountFixtures());
      },
    });
    test.beforeEach(async ({ page }) => {
      await page.setViewportSize({ width: context.width, height: 1000 });
      await page.addInitScript(() => {
        window.__RUNTIME_CONFIG__ = { IS_EMAIL_CONFIGURED: 'true' };
      });
    });

    test('preserves labelled sign-in and registration fields through validation', async ({
      page,
    }) => {
      const dictionary = DICTIONARIES[context.locale];
      await openAccount({ page, ...context, isSignedIn: false });
      await expect(
        page.getByRole('textbox', { name: dictionary.users.fields.yourEmail }),
      ).toBeVisible();
      await assertEnlargedTextReflow(page);

      await page
        .getByRole('button', {
          name: dictionary.users.signIn.noAccount.match(/<signUpLink>(.*?)<\/signUpLink>/)![1],
          exact: true,
        })
        .click();
      const name = page.getByRole('textbox', {
        name: dictionary.users.fields.yourName,
        exact: true,
      });
      await expect(name).toBeVisible();
      await page.getByRole('button', { name: dictionary.users.signUp.submit, exact: true }).click();
      await expect(
        page.getByText(dictionary.users.validation.nameRequired, { exact: true }),
      ).toBeVisible();
      await name.fill('Registration draft');
      await assertEnlargedTextReflow(page);

      if (context.width === 320) {
        await enlargeText(page.locator('body'));
        await assertAccountReflow(page.locator('body'));
        await expect(name).toHaveValue('Registration draft');
      }
    });

    test('keeps sign-in code correction and resend refusal on the same screen', async ({
      page,
      httpFixtures,
    }) => {
      const dictionary = DICTIONARIES[context.locale];
      const session = httpFixtures.find((fixture) => fixture.path === '/sessions')!;
      Object.assign(
        session,
        apiResponse({
          method: 'post',
          path: '/sessions',
          status: 200,
          body: {
            outcome: 'CODE_REQUIRED',
            pendingSignInId: 'pending-id',
            email: 'owner@example.com',
          },
        }),
      );
      httpFixtures.push(
        apiResponse({
          method: 'post',
          path: '/pending-sign-ins/{pendingSignInId}/verify',
          status: 400,
          body: {
            type: 'urn:databasus:problem:sign_in_code_incorrect',
            title: 'Incorrect code',
            detail: 'Incorrect code',
            status: 400,
            code: 'sign_in_code_incorrect',
          },
        }),
        apiResponse({
          method: 'post',
          path: '/pending-sign-ins/{pendingSignInId}/resend',
          status: 429,
          body: {
            type: 'urn:databasus:problem:sign_in_code_resent_too_soon',
            title: 'Too soon',
            detail: 'Too soon',
            status: 429,
            code: 'sign_in_code_resent_too_soon',
          },
        }),
      );
      await openAccount({ page, ...context, isSignedIn: false });
      await page
        .getByRole('textbox', { name: dictionary.users.fields.yourEmail })
        .fill('owner@example.com');
      await page
        .getByLabel(dictionary.common.fields.password, { exact: true })
        .fill('synthetic-password');
      await page.getByRole('button', { name: dictionary.users.signIn.submit, exact: true }).click();
      const code = page.getByRole('textbox', {
        name: dictionary.users.signInCode.code,
        exact: true,
      });
      await code.fill('123456');
      await page
        .getByRole('button', { name: dictionary.users.signInCode.submit, exact: true })
        .click();
      await expect(
        page.getByText(dictionary.errors.signInCodeIncorrect, { exact: true }),
      ).toBeVisible();
      await page
        .getByRole('button', { name: dictionary.users.signInCode.resend, exact: true })
        .click();
      await expect(
        page.getByText(dictionary.errors.signInCodeResentTooSoon, { exact: true }),
      ).toBeVisible();
      await assertEnlargedTextReflow(page);

      if (context.width === 320) {
        await enlargeText(page.locator('body'));
        await assertAccountReflow(page.locator('body'));
      }
      await expect(code).toHaveValue('123456');
    });

    test('retains the submitted email when reset advances to code and password validation', async ({
      page,
      httpFixtures,
    }) => {
      const dictionary = DICTIONARIES[context.locale];
      httpFixtures.push(
        apiResponse({ method: 'post', path: '/password-resets', status: 204, body: undefined }),
      );
      await openAccount({ page, ...context, isSignedIn: false });
      await page
        .getByRole('button', { name: dictionary.users.signIn.forgotPassword, exact: true })
        .click();
      await page
        .getByRole('textbox', { name: dictionary.users.fields.yourEmail })
        .fill('owner@example.com');
      await assertEnlargedTextReflow(page);

      await page
        .getByRole('button', { name: dictionary.users.requestPasswordReset.submit, exact: true })
        .click();
      const code = page.getByRole('textbox', {
        name: dictionary.users.resetPassword.code,
        exact: true,
      });
      await expect(code).toBeVisible();
      await expect(
        page.getByRole('textbox', { name: dictionary.users.fields.yourEmail }),
      ).toHaveValue('owner@example.com');
      await page
        .getByRole('button', { name: dictionary.users.resetPassword.submit, exact: true })
        .click();
      await expect(
        page.getByText(dictionary.users.resetPassword.invalidCode, { exact: true }),
      ).toBeVisible();
      await assertEnlargedTextReflow(page);

      if (context.width === 320) {
        await enlargeText(page.locator('body'));
        await assertAccountReflow(page.locator('body'));
      }
    });

    test('scrubs invalid callbacks and shows recoverable translated feedback', async ({ page }) => {
      const dictionary = DICTIONARIES[context.locale];
      await openAccount({ page, ...context, isSignedIn: false });
      await page.goto('/auth/callback?error=access_denied&state=obsolete');
      await expect(page.getByRole('alert')).toContainText(dictionary.app.oauthCallback.failedTitle);
      await expect(page).toHaveURL(/\/auth\/callback$/);
      await assertEnlargedTextReflow(page);

      await page.goto('/storages/google-oauth?state=obsolete');
      await expect(page.getByRole('alert')).toContainText(dictionary.app.oauthStorage.restart);
      await expect(page).toHaveURL(/\/storages\/google-oauth$/);
      await assertEnlargedTextReflow(page);
    });

    test('keeps the callback pending when a valid OAuth exchange has not answered', async ({
      page,
      httpFixtures,
      boundary,
    }) => {
      const dictionary = DICTIONARIES[context.locale];
      const state = 'a'.repeat(64);
      let releaseExchange = () => {};
      const exchange = new Promise<void>((resolve) => {
        releaseExchange = resolve;
      });
      httpFixtures.push(
        apiResponse({
          method: 'post',
          path: '/github-sessions',
          status: 403,
          body: {
            type: 'urn:databasus:problem:permission_denied',
            title: 'Forbidden',
            detail: 'Synthetic refusal',
            status: 403,
            code: 'permission_denied',
          },
        }),
      );
      await page.route('**/api/v1/github-sessions', async (route) => {
        expect(route.request().method()).toBe('POST');

        await exchange;
        await route.fallback();
      });

      try {
        await openAccount({ page, ...context, isSignedIn: false });
        await page.evaluate((state) => {
          sessionStorage.setItem(
            'databasus_oauth_sign_in_attempt',
            JSON.stringify({
              state,
              provider: 'github',
              createdAt: Date.now(),
              redirectUri: `${location.origin}/auth/callback`,
            }),
          );
        }, state);
        const exchangeStarted = page.waitForRequest(
          (request) =>
            request.method() === 'POST' &&
            new URL(request.url()).pathname === '/api/v1/github-sessions',
        );

        await page.goto(`/auth/callback?code=synthetic-code&state=${state}`);
        const request = await exchangeStarted;

        expect(request.postDataJSON()).toEqual({
          code: 'synthetic-code',
          redirectUri: new URL('/auth/callback', page.url()).href,
        });
        await expect(page.getByRole('status')).toBeVisible();
        await expect(
          page.getByText(dictionary.app.oauthCallback.completing, { exact: true }),
        ).toBeVisible();
        await expect(page.getByRole('alert')).toHaveCount(0);
        await expect(page).toHaveURL(/\/auth\/callback$/);
        expect(
          await page.evaluate(() => sessionStorage.getItem('databasus_oauth_sign_in_attempt')),
        ).toBeNull();

        if (context.width === 320) await enlargeText(page.locator('body'));

        await assertAccountReflow(page.locator('body'));
        const spinner = await page.getByRole('status').boundingBox();
        const label = await page
          .getByText(dictionary.app.oauthCallback.completing, { exact: true })
          .boundingBox();
        const spinnerSize = context.width === 320 ? 40 : 20;

        expect(spinner?.width).toBe(spinnerSize);
        expect(spinner?.height).toBe(spinnerSize);
        expect(label!.y - spinner!.y - spinner!.height).toBe(16);

        releaseExchange();

        await expect(page.getByRole('alert')).toContainText(dictionary.errors.permissionDenied);
        await expect(page.getByRole('status')).toHaveCount(0);
        expect(boundary.requests.filter((request) => request.path === '/github-sessions')).toEqual([
          {
            method: 'POST',
            path: '/github-sessions',
            body: {
              code: 'synthetic-code',
              redirectUri: new URL('/auth/callback', page.url()).href,
            },
          },
        ]);
      } finally {
        releaseExchange();
      }
    });
  });
}
