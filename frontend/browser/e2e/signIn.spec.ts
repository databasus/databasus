import { expect, test } from '@playwright/test';

import { openSignIn, signIn } from './authentication';

test('signs in and out when the real API owns the account', async ({ page }) => {
  const probe = process.env.DATABASUS_BROWSER_PROBE;
  await openSignIn(page);
  await expect(page.getByRole('button', { name: 'Sign in', exact: true })).toBeVisible();
  await expect(page.getByRole('button', { name: 'Forgot password?', exact: true })).toBeVisible();

  if (probe === 'interrupt') {
    console.log('E2E_INTERRUPT_READY');
    await page.waitForEvent('close');
    return;
  }

  await signIn(page);
  await page.getByRole('button', { name: 'Profile', exact: true }).click();
  await expect(page.getByRole('textbox', { name: 'Email', exact: true })).toHaveValue(
    process.env.E2E_EMAIL!,
  );

  if (probe === 'scenario') {
    await page.evaluate(() => {
      history.replaceState(null, '', '/?code=private-code&state=private-state');
      sessionStorage.setItem('probe-token', 'oauth-session');
      const diagnostic = document.createElement('p');
      diagnostic.textContent = 'Authorization: Bearer synthetic-secret-marker';
      document.body.appendChild(diagnostic);
    });
    throw new Error('deliberate E2E scenario failure after readiness');
  }

  await page.getByRole('button', { name: 'Profile', exact: true }).click();
  await page.getByRole('button', { name: /logout|sign out/i }).click();
  await expect(page.getByRole('button', { name: 'Sign in', exact: true })).toBeVisible();
});
