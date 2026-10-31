import { expect, test } from '@playwright/test';

test('signs in and out when the real API owns the account', async ({ page }) => {
  const probe = process.env.DATABASUS_BROWSER_PROBE;
  await page.route('https://api.github.com/repos/databasus/databasus', (route) =>
    route.fulfill({ json: { stargazers_count: 0 } }),
  );
  await page.route('https://api.github.com/repos/databasus/databasus/releases/latest', (route) =>
    route.fulfill({ json: { tag_name: 'dev' } }),
  );
  await page.goto('/');
  await expect(page.getByRole('button', { name: 'Sign in', exact: true })).toBeVisible();
  await expect(page.getByRole('button', { name: 'Forgot password?', exact: true })).toBeVisible();
  if (probe === 'interrupt') {
    console.log('E2E_INTERRUPT_READY');
    await page.waitForEvent('close');
    return;
  }
  await page.getByPlaceholder('your@email.com').fill(process.env.E2E_EMAIL!);
  await page.getByPlaceholder('********', { exact: true }).fill(process.env.E2E_PASSWORD!);
  await page.getByRole('button', { name: 'Sign in', exact: true }).click();
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
  await page.getByRole('button', { name: /logout|sign out/i }).click();
  await expect(page.getByRole('button', { name: 'Sign in', exact: true })).toBeVisible();
});
