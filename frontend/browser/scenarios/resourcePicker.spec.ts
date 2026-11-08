import { expect } from '@playwright/test';

import { DICTIONARIES } from '../../src/shared/i18n/dictionaries';
import { smtpNotifier, workspace } from '../applicationFixtures';
import { test } from '../browserTest';
import { databaseFixtures } from '../databaseFixtures';
import { enlargeText } from '../enlargeText';
import { openAccount } from '../openAccount';
import { openWorkspaceScreen } from '../openWorkspaceScreen';

const contexts = [
  { locale: 'en' as const, theme: 'light', width: 1280, height: 720, isEnlarged: false },
  { locale: 'en' as const, theme: 'dark', width: 1280, height: 720, isEnlarged: false },
  { locale: 'fr' as const, theme: 'light', width: 320, height: 568, isEnlarged: true },
  { locale: 'fr' as const, theme: 'dark', width: 320, height: 568, isEnlarged: true },
];

test.use({
  // eslint-disable-next-line no-empty-pattern -- Playwright requires an object pattern for fixture dependencies
  httpFixtures: async ({}, provideFixtures) => {
    await provideFixtures(databaseFixtures());
  },
});

for (const resource of ['workspaces', 'storages', 'notifiers'] as const) {
  for (const context of contexts) {
    test(`keeps options${resource === 'storages' ? '' : ', search'} and creation reachable with many ${resource} in ${context.theme} ${context.locale} at ${context.width}px${context.isEnlarged ? ' and 200% text' : ''}`, async ({
      page,
      httpFixtures,
    }) => {
      const dictionary = DICTIONARIES[context.locale];
      const resources = Array.from({ length: 60 }, (_, index) => ({
        id: `resource-${index}`,
        name: `Resource ${String(index).padStart(2, '0')}`,
      }));
      const items =
        resource === 'workspaces'
          ? [workspace, ...resources.map((item) => ({ ...workspace, ...item }))]
          : resource === 'storages'
            ? resources.map((item) => ({
                ...item,
                workspaceId: workspace.id,
                settings: { type: 'LOCAL' as const },
              }))
            : resources.map((item) => ({ ...smtpNotifier, ...item }));
      httpFixtures.find(
        (fixture) => fixture.method === 'GET' && fixture.path === `/${resource}`,
      )!.body = { items };

      await page.setViewportSize({ width: context.width, height: context.height });
      await openAccount({ page, ...context });

      if (resource !== 'workspaces') {
        await openWorkspaceScreen({ page, locale: context.locale, tab: 'databases' });
        if (context.width === 320) await page.getByRole('link', { name: /^Orders/ }).click();
        await page
          .getByRole('tab', { name: dictionary.databases.tabs.config, exact: true })
          .click();
        const section =
          resource === 'storages'
            ? dictionary.databases.config.sections.backupConfig
            : dictionary.databases.config.sections.notifiers;

        await page
          .getByRole('button', {
            name: `${dictionary.common.actions.edit} ${section}`,
            exact: true,
          })
          .click();
      }

      const label =
        resource === 'workspaces'
          ? dictionary.app.workspaceSelection.selected
          : resource === 'storages'
            ? dictionary.backups.config.storage
            : dictionary.databases.notifiers.title;
      const createLabel =
        resource === 'workspaces'
          ? dictionary.app.workspaceSelection.createNew
          : resource === 'storages'
            ? dictionary.backups.config.createNewStorage
            : dictionary.databases.notifiers.createNew;
      const creationTitle =
        resource === 'workspaces'
          ? dictionary.workspaces.create.title
          : resource === 'storages'
            ? dictionary.backups.config.addStorageTitle
            : dictionary.databases.notifiers.addTitle;
      const target = page.getByRole('combobox', { name: label, exact: true });

      if (context.isEnlarged) await enlargeText(page.locator('body'));
      await target.click();

      const listbox = page.getByRole('listbox');
      const create = page.getByRole('button', { name: createLabel, exact: true });
      await expect(listbox).toBeVisible();
      if (context.isEnlarged) await enlargeText(page.locator('body'));

      for (const control of [listbox, create]) {
        await expect(control).toBeInViewport({ ratio: 1 });
        await expect
          .poll(() =>
            control.evaluate((element) => {
              const bounds = element.getBoundingClientRect();
              return bounds.top >= 0 && bounds.bottom <= innerHeight;
            }),
          )
          .toBe(true);
      }

      const search = page.getByRole('textbox', { name: label, exact: true });
      if (resource !== 'storages') await expect(search).toBeInViewport({ ratio: 1 });
      expect(await listbox.evaluate((element) => element.scrollHeight)).toBeGreaterThan(
        await listbox.evaluate((element) => element.clientHeight),
      );

      await listbox.hover();
      await page.mouse.wheel(0, await listbox.evaluate((element) => element.scrollHeight));

      await expect.poll(() => listbox.evaluate((element) => element.scrollTop)).toBeGreaterThan(0);
      const lastResource = resources.at(-1)!;
      const lastOption = page.getByRole('option', { name: lastResource.name, exact: true });
      await expect(lastOption).toBeInViewport({ ratio: 1 });
      await expect(create).toBeInViewport({ ratio: 1 });
      if (resource !== 'storages') {
        await expect(search).toBeInViewport({ ratio: 1 });
        await search.fill(lastResource.name);

        await expect(page.getByRole('option')).toHaveCount(1);
      }

      await lastOption.click();

      if (resource === 'notifiers') {
        await expect(
          page.getByRole('button', {
            name: `${dictionary.common.actions.remove} ${lastResource.name}`,
            exact: true,
          }),
        ).toBeVisible();
        await page.keyboard.press('Escape');
      } else {
        await expect(target).toHaveText(lastResource.name);
      }

      await target.click();
      if (context.isEnlarged) await enlargeText(page.locator('body'));

      await expect(create).toBeInViewport({ ratio: 1 });
      await create.click();

      const creation = page.getByRole('dialog', { name: creationTitle, exact: true });
      await expect(creation).toBeVisible();
      await expect
        .poll(() => creation.evaluate((element) => element.contains(document.activeElement)))
        .toBe(true);

      await page.keyboard.press('Escape');

      await expect(creation).toBeHidden();
      await expect(target).toBeFocused();
    });
  }
}
