import { expect } from '@playwright/test';

import { DICTIONARIES } from '../src/shared/i18n/dictionaries';
import { applicationFixtures, workspace } from './applicationFixtures';
import { assertEnlargedTextReflow } from './assertEnlargedTextReflow';
import { test } from './browserTest';
import { apiResponse } from './HttpFixture';
import { openAccount } from './openAccount';
import { openWorkspaceScreen } from './openWorkspaceScreen';
import { visualContexts } from './visualContexts';

const localResource = {
  id: 'local-id',
  workspaceId: workspace.id,
  name: 'Local backups',
  settings: { type: 'LOCAL' as const },
};
export const registerResourceAccessScenarios = (resource: 'storages' | 'notifiers') => {
  test.describe(`${resource} access and refusal`, () => {
    test.use({
      httpFixtures: [
        [
          ...applicationFixtures().filter(
            (fixture) => !['/storages', '/storages/{storageId}'].includes(fixture.path),
          ),
          apiResponse({
            method: 'get',
            path: '/storages',
            status: 200,
            body: { items: [localResource] },
          }),
          apiResponse({
            method: 'get',
            path: '/storages/{storageId}',
            status: 200,
            body: localResource,
          }),
        ],
        { scope: 'test' },
      ],
    });

    test('retains saved testing when a viewer opens the resource', async ({
      page,
      httpFixtures,
    }) => {
      const userFixture = httpFixtures.find((fixture) => fixture.path === '/users/{userId}')!;
      userFixture.body = { ...(userFixture.body as object), role: 'MEMBER' };
      const workspacesFixture = httpFixtures.find((fixture) => fixture.path === '/workspaces')!;
      workspacesFixture.body = { items: [{ ...workspace, currentUserRole: 'WORKSPACE_VIEWER' }] };

      await openAccount({ page });
      await openWorkspaceScreen({ page, locale: 'en', tab: resource });

      await expect(
        page
          .getByText(resource === 'storages' ? localResource.name : 'Operations mail', {
            exact: true,
          })
          .last(),
      ).toBeVisible();
      await expect(
        page.getByRole('button', {
          name: resource === 'storages' ? 'Test connection' : 'Send test notification',
          exact: true,
        }),
      ).toBeVisible();
      await expect(page.getByRole('button', { name: 'Edit', exact: true })).toHaveCount(0);
      await expect(page.getByRole('button', { name: 'Remove', exact: true })).toHaveCount(0);
      await expect(
        page.getByRole('button', {
          name: resource === 'storages' ? 'Add storage' : 'Add notifier',
          exact: true,
        }),
      ).toHaveCount(0);
    });

    for (const context of visualContexts) {
      test(`retains the selected resource after deletion is refused in ${context.theme} ${context.locale} ${context.width}`, async ({
        page,
        httpFixtures,
      }) => {
        httpFixtures.find((fixture) => fixture.path === '/users/{userId}')!.body = {
          id: 'owner-id',
          email: 'owner@example.com',
          name: 'Owner',
          role: 'ADMIN',
          status: 'ACTIVE',
          createdAt: workspace.createdAt,
        };
        httpFixtures.find((fixture) => fixture.path === '/workspaces')!.body = {
          items: [workspace],
        };
        const path = resource === 'storages' ? '/storages/{storageId}' : '/notifiers/{notifierId}';
        const deletion = httpFixtures.find(
          (fixture) => fixture.method === 'DELETE' && fixture.path === path,
        );
        const refusal = {
          method: 'DELETE',
          path,
          status: 409,
          body: {
            type: 'urn:databasus:problem:in_use',
            title: 'In use',
            detail: 'In use',
            status: 409,
            code: 'in_use',
          },
        };

        if (deletion) Object.assign(deletion, refusal);
        else httpFixtures.push(refusal);

        const dictionary = DICTIONARIES[context.locale];
        await page.setViewportSize({ width: context.width, height: 1000 });
        await openAccount({ page, ...context });
        await openWorkspaceScreen({ page, locale: context.locale, tab: resource });
        if (context.width === 320)
          await page
            .getByRole('button', {
              name: resource === 'storages' ? localResource.name : 'Operations mail',
              exact: true,
            })
            .click();
        await page
          .getByRole('button', { name: dictionary.common.actions.remove, exact: true })
          .click();
        const confirmation = page.getByRole('dialog');
        const remove = confirmation.getByRole('button', {
          name: dictionary.common.actions.remove,
          exact: true,
        });
        const cancel = confirmation.getByRole('button', {
          name: dictionary.common.actions.cancel,
          exact: true,
        });
        const actionColors = await confirmation.evaluate((element) => {
          const probe = document.createElement('span');
          element.append(probe);

          const resolveColor = (value: string) => {
            probe.style.color = value;
            return getComputedStyle(probe).color;
          };

          const colors = {
            dangerFill: resolveColor('var(--app-error-fill)'),
            ordinaryFill: resolveColor('var(--app-surface)'),
            ordinaryText: resolveColor('var(--app-content)'),
            ordinaryBorder: resolveColor('var(--app-border)'),
          };

          probe.remove();

          return colors;
        });

        await expect(remove).toHaveCSS('background-color', actionColors.dangerFill);
        await expect(remove).toHaveCSS('color', 'rgb(255, 255, 255)');
        await expect(cancel).toHaveCSS('background-color', actionColors.ordinaryFill);
        await expect(cancel).toHaveCSS('color', actionColors.ordinaryText);
        await expect(cancel).toHaveCSS('border-top-color', actionColors.ordinaryBorder);
        await assertEnlargedTextReflow(page);

        await page
          .getByRole('dialog')
          .getByRole('button', { name: dictionary.common.actions.remove, exact: true })
          .click();

        await expect(page.getByRole('alert')).toHaveText(
          resource === 'storages'
            ? dictionary.storages.config.removeBlockedByBackups
            : dictionary.errors.inUse,
        );
        await expect(page.getByRole('dialog')).not.toBeVisible();
        await expect(
          page
            .getByText(resource === 'storages' ? localResource.name : 'Operations mail', {
              exact: true,
            })
            .last(),
        ).toBeVisible();
        await expect(
          page.getByRole('button', { name: dictionary.common.actions.remove, exact: true }),
        ).toBeVisible();
      });
    }
  });
};
