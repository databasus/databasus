import { expect, type Locator } from '@playwright/test';

import { DICTIONARIES } from '../src/shared/i18n/dictionaries';
import { applicationFixtures, workspace } from './applicationFixtures';
import { test } from './browserTest';
import { apiResponse } from './HttpFixture';
import { openAccount } from './openAccount';
import { openWorkspaceScreen } from './openWorkspaceScreen';

export const assertSingleScrollOwner = async (container: Locator) => {
  const ownership = await container.evaluate((element) => {
    const competingScrollers: string[] = [];
    let ancestor = element.parentElement;

    while (ancestor) {
      if (
        /^(auto|scroll)$/.test(getComputedStyle(ancestor).overflowY) &&
        ancestor.scrollHeight > ancestor.clientHeight + 1
      )
        competingScrollers.push(ancestor.tagName);

      ancestor = ancestor.parentElement;
    }

    const outer = document.scrollingElement!;

    return {
      competingScrollers,
      hasPageOverflow: outer.scrollHeight > outer.clientHeight + 1,
      pageScrollTop: outer.scrollTop,
    };
  });

  expect(ownership).toEqual({ competingScrollers: [], hasPageOverflow: false, pageScrollTop: 0 });
};

const assertBoundedScroll = async ({
  container,
  finalControl,
}: {
  container: Locator;
  finalControl: Locator;
}) => {
  await expect
    .poll(() =>
      container.evaluate(
        (element) =>
          element.clientHeight > 0 &&
          element.scrollHeight > element.clientHeight + 1 &&
          /^(auto|scroll)$/.test(getComputedStyle(element).overflowY),
      ),
    )
    .toBe(true);
  await expect(finalControl).not.toBeInViewport();

  await finalControl.scrollIntoViewIfNeeded();
  await finalControl.focus();

  await expect(finalControl).toBeInViewport();
  await expect(finalControl).toBeFocused();
  await expect.poll(() => container.evaluate((element) => element.scrollTop)).toBeGreaterThan(0);

  await assertSingleScrollOwner(container);
};

export const registerResourceScrollScenarios = (resource: 'storages' | 'notifiers') => {
  test.describe(`${resource} scrolling`, () => {
    test.use({ httpFixtures: [applicationFixtures(), { scope: 'test' }] });

    for (const width of [320, 1280]) {
      test(`keeps overflowing lists and details independently scrollable when viewport width is ${width}`, async ({
        page,
        httpFixtures,
      }) => {
        const detailFixtures = Array.from({ length: 30 }, (_, index) => {
          const identity = {
            id: `resource-${index}`,
            workspaceId: workspace.id,
            name: `Archive destination ${String(index + 1).padStart(2, '0')}`,
          };

          return resource === 'storages'
            ? {
                ...apiResponse({
                  method: 'get',
                  path: '/storages/{storageId}',
                  status: 200,
                  body: {
                    ...identity,
                    settings: {
                      type: 'SFTP',
                      host: 'regional-archive-backup.'.repeat(6) + 'example.com',
                      port: 22,
                      username: 'backup',
                      path: '/archive/regional-backups'.repeat(12),
                      authMethods: ['PASSWORD', 'PRIVATE_KEY'],
                      isSkipHostKeyVerify: true,
                    },
                  },
                }),
                path: `/storages/${identity.id}`,
              }
            : {
                ...apiResponse({
                  method: 'get',
                  path: '/notifiers/{notifierId}',
                  status: 200,
                  body: {
                    ...identity,
                    lastSendError: null,
                    settings: {
                      type: 'EMAIL',
                      targetEmail: 'operations@example.com',
                      smtpHost: 'regional-archive-relay.'.repeat(6) + 'example.com',
                      smtpPort: 587,
                      smtpUser: 'regional-archive-operations-'.repeat(6),
                      from: `"${'Regional archive operations '.repeat(6)}" <backup@example.com>`,
                      security: 'STARTTLS',
                      isInsecureSkipVerify: true,
                    },
                  },
                }),
                path: `/notifiers/${identity.id}`,
              };
        });
        const listFixture = httpFixtures.find(
          (fixture) => fixture.method === 'GET' && fixture.path === `/${resource}`,
        )!;
        listFixture.body = { items: detailFixtures.map((fixture) => fixture.body) };
        const genericDetail =
          resource === 'storages' ? '/storages/{storageId}' : '/notifiers/{notifierId}';
        const genericIndex = httpFixtures.findIndex(
          (fixture) => fixture.method === 'GET' && fixture.path === genericDetail,
        );

        if (genericIndex >= 0) httpFixtures.splice(genericIndex, 1);

        httpFixtures.push(...detailFixtures);

        await page.setViewportSize({ width, height: width === 320 ? 560 : 360 });
        await openAccount({ page });
        await openWorkspaceScreen({ page, locale: 'en', tab: resource });
        const finalCard = page.getByRole('button', { name: 'Archive destination 30', exact: true });
        const listContainer = finalCard.locator(
          'xpath=ancestor::div[contains(@class,"overflow-y-auto")][1]',
        );

        await assertBoundedScroll({ container: listContainer, finalControl: finalCard });

        await finalCard.click();
        const finalDetailAction = page.getByRole('button', { name: 'Remove', exact: true });
        await expect(finalDetailAction).toBeVisible();

        const detailContainer = finalDetailAction.locator(
          'xpath=ancestor::div[contains(@class,"overflow-y-auto")][1]',
        );
        const listScrollTop =
          width === 1280 ? await listContainer.evaluate((element) => element.scrollTop) : undefined;

        await assertBoundedScroll({ container: detailContainer, finalControl: finalDetailAction });

        if (width === 1280) {
          expect(await listContainer.evaluate((element) => element.scrollTop)).toBe(listScrollTop);

          const finalNavigationAction = page.getByRole('button', {
            name: DICTIONARIES.en.app.navigation.users,
            exact: true,
          });
          const navigationContainer = finalNavigationAction.locator(
            'xpath=ancestor::div[contains(@class,"overflow-y-auto")][1]',
          );

          await assertBoundedScroll({
            container: navigationContainer,
            finalControl: finalNavigationAction,
          });
        } else {
          const back = page.getByRole('button', {
            name: DICTIONARIES.en[resource].list.backToList,
            exact: true,
          });

          await expect(back).toBeInViewport();

          await back.click();

          await expect(
            page.getByRole('button', { name: 'Archive destination 01', exact: true }),
          ).toBeInViewport();
        }
      });
    }
  });
};
