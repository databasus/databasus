import { expect } from '@playwright/test';

import { DICTIONARIES } from '../../src/shared/i18n/dictionaries';
import { accountFixtures } from '../accountFixtures';
import { assertAccountReflow } from '../assertAccountReflow';
import { assertSelectedNavigationEntry } from '../assertSelectedNavigationEntry';
import { assertThemeTextColor } from '../assertThemeTextColor';
import { test } from '../browserTest';
import { enlargeText } from '../enlargeText';
import { apiResponse } from '../HttpFixture';
import { openAccount } from '../openAccount';
import { openAccountTab } from '../openAccountTab';

test.use({ httpFixtures: [accountFixtures(), { scope: 'test' }] });

for (const usedPercent of [50, 90, 95, 96]) {
  test.describe(`disk usage at ${usedPercent} percent`, () => {
    const totalSpaceBytes = 100 * 1024 ** 3;
    const usedSpaceBytes = usedPercent * 1024 ** 3;
    const diskSummary = `${usedPercent.toFixed(1)} of 100.0 GB used (${usedPercent.toFixed(1)}%)`;
    const colorRole = usedPercent > 95 ? '--app-danger' : '--app-muted';
    test.use({
      httpFixtures: [
        accountFixtures().map((fixture) =>
          fixture.method === 'GET' && fixture.path === '/system/disk-usages'
            ? apiResponse({
                method: 'get',
                path: '/system/disk-usages',
                status: 200,
                body: {
                  usedSpaceBytes,
                  totalSpaceBytes,
                  freeSpaceBytes: totalSpaceBytes - usedSpaceBytes,
                },
              })
            : fixture,
        ),
        { scope: 'test' },
      ],
    });

    if (usedPercent === 90 || usedPercent === 96) {
      test('preserves warning color and header reflow when the theme changes with enlarged text', async ({
        page,
      }) => {
        await page.setViewportSize({ width: 769, height: 900 });
        await openAccount({ page });
        const header = page.getByRole('banner');
        const summary = header.getByText(diskSummary, { exact: true });
        await expect(summary).toBeVisible();
        await assertThemeTextColor({ text: summary, role: colorRole });

        await summary.hover();
        await expect(page.getByRole('tooltip')).toHaveText(DICTIONARIES.en.app.diskUsage.hint);
        await page.mouse.move(0, 0);
        await expect(page.getByRole('tooltip')).not.toBeVisible();

        await enlargeText(page.locator('body'));
        await assertAccountReflow(header);
        await expect(summary).toBeInViewport();

        const lightColor = await summary.evaluate((element) => getComputedStyle(element).color);
        await header
          .getByRole('button', {
            name: DICTIONARIES.en.app.theme.title.replace(
              '{{theme}}',
              DICTIONARIES.en.app.theme.light,
            ),
            exact: true,
          })
          .click();
        await page
          .getByRole('menuitem', { name: DICTIONARIES.en.app.theme.dark, exact: true })
          .click();

        await expect(page.locator('html')).toHaveClass(/dark/);
        await assertThemeTextColor({ text: summary, role: colorRole });
        await expect(summary).not.toHaveCSS('color', lightColor);
        await expect(summary).toHaveText(diskSummary);
        await assertAccountReflow(header);
        await expect(summary).toBeInViewport();
      });
    }

    if (usedPercent !== 90) {
      for (const theme of ['light', 'dark']) {
        test(`keeps mobile disk information available in the ${theme} theme`, async ({ page }) => {
          await page.setViewportSize({ width: 320, height: 900 });
          await openAccount({ page, theme });
          await page
            .getByRole('button', { name: DICTIONARIES.en.app.navigation.menu, exact: true })
            .click();

          const sidebar = page.getByRole('dialog');
          const summary = sidebar.getByText(diskSummary, { exact: true });
          await expect(summary).toBeVisible();
          await assertThemeTextColor({ text: summary, role: colorRole });
          await expect(
            page.getByRole('banner').getByText(diskSummary, { exact: true }),
          ).toHaveCount(0);
        });
      }
    }
  });
}

