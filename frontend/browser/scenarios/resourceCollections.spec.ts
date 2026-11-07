import { expect } from '@playwright/test';

import { DICTIONARIES } from '../../src/shared/i18n/dictionaries';
import { applicationFixtures, smtpNotifier, workspace } from '../applicationFixtures';
import { assertAccountReflow } from '../assertAccountReflow';
import { assertEnlargedTextReflow } from '../assertEnlargedTextReflow';
import { test } from '../browserTest';
import { enlargeText } from '../enlargeText';
import { apiResponse } from '../HttpFixture';
import { openAccount } from '../openAccount';
import { openWorkspaceScreen } from '../openWorkspaceScreen';

for (const domain of ['storages', 'notifiers'] as const) {
  test.describe(`${domain} collection interaction`, () => {
    test.use({
      // eslint-disable-next-line no-empty-pattern -- Playwright requires an object pattern for fixture dependencies
      httpFixtures: async ({}, provideFixtures) => {
        await provideFixtures(applicationFixtures());
      },
    });

    test(`keeps keyboard search and creation focus usable across narrow list and detail navigation`, async ({
      page,
      httpFixtures,
      boundary,
    }) => {
      const resources = Array.from({ length: 5 }, (_, index) => {
        const identity = {
          id: `archive-${index}`,
          workspaceId: workspace.id,
          name: `Archive ${index}`,
        };
        return domain === 'storages'
          ? { ...identity, settings: { type: 'LOCAL' as const } }
          : { ...smtpNotifier, ...identity };
      });
      httpFixtures.find(
        (fixture) => fixture.method === 'GET' && fixture.path === `/${domain}`,
      )!.body = { items: resources };
      const detailPath =
        domain === 'storages' ? '/storages/{storageId}' : '/notifiers/{notifierId}';
      const detail = httpFixtures.find(
        (fixture) => fixture.method === 'GET' && fixture.path === detailPath,
      );
      if (detail) detail.body = resources[0];
      else httpFixtures.push({ method: 'GET', path: detailPath, status: 200, body: resources[0] });

      const locale = 'ru';
      const dictionary = DICTIONARIES[locale];
      const labels = dictionary[domain];

      await openAccount({ page, locale });
      await openWorkspaceScreen({ page, locale, tab: domain });
      const search = page.getByRole('textbox', { name: labels.list.searchPlaceholder });
      await expect(
        page.getByRole('button', { name: resources[0].name, exact: true }),
      ).toBeVisible();
      const listRequests = () =>
        boundary.requests.filter(
          (request) => request.method === 'GET' && request.path === `/${domain}`,
        ).length;
      const requestsBeforeSearch = listRequests();
      await search.focus();
      await page.keyboard.type('aRcHiVe 4');

      await expect(
        page.getByRole('button', { name: resources[4].name, exact: true }),
      ).toBeVisible();
      await expect(page.getByRole('button', { name: resources[0].name, exact: true })).toHaveCount(
        0,
      );

      await search.press('ControlOrMeta+A');
      await page.keyboard.type('missing');

      await expect(
        page.getByText(labels.list.noSearchResults.replace('{{query}}', 'missing'), {
          exact: true,
        }),
      ).toBeVisible();

      await search.press('ControlOrMeta+A');
      await search.press('Backspace');

      await expect(
        page.getByRole('button', { name: resources[0].name, exact: true }),
      ).toBeVisible();
      expect(listRequests()).toBe(requestsBeforeSearch);

      const addLabel =
        domain === 'storages'
          ? dictionary.storages.list.addStorage
          : dictionary.notifiers.list.addNotifier;
      const add = page.getByRole('button', { name: addLabel, exact: true });
      await add.click();
      const creation = page.getByRole('dialog', { name: addLabel, exact: true });
      await expect(creation).toBeVisible();
      await creation
        .getByRole('textbox', { name: dictionary.common.fields.name, exact: true })
        .focus();
      await assertEnlargedTextReflow(page);

      await page.keyboard.press('Escape');
      await expect(creation).toBeHidden();
      await expect(add).toBeFocused();

      await page.setViewportSize({ width: 320, height: 1000 });
      const back = page.getByRole('button', { name: labels.list.backToList, exact: true });
      await back.click();
      await enlargeText(page.locator('main'));
      await assertEnlargedTextReflow(page);

      await assertAccountReflow(page.locator('main'));
      await expect(search).toBeInViewport();
      await search.fill('Archive 0');
      await page.getByRole('button', { name: resources[0].name, exact: true }).click();
      await expect(back).toBeVisible();
      await back.click();
      await expect(search).toHaveValue('Archive 0');
      await assertEnlargedTextReflow(page);

      await assertAccountReflow(page.locator('main'));
      await expect(search).toBeInViewport();
    });

    test(`contains enlarged transfer controls and restores focus after keyboard dismissal`, async ({
      page,
      httpFixtures,
    }) => {
      const storage = {
        id: 'archive-id',
        workspaceId: workspace.id,
        name: 'Archive',
        settings: { type: 'LOCAL' as const },
      };
      if (domain === 'storages') {
        httpFixtures.find((fixture) => fixture.path === '/storages')!.body = { items: [storage] };
        httpFixtures.push(
          apiResponse({ method: 'get', path: '/storages/{storageId}', status: 200, body: storage }),
        );
      }
      httpFixtures.find((fixture) => fixture.path === '/workspaces')!.body = {
        items: [workspace, { ...workspace, id: 'target-id', name: 'Regional archive destination' }],
      };
      const locale = 'fr';
      const dictionary = DICTIONARIES[locale];
      const labels = dictionary[domain];

      await openAccount({ page, locale });
      await openWorkspaceScreen({ page, locale, tab: domain });
      await page
        .getByRole('button', {
          name: domain === 'storages' ? storage.name : smtpNotifier.name,
          exact: true,
        })
        .click();
      await page.setViewportSize({ width: 320, height: 1000 });
      const invoker = page.getByRole('button', {
        name:
          domain === 'storages'
            ? dictionary.storages.transfer.title
            : dictionary.notifiers.transfer.submit,
        exact: true,
      });
      await invoker.click();
      const dialog = page.getByRole('dialog', { name: labels.transfer.title, exact: true });
      const target = dialog.getByRole('combobox', {
        name: labels.transfer.targetWorkspace,
        exact: true,
      });
      await expect(target).toBeVisible();
      await enlargeText(dialog);
      await assertEnlargedTextReflow(page);

      await assertAccountReflow(dialog);
      await target.focus();
      await target.press('ArrowDown');
      await page.getByRole('option', { name: 'Regional archive destination', exact: true }).click();
      await assertEnlargedTextReflow(page);

      await page.keyboard.press('Escape');
      await expect(dialog).toBeHidden();
      await expect(invoker).toBeFocused();
    });
  });
}
