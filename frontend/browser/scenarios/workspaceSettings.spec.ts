import AxeBuilder from '@axe-core/playwright';
import { expect } from '@playwright/test';

import { DICTIONARIES } from '../../src/shared/i18n/dictionaries';
import { applicationFixtures, workspace } from '../applicationFixtures';
import { assertAccountReflow } from '../assertAccountReflow';
import { test } from '../browserTest';
import { enlargeText } from '../enlargeText';
import { apiResponse } from '../HttpFixture';
import { openAccount } from '../openAccount';
import { openWorkspaceScreen } from '../openWorkspaceScreen';

test.describe('Workspace recovery', () => {
  test.use({
    httpFixtures: [
      [
        ...applicationFixtures(),
        {
          method: 'GET',
          path: '/workspaces/{workspaceId}',
          status: 500,
          body: {
            type: 'urn:databasus:problem:internal',
            title: 'Internal',
            detail: 'Internal',
            status: 500,
            code: 'internal',
          },
        },
        apiResponse({
          method: 'get',
          path: '/workspace-members',
          status: 200,
          body: { items: [] },
        }),
        apiResponse({ method: 'get', path: '/audit-events', status: 200, body: { items: [] } }),
      ],
      { scope: 'test' },
    ],
  });

  test('preserves the name draft when failed workspace settings recover at narrow width', async ({
    page,
    httpFixtures,
  }) => {
    await openAccount({ page });
    await openWorkspaceScreen({ page, locale: 'en', tab: 'workspaceSettings' });
    const retry = page.getByRole('button', { name: 'Retry', exact: true });

    await expect(retry).toBeVisible();

    const fixture = httpFixtures.find((fixture) => fixture.path === '/workspaces/{workspaceId}')!;
    fixture.status = 200;
    fixture.body = workspace;

    await retry.click();

    const name = page.getByRole('textbox', { name: 'Workspace name', exact: true });
    await name.fill('Retained workspace draft');

    await page.setViewportSize({ width: 320, height: 1000 });
    const settings = page.getByRole('main');
    await enlargeText(settings);

    await assertAccountReflow(settings);
    await expect(name).toHaveValue('Retained workspace draft');
  });
});

test.describe('Workspace ownership warning', () => {
  test.use({
    httpFixtures: [
      [
        ...applicationFixtures(),
        apiResponse({
          method: 'get',
          path: '/workspaces/{workspaceId}',
          status: 200,
          body: workspace,
        }),
        apiResponse({
          method: 'get',
          path: '/workspace-members',
          status: 200,
          body: {
            items: [
              {
                id: 'member-id',
                workspaceId: workspace.id,
                userId: 'member-user',
                name: 'Member',
                email: 'member@example.com',
                role: 'WORKSPACE_MEMBER',
                isInvited: false,
                createdAt: workspace.createdAt,
              },
            ],
          },
        }),
        apiResponse({ method: 'get', path: '/audit-events', status: 200, body: { items: [] } }),
      ],
      { scope: 'test' },
    ],
  });

  for (const theme of ['light', 'dark']) {
    test(`keeps the ownership warning readable with ${theme} theme`, async ({ page }) => {
      await openAccount({ page, theme });
      await openWorkspaceScreen({ page, locale: 'en', tab: 'workspaceSettings' });
      await page.getByRole('button', { name: /Transfer ownership/ }).click();
      const dialog = page.getByRole('dialog');

      await expect(
        dialog.getByRole('combobox', {
          name: DICTIONARIES.en.workspaces.transferOwnership.newOwner,
        }),
      ).toBeVisible();

      await dialog.evaluate(async (element) => {
        await Promise.allSettled(
          element.getAnimations({ subtree: true }).map((animation) => animation.finished),
        );
      });

      const accessibility = await new AxeBuilder({ page })
        .disableRules(['color-contrast'])
        .include('.ant-modal-content')
        .withTags(['wcag2a', 'wcag2aa'])
        .analyze();

      expect(accessibility.violations).toEqual([]);
    });
  }
});
