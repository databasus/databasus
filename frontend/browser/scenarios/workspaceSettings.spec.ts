import AxeBuilder from '@axe-core/playwright';
import { expect } from '@playwright/test';

import { DICTIONARIES } from '../../src/shared/i18n/dictionaries';
import { accountFixtures } from '../accountFixtures';
import { applicationFixtures, workspace } from '../applicationFixtures';
import { assertAccountReflow } from '../assertAccountReflow';
import { assertEnlargedTextReflow } from '../assertEnlargedTextReflow';
import { test } from '../browserTest';
import { enlargeText } from '../enlargeText';
import { apiResponse } from '../HttpFixture';
import { openAccount } from '../openAccount';
import { openWorkspaceScreen } from '../openWorkspaceScreen';
import { visualContexts } from '../visualContexts';

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
    await assertEnlargedTextReflow(page);

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
        .include('[role="dialog"]')
        .withTags(['wcag2a', 'wcag2aa'])
        .analyze();

      expect(accessibility.violations).toEqual([]);
    });
  }
});

for (const context of visualContexts) {
  test.describe(`Workspace dialogs ${context.locale} ${context.theme} ${context.width}`, () => {
    test.use({
      httpFixtures: [
        accountFixtures().concat([
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
                  id: 'colleague-id',
                  workspaceId: workspace.id,
                  userId: 'colleague-user',
                  name: 'Colleague',
                  email: 'colleague@example.com',
                  role: 'WORKSPACE_MEMBER',
                  isInvited: false,
                  createdAt: workspace.createdAt,
                },
              ],
            },
          }),
          apiResponse({
            method: 'post',
            path: '/workspace-members',
            status: 201,
            body: {
              outcome: 'INVITED',
              email: 'newcomer@example.com',
              id: 'invited-id',
              workspaceId: workspace.id,
              userId: 'invited-user',
              name: 'Newcomer',
              role: 'WORKSPACE_MEMBER',
              isInvited: true,
              createdAt: workspace.createdAt,
            },
          }),
        ]),
        { scope: 'test' },
      ],
    });

    test('retains workspace context through member and deletion dialogs', async ({ page }) => {
      const dictionary = DICTIONARIES[context.locale];

      await page.setViewportSize({ width: context.width, height: 1000 });
      await openAccount({ page, ...context });
      await openWorkspaceScreen({ page, locale: context.locale, tab: 'workspaceSettings' });

      await expect(page.getByText('colleague@example.com', { exact: true })).toBeVisible();
      await assertEnlargedTextReflow(page);

      const name = page.getByRole('textbox', { name: dictionary.workspaces.fields.name });

      await name.clear();
      await page
        .getByRole('button', { name: dictionary.workspaces.settings.saveChanges, exact: true })
        .click();

      await expect(name).toHaveAccessibleDescription(dictionary.workspaces.settings.nameRequired);
      await expect(name).toHaveAttribute('aria-invalid', 'true');
      await assertEnlargedTextReflow(page);

      await name.fill(workspace.name);

      await expect(name).not.toHaveAttribute('aria-invalid', 'true');
      await expect(name).not.toHaveAttribute('aria-describedby');

      await page.getByRole('button', { name: dictionary.workspaces.members.add.button }).click();
      const add = page.getByRole('dialog', {
        name: dictionary.workspaces.members.add.title,
        exact: true,
      });

      await add
        .getByRole('button', { name: dictionary.workspaces.members.add.submit, exact: true })
        .click();

      await expect(
        page.getByText(dictionary.workspaces.members.add.emailRequired, { exact: true }),
      ).toBeVisible();
      await assertEnlargedTextReflow(page);

      await add
        .getByRole('combobox', { name: dictionary.workspaces.members.fields.email })
        .fill('newcomer@example.com');
      await add
        .getByRole('button', { name: dictionary.workspaces.members.add.submit, exact: true })
        .click();

      const invited = page.getByRole('dialog', {
        name: dictionary.workspaces.members.invited.title,
        exact: true,
      });

      await expect(invited).toBeVisible();
      await assertEnlargedTextReflow(page);

      await invited
        .getByRole('button', { name: dictionary.common.actions.ok, exact: true })
        .click();

      await expect(invited).toBeHidden();

      await page
        .getByRole('button', { name: dictionary.workspaces.transferOwnership.button })
        .click();

      const transfer = page.getByRole('dialog', {
        name: dictionary.workspaces.transferOwnership.title,
        exact: true,
      });

      await expect(
        transfer.getByRole('combobox', { name: dictionary.workspaces.transferOwnership.newOwner }),
      ).toBeVisible();
      await assertEnlargedTextReflow(page);

      await transfer
        .getByRole('button', { name: dictionary.common.actions.cancel, exact: true })
        .focus();
      await page.keyboard.press('Escape');

      await expect(transfer).toBeHidden();

      await page
        .getByRole('button', { name: dictionary.workspaces.members.remove.tooltip, exact: true })
        .click();

      await expect(
        page.getByText(
          dictionary.workspaces.members.remove.question.replace(
            '{{email}}',
            'colleague@example.com',
          ),
          { exact: true },
        ),
      ).toBeVisible();
      await assertEnlargedTextReflow(page);

      await page.keyboard.press('Escape');
      await page
        .getByRole('button', { name: dictionary.workspaces.delete.submit, exact: true })
        .click();

      const deletion = page.getByRole('dialog', {
        name: dictionary.workspaces.delete.confirmation.title,
        exact: true,
      });

      await expect(deletion).toContainText(workspace.name);
      await assertEnlargedTextReflow(page);

      if (context.width === 320) {
        await enlargeText(deletion);
        await assertAccountReflow(deletion);
      }

      await page.keyboard.press('Escape');

      await expect(deletion).toBeHidden();
      await expect(page.getByText('<literal> {{audit}}', { exact: true })).toBeVisible();
      await assertEnlargedTextReflow(page);
    });
  });
}