test('keeps desktop header actions reachable when the entire shell uses 200% text just above the mobile breakpoint', async ({
  page,
}) => {
  await page.setViewportSize({ width: 769, height: 900 });
  await openAccount({ page, locale: 'ru', theme: 'dark' });
  await openAccountTab({ page, locale: 'ru', tab: 'profile' });

  const dictionary = DICTIONARIES.ru;
  const name = page.getByRole('textbox', { name: dictionary.users.fields.name, exact: true });
  await name.fill('Retained desktop shell draft');
  await enlargeText(page.locator('body'));
  await assertAccountReflow(page.locator('body'));

  const header = page.getByRole('banner');
  for (const action of [
    header.getByRole('link', { name: dictionary.app.navigation.docs, exact: true }),
    header.getByRole('link', { name: dictionary.app.navigation.community, exact: true }),
    header.getByRole('link', { name: dictionary.app.starButton.ariaLabel, exact: true }),
    header.getByRole('button', {
      name: dictionary.app.languageControl.title.replace(
        '{{language}}',
        dictionary.language.nativeName,
      ),
      exact: true,
    }),
  ]) {
    await expect(action).toBeInViewport();
    await action.focus();
    await expect(action).toBeFocused();
  }

  await expect(name).toHaveValue('Retained desktop shell draft');
});

test('keeps workspace creation and navigation reachable when the entire shell uses 200% text at 320px', async ({
  page,
}) => {
  await page.setViewportSize({ width: 320, height: 900 });
  await openAccount({ page, locale: 'ru', theme: 'dark' });
  await openAccountTab({ page, locale: 'ru', tab: 'profile' });

  const dictionary = DICTIONARIES.ru;
  const name = page.getByRole('textbox', { name: dictionary.users.fields.name, exact: true });
  await name.fill('Retained shell draft');
  const shell = page.locator('body');
  await enlargeText(shell);
  await assertAccountReflow(shell);

  const workspaceSelector = page.getByRole('combobox', {
    name: dictionary.app.workspaceSelection.selected,
  });
  await workspaceSelector.focus();
  await workspaceSelector.press('ArrowDown');
  await workspaceSelector.press('Tab');

  const create = page.getByRole('button', {
    name: dictionary.app.workspaceSelection.createNew,
    exact: true,
  });
  await create.scrollIntoViewIfNeeded();
  await expect(create).toBeInViewport();
  await expect(create).toBeFocused();
  await create.press('Enter');

  const creation = page.getByRole('dialog', {
    name: dictionary.workspaces.create.title,
    exact: true,
  });
  await expect(creation).toBeVisible();
  await expect
    .poll(() => creation.evaluate((dialog) => dialog.contains(document.activeElement)))
    .toBe(true);

  await page.keyboard.press('Escape');
  await expect(creation).not.toBeVisible();
  await expect(workspaceSelector).toBeFocused();

  const menu = page.getByRole('button', { name: dictionary.app.navigation.menu, exact: true });
  await menu.focus();
  await menu.press('Enter');
  await expect(page.getByRole('dialog')).toBeVisible();
  await expect(name).toHaveValue('Retained shell draft');
});

test('keeps navigation reachable when the viewport crosses the 768px boundary', async ({
  page,
}) => {
  await page.setViewportSize({ width: 768, height: 900 });
  await openAccount({ page });

  const dictionary = DICTIONARIES.en;
  const menu = page.getByRole('button', { name: dictionary.app.navigation.menu, exact: true });
  await expect(menu).toBeVisible();
  await openAccountTab({ page, locale: 'en', tab: 'profile' });
  const name = page.getByRole('textbox', { name: dictionary.users.fields.name, exact: true });
  await name.fill('Retained boundary draft');

  await page.setViewportSize({ width: 769, height: 900 });
  await expect(menu).not.toBeVisible();
  await expect(
    page.getByRole('button', { name: dictionary.app.navigation.users, exact: true }),
  ).toBeVisible();
  await expect(name).toHaveValue('Retained boundary draft');

  await page.setViewportSize({ width: 768, height: 900 });
  await expect(menu).toBeVisible();
  await openAccountTab({ page, locale: 'en', tab: 'users' });
  await expect(
    page.getByRole('heading', { name: dictionary.users.list.title, exact: true }),
  ).toBeVisible();
});

