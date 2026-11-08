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

for (const context of [
  { locale: 'en' as const, theme: 'light' },
  { locale: 'ru' as const, theme: 'dark' },
]) {
  test(`keeps profile correction and primary actions reachable at 320px with 200% text in ${context.locale} ${context.theme}`, async ({
    page,
  }) => {
    await page.setViewportSize({ width: 320, height: 900 });
    await openAccount({ page, ...context });
    await openAccountTab({ page, locale: context.locale, tab: 'profile' });

    const dictionary = DICTIONARIES[context.locale];
    const region = page.getByRole('main');
    await region
      .getByRole('textbox', { name: dictionary.users.fields.name, exact: true })
      .fill('New profile draft');
    await region
      .getByRole('button', { name: dictionary.users.profile.saveChanges, exact: true })
      .click();
    await expect(
      region.getByText(dictionary.errors.humanVerificationFailed, { exact: true }),
    ).toBeVisible();

    await region.getByLabel(dictionary.common.fields.password, { exact: true }).fill('short');
    await region
      .getByLabel(dictionary.users.profile.changePassword.confirmPassword, { exact: true })
      .fill('password');
    const change = region.getByRole('button', {
      name: dictionary.users.profile.changePassword.submit,
      exact: true,
    });
    await change.click();

    await expect(
      region.getByText(dictionary.users.validation.passwordTooShort.replace('{{minLength}}', '8'), {
        exact: true,
      }),
    ).toBeVisible();

    await enlargeText(region);
    await assertEnlargedTextReflow(page);

    await assertAccountReflow(region);
    for (const name of [dictionary.users.profile.saveChanges, dictionary.users.profile.logout]) {
      const action = region.getByRole('button', { name, exact: true });
      await action.scrollIntoViewIfNeeded();
      await expect(action).toBeInViewport();
      await action.focus();
      await expect(action).toBeFocused();
    }

    await change.scrollIntoViewIfNeeded();
    await expect(change).toBeInViewport();
    await assertEnlargedTextReflow(page);

    await assertAccountReflow(region);
  });
}

test('retains newer profile input and a single pending save when locale and theme change', async ({
  page,
}) => {
  await openAccount({ page });
  await openAccountTab({ page, locale: 'en', tab: 'profile' });

  let releaseSave: () => void = () => {};
  const pending = new Promise<void>((resolve) => {
    releaseSave = resolve;
  });
  let saveCount = 0;

  await page.route('**/api/v1/users/owner-id', async (route) => {
    if (route.request().method() !== 'PUT') {
      await route.fallback();
      return;
    }
    saveCount++;
    await pending;
    await route.fulfill({
      status: 200,
      contentType: 'application/json',
      body: JSON.stringify({
        id: 'owner-id',
        name: 'Submitted profile',
        email: 'owner@example.com',
        role: 'ADMIN',
        status: 'ACTIVE',
        createdAt: '2026-10-01T00:00:00Z',
      }),
    });
  });

  try {
    await page.getByRole('textbox', { name: 'Name', exact: true }).fill('Submitted profile');
    await page.getByRole('button', { name: 'Save changes', exact: true }).click();
    await expect.poll(() => saveCount).toBe(1);
    await page.getByRole('textbox', { name: 'Name', exact: true }).fill('Newer retained profile');
    await page.getByRole('button', { name: /Language:/ }).click();
    await page
      .getByRole('menuitemradio', { name: DICTIONARIES.fr.language.nativeName, exact: true })
      .click();

    const dictionary = DICTIONARIES.fr;
    await page
      .getByRole('button', {
        name: dictionary.app.theme.title.replace('{{theme}}', dictionary.app.theme.light),
        exact: true,
      })
      .click();
    await page.getByRole('menuitemradio', { name: dictionary.app.theme.dark, exact: true }).click();

    await expect(
      page.getByRole('button', { name: dictionary.users.profile.saveChanges, exact: true }),
    ).toBeDisabled();
    expect(saveCount).toBe(1);
    releaseSave();
    await expect(page.getByText(dictionary.users.profile.updated, { exact: true })).toBeVisible();
    await expect(
      page.getByRole('textbox', { name: dictionary.users.fields.name, exact: true }),
    ).toHaveValue('Newer retained profile');
    await expect(
      page.getByRole('button', { name: dictionary.users.profile.saveChanges, exact: true }),
    ).toBeEnabled();
    await expect(page.locator('html')).toHaveClass(/dark/);
  } finally {
    releaseSave();
  }
});

test('keeps pending profile actions reachable when text is enlarged to 200% at 320px', async ({
  page,
}) => {
  await page.setViewportSize({ width: 320, height: 900 });
  await openAccount({ page, locale: 'ru', theme: 'dark' });
  await openAccountTab({ page, locale: 'ru', tab: 'profile' });
  const dictionary = DICTIONARIES.ru;

  let releaseSave: () => void = () => {};
  const pending = new Promise<void>((resolve) => {
    releaseSave = resolve;
  });
  let saveCount = 0;

  await page.route('**/api/v1/users/owner-id', async (route) => {
    if (route.request().method() !== 'PUT') {
      await route.fallback();
      return;
    }
    saveCount++;
    await pending;
    await route.fulfill({
      status: 200,
      contentType: 'application/json',
      body: JSON.stringify({
        id: 'owner-id',
        email: 'owner@example.com',
        name: 'Submitted narrow profile',
        role: 'ADMIN',
        status: 'ACTIVE',
        createdAt: '2026-10-01T00:00:00Z',
      }),
    });
  });

  try {
    const region = page.getByRole('main');
    const name = region.getByRole('textbox', { name: dictionary.users.fields.name, exact: true });
    await name.fill('Submitted narrow profile');
    const save = region.getByRole('button', {
      name: dictionary.users.profile.saveChanges,
      exact: true,
    });
    await save.click();
    await expect.poll(() => saveCount).toBe(1);

    await name.fill('Newer narrow profile');
    await enlargeText(region);
    await assertEnlargedTextReflow(page);

    await assertAccountReflow(region);
    await save.scrollIntoViewIfNeeded();
    await expect(save).toBeInViewport();
    await expect(save).toBeDisabled();

    const logout = region.getByRole('button', {
      name: dictionary.users.profile.logout,
      exact: true,
    });
    await logout.scrollIntoViewIfNeeded();
    await expect(logout).toBeInViewport();
    await logout.focus();
    await expect(logout).toBeFocused();

    expect(saveCount).toBe(1);
    releaseSave();
    await expect(page.getByText(dictionary.users.profile.updated, { exact: true })).toBeVisible();
    await expect(name).toHaveValue('Newer narrow profile');
    await expect(save).toBeEnabled();
    await assertEnlargedTextReflow(page);

    await assertAccountReflow(region);
  } finally {
    releaseSave();
  }
});

for (const context of visualContexts) {
  test(`retains the saved profile in ${context.locale} ${context.theme} ${context.width}`, async ({
    page,
  }) => {
    const dictionary = DICTIONARIES[context.locale];

    await page.setViewportSize({ width: context.width, height: 1000 });
    await openAccount({ page, ...context });
    await openAccountTab({ page, locale: context.locale, tab: 'profile' });

    await expect(
      page.getByRole('textbox', { name: dictionary.users.fields.name, exact: true }),
    ).toHaveValue('Owner');
    await assertEnlargedTextReflow(page);
  });
}
