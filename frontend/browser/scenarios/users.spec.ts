import { expect } from '@playwright/test';

import { DICTIONARIES } from '../../src/shared/i18n/dictionaries';
import { accountFixtures } from '../accountFixtures';
import { assertAccountReflow } from '../assertAccountReflow';
import { assertEnlargedTextReflow } from '../assertEnlargedTextReflow';
import { test } from '../browserTest';
import { enlargeText } from '../enlargeText';
import { openAccount } from '../openAccount';
import { openAccountTab } from '../openAccountTab';
import { visualContexts } from '../visualContexts';

test.use({ httpFixtures: [accountFixtures(), { scope: 'test' }] });

test('keeps translated retry reachable with 200% text when the user list fails at 320px', async ({
  page,
}) => {
  await page.setViewportSize({ width: 320, height: 900 });
  let shouldRefuseList = true;
  await page.route(
    (url) => url.pathname === '/api/v1/users',
    async (route) => {
      if (!shouldRefuseList || route.request().method() !== 'GET') return route.fallback();
      await route.fulfill({
        status: 403,
        contentType: 'application/json',
        body: JSON.stringify({
          type: 'urn:databasus:problem:human_verification_failed',
          title: 'Refused',
          detail: 'Synthetic refusal',
          status: 403,
          code: 'human_verification_failed',
        }),
      });
    },
  );

  await openAccount({ page, locale: 'ru', theme: 'dark' });
  await openAccountTab({ page, locale: 'ru', tab: 'users' });
  const dictionary = DICTIONARIES.ru;
  const region = page.getByRole('main');

  await expect(
    region.getByText(dictionary.errors.humanVerificationFailed, { exact: true }),
  ).toBeVisible();
  await expect(region.getByText(dictionary.users.list.empty, { exact: true })).not.toBeVisible();

  await enlargeText(region);
  await assertEnlargedTextReflow(page);

  await assertAccountReflow(region);
  const retry = region.getByRole('button', { name: dictionary.common.actions.retry, exact: true });
  await retry.scrollIntoViewIfNeeded();
  await expect(retry).toBeInViewport();

  shouldRefuseList = false;
  await retry.focus();
  await retry.press('Enter');

  await expect(region.getByText('Member <literal>', { exact: true })).toBeVisible();
  await expect(
    region.getByText(dictionary.errors.humanVerificationFailed, { exact: true }),
  ).not.toBeVisible();
  await expect(retry).not.toBeVisible();
});

test('keeps the user drawer contained and restores its invoker after keyboard dismissal at 320px', async ({
  page,
}) => {
  await page.setViewportSize({ width: 320, height: 900 });
  await openAccount({ page, locale: 'fr', theme: 'dark' });
  await openAccountTab({ page, locale: 'fr', tab: 'users' });

  const dictionary = DICTIONARIES.fr;
  const audit = page.getByRole('button', {
    name: dictionary.users.list.viewAuditLogs,
    exact: true,
  });
  await audit.focus();
  await audit.press('Enter');

  const drawer = page.getByRole('dialog', { name: new RegExp(dictionary.users.auditLogs.title) });
  await expect(drawer.getByText('<literal> {{audit}}', { exact: true })).toBeVisible();
  expect(
    await drawer.evaluate(
      (element) => Math.round(element.getBoundingClientRect().width * 100) / 100,
    ),
  ).toBeLessThanOrEqual(320);
  await assertEnlargedTextReflow(page);

  await page.keyboard.press('Escape');
  await expect(drawer).not.toBeVisible();
  await expect(audit).toBeFocused();
  await assertEnlargedTextReflow(page);

  await assertAccountReflow(page.getByRole('main'));
});

test('keeps user actions and a translated refusal reachable when text is enlarged to 200% at 320px', async ({
  page,
}) => {
  await page.setViewportSize({ width: 320, height: 900 });
  await openAccount({ page, locale: 'ru', theme: 'dark' });
  await openAccountTab({ page, locale: 'ru', tab: 'users' });

  const dictionary = DICTIONARIES.ru;
  const region = page.getByRole('main');
  await region.getByRole('switch').click();
  await expect(
    region.getByText(dictionary.errors.humanVerificationFailed, { exact: true }),
  ).toBeVisible();

  await enlargeText(region);
  await assertEnlargedTextReflow(page);

  await assertAccountReflow(region);

  for (const action of [
    region.getByRole('switch'),
    region.getByRole('combobox'),
    region.getByRole('button', { name: dictionary.users.list.viewAuditLogs, exact: true }),
  ]) {
    await action.scrollIntoViewIfNeeded();
    await expect(action).toBeInViewport();
    await action.focus();
    await expect(action).toBeFocused();
  }
});

for (const context of visualContexts) {
  test(`restores focus after dismissing literal user audit details in ${context.locale} ${context.theme} ${context.width}`, async ({
    page,
  }) => {
    const dictionary = DICTIONARIES[context.locale];

    await page.setViewportSize({ width: context.width, height: 1000 });
    await openAccount({ page, ...context });
    await openAccountTab({ page, locale: context.locale, tab: 'users' });

    const action = page.getByRole('button', {
      name: dictionary.users.list.viewAuditLogs,
      exact: true,
    });

    await expect(action).toBeVisible();
    await assertEnlargedTextReflow(page);

    await action.focus();
    await page.keyboard.press('Enter');

    const drawer = page.getByRole('dialog', {
      name: dictionary.users.auditLogs.title,
    });

    await expect(drawer).toContainText('<literal> {{audit}}');
    await assertEnlargedTextReflow(page);

    await page.keyboard.press('Escape');

    await expect(drawer).toBeHidden();
    await expect(action).toBeFocused();
  });
}