for (const theme of ['light', 'dark']) {
  test(`shows the selected section on the sidebar in the ${theme} theme`, async ({ page }) => {
    await page.setViewportSize({ width: 769, height: 900 });
    await openAccount({ page, theme });

    const navigation = DICTIONARIES.en.app.navigation;
    const profile = page.getByRole('button', { name: navigation.profile, exact: true });
    const users = page.getByRole('button', { name: navigation.users, exact: true });

    await profile.click();
    await page.mouse.move(0, 0);
    await assertSelectedNavigationEntry({ entry: profile, iconName: 'profile' });
    await expect(page.locator('[aria-current="page"]')).toHaveCount(1);

    await users.click();
    await page.mouse.move(0, 0);
    await assertSelectedNavigationEntry({ entry: users, iconName: 'user-card' });
    await expect(profile).not.toHaveAttribute('aria-current');
    await expect(page.locator('[aria-current="page"]')).toHaveCount(1);
  });

  test(`shows the selected section in the narrow menu in the ${theme} theme`, async ({ page }) => {
    await page.setViewportSize({ width: 320, height: 900 });
    await openAccount({ page, theme });

    const navigation = DICTIONARIES.en.app.navigation;
    const menu = page.getByRole('button', { name: navigation.menu, exact: true });
    const profile = page.getByRole('button', { name: navigation.profile, exact: true });
    const users = page.getByRole('button', { name: navigation.users, exact: true });

    await openAccountTab({ page, locale: 'en', tab: 'profile' });
    await menu.click();
    await page.mouse.move(0, 0);
    await assertSelectedNavigationEntry({
      entry: profile,
      iconName: 'profile',
      label: profile.getByText(navigation.profile, { exact: true }),
    });
    await expect(page.locator('[aria-current="page"]')).toHaveCount(1);

    await users.click();
    await expect(menu).toBeFocused();
    await menu.click();
    await page.mouse.move(0, 0);
    await assertSelectedNavigationEntry({
      entry: users,
      iconName: 'user-card',
      label: users.getByText(navigation.users, { exact: true }),
    });
    await expect(profile).not.toHaveAttribute('aria-current');
    await expect(page.locator('[aria-current="page"]')).toHaveCount(1);
  });
}

test('retains the profile draft and sidebar when language is selected with the keyboard at 320px', async ({
  page,
}) => {
  await page.setViewportSize({ width: 320, height: 900 });
  await openAccount({ page });
  await openAccountTab({ page, locale: 'en', tab: 'profile' });
  await page.getByRole('textbox', { name: 'Name', exact: true }).fill('Retained narrow draft');

  const menu = page.getByRole('button', { name: DICTIONARIES.en.app.navigation.menu, exact: true });
  await menu.click();
  const sidebar = page.getByRole('dialog');
  await sidebar.getByRole('button', { name: /Language:/ }).focus();
  await page.keyboard.press('ArrowDown');
  await expect
    .poll(() => page.getByRole('menu').evaluate((menu) => menu.contains(document.activeElement)))
    .toBe(true);

  await page.keyboard.press('Home');
  await page.keyboard.press('ArrowDown');
  await expect(
    page.getByRole('menuitem', { name: DICTIONARIES.ru.language.nativeName, exact: true }),
  ).toBeFocused();
  await page.keyboard.press('Enter');
  const language = sidebar.getByRole('button', {
    name: DICTIONARIES.ru.app.languageControl.title.replace(
      '{{language}}',
      DICTIONARIES.ru.language.nativeName,
    ),
    exact: true,
  });

  await expect(sidebar).toBeVisible();
  await expect(
    sidebar.getByRole('button', { name: DICTIONARIES.ru.app.navigation.profile, exact: true }),
  ).toBeVisible();
  await expect(language).toBeFocused();
  await language.press('ArrowDown');
  await expect
    .poll(() => page.getByRole('menu').evaluate((menu) => menu.contains(document.activeElement)))
    .toBe(true);

  await page.keyboard.press('Escape');
  await expect(sidebar).toBeVisible();
  await expect(language).toBeFocused();
  await language.press('Escape');
  await expect(sidebar).not.toBeVisible();

  await expect(
    page.getByRole('button', { name: DICTIONARIES.ru.app.navigation.menu, exact: true }),
  ).toBeFocused();
  await expect(
    page.getByRole('textbox', { name: DICTIONARIES.ru.users.fields.name, exact: true }),
  ).toHaveValue('Retained narrow draft');
});
