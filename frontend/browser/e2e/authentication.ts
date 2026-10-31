import type { Page } from '@playwright/test';

export const openSignIn = async (page: Page) => {
  await page.route('https://api.github.com/repos/databasus/databasus', (route) =>
    route.fulfill({ json: { stargazers_count: 0 } }),
  );
  await page.route('https://api.github.com/repos/databasus/databasus/releases/latest', (route) =>
    route.fulfill({ json: { tag_name: 'dev' } }),
  );
  await page.goto('/');
};

export const signIn = async (page: Page) => {
  await page.getByPlaceholder('your@email.com').fill(process.env.E2E_EMAIL!);
  await page.getByPlaceholder('********', { exact: true }).fill(process.env.E2E_PASSWORD!);
  await page.getByRole('button', { name: 'Sign in', exact: true }).click();
};
