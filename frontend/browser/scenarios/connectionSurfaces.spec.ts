import { expect } from '@playwright/test';

import { DICTIONARIES } from '../../src/shared/i18n/dictionaries';
import { applicationFixtures } from '../applicationFixtures';
import { assertAccountReflow } from '../assertAccountReflow';
import { assertEnlargedTextReflow } from '../assertEnlargedTextReflow';
import { assertScreenshot } from '../assertScreenshot';
import { test } from '../browserTest';
import { enlargeText } from '../enlargeText';
import { apiResponse } from '../HttpFixture';
import { openAccount } from '../openAccount';
import { openWorkspaceScreen } from '../openWorkspaceScreen';
import { visualContexts } from '../visualContexts';

const engines = [
  {
    connectionType: 'postgresql-logical',
    engine: 'postgresql' as const,
    uri: 'postgres://backup:synthetic-password@postgres.example.com:5432/orders',
    isPhysical: false,
  },
  {
    connectionType: 'postgresql-physical',
    engine: 'postgresql' as const,
    uri: 'postgres://backup:synthetic-password@postgres.example.com:5432/orders',
    isPhysical: true,
  },
  {
    connectionType: 'mysql',
    engine: 'mysql' as const,
    uri: 'mysql://backup:synthetic-password@mysql.example.com:3306/orders',
    isPhysical: false,
  },
  {
    connectionType: 'mariadb',
    engine: 'mariadb' as const,
    uri: 'mariadb://backup:synthetic-password@mariadb.example.com:3306/orders',
    isPhysical: false,
  },
  {
    connectionType: 'mongodb',
    engine: 'mongodb' as const,
    uri: 'mongodb://backup:synthetic-password@mongo.example.com:27017/orders',
    isPhysical: false,
  },
];

test.use({
  // eslint-disable-next-line no-empty-pattern -- Playwright requires an object pattern for fixture dependencies
  httpFixtures: async ({}, provideFixtures) => {
    await provideFixtures(
      applicationFixtures().concat([
        apiResponse({
          method: 'post',
          path: '/database-tests',
          status: 200,
          body: { isConnected: true },
        }),
        apiResponse({
          method: 'post',
          path: '/limited-user-checks',
          status: 200,
          body: {
            isAdvisable: true,
            privileges: ['CREATE', 'UPDATE', 'DELETE', 'ALTER', 'DROP', 'TRUNCATE', 'INSERT'],
          },
        }),
      ]),
    );
  },
});

for (const context of visualContexts) {
  for (const engine of engines) {
    test(`keeps ${engine.connectionType} connection and nested paste usable in ${context.theme} ${context.locale} ${context.width}`, async ({
      page,
    }) => {
      const dictionary = DICTIONARIES[context.locale];
      await page.setViewportSize({ width: context.width, height: 1000 });
      await page.addInitScript(() =>
        Object.defineProperty(navigator, 'clipboard', { configurable: true, value: undefined }),
      );
      await openAccount({ page, ...context });
      await openWorkspaceScreen({ page, locale: context.locale, tab: 'databases' });
      await page
        .getByRole('button', { name: dictionary.databases.list.addDatabase, exact: true })
        .click();
      const wizard = page.getByRole('dialog', {
        name: dictionary.databases.create.title,
        exact: true,
      });
      await expect
        .poll(() => wizard.evaluate((dialog) => dialog.contains(document.activeElement)))
        .toBe(true);
      await wizard
        .getByRole('textbox', { name: dictionary.common.fields.name, exact: true })
        .fill('Orders');
      const engineCard = wizard.getByRole('button', {
        name: dictionary.databases.types[engine.engine],
        exact: true,
      });
      await engineCard.focus();
      await expect(engineCard).toBeFocused();
      await page.keyboard.press('Enter');
      await expect(engineCard).toHaveAttribute('aria-pressed', 'true');
      await assertEnlargedTextReflow(page);

      await wizard
        .getByRole('button', { name: dictionary.common.actions.continue, exact: true })
        .click();
      if (engine.engine === 'postgresql') {
        if (engine.isPhysical)
          await wizard
            .getByRole('button', {
              name: new RegExp(`^${dictionary.databases.create.backupType.physical.title}`),
            })
            .click();
        await assertEnlargedTextReflow(page);

        await wizard
          .getByRole('button', { name: dictionary.common.actions.continue, exact: true })
          .click();
      }
      const pasteInvoker = wizard.getByRole('button', {
        name: dictionary.databases.edit.parseFromClipboard,
        exact: true,
      });
      await expect(pasteInvoker).toBeVisible();
      await assertEnlargedTextReflow(page);

      await pasteInvoker.focus();
      await page.keyboard.press('Enter');
      const paste = page.getByRole('dialog', {
        name: dictionary.app.clipboardPaste.title,
        exact: true,
      });
      const submitPaste = paste.getByRole('button', {
        name: dictionary.app.clipboardPaste.submit,
        exact: true,
      });
      await expect(submitPaste).toBeDisabled();
      await paste
        .getByRole('textbox', { name: dictionary.app.clipboardPaste.title, exact: true })
        .fill(engine.uri);
      await assertEnlargedTextReflow(page);
      if (engine.connectionType === 'postgresql-logical')
        await assertScreenshot(
          page,
          `connection-paste-${context.theme}-${context.locale}-${context.width}.png`,
        );
      await submitPaste.click();
      await expect(paste).toBeHidden();
      await wizard
        .getByRole('button', { name: dictionary.databases.edit.advancedSettings, exact: true })
        .click();
      await assertEnlargedTextReflow(page);

      const tunnel = wizard.getByRole('checkbox', {
        name: dictionary.databases.sshTunnel.title,
        exact: true,
      });
      await tunnel.check();
      const sshHost = wizard.getByRole('textbox', {
        name: dictionary.databases.sshTunnel.host,
        exact: true,
      });
      await sshHost.fill('bastion.example.com');
      await wizard
        .getByRole('textbox', { name: dictionary.databases.sshTunnel.username, exact: true })
        .fill('backup');
      await wizard
        .getByLabel(dictionary.databases.sshTunnel.password, { exact: true })
        .fill('synthetic-password');
      await assertEnlargedTextReflow(page);

      const authentication = wizard.getByRole('combobox', {
        name: dictionary.databases.sshTunnel.authType,
        exact: true,
      });
      await authentication.focus();
      await authentication.press('ArrowDown');
      await assertEnlargedTextReflow(page);

      const privateKeyOption = page.getByRole('option', {
        name: dictionary.databases.sshTunnel.authTypes.privateKey,
        exact: true,
        includeHidden: true,
      });
      const privateKeyOptionId = await privateKeyOption.getAttribute('id');
      if (!privateKeyOptionId)
        throw new Error('The SSH authentication option has no accessible ID');
      if ((await authentication.getAttribute('aria-activedescendant')) !== privateKeyOptionId) {
        await authentication.press('ArrowDown');
      }
      await expect(authentication).toHaveAttribute('aria-activedescendant', privateKeyOptionId);
      await authentication.press('Enter');
      const privateKey = wizard.getByRole('textbox', {
        name: dictionary.databases.sshTunnel.privateKey,
        exact: true,
      });
      await privateKey.focus();
      await expect(privateKey).toBeFocused();
      await privateKey.pressSequentially('synthetic-private-key');
      await expect(privateKey).toHaveValue('synthetic-private-key');
      await wizard
        .getByLabel(dictionary.databases.sshTunnel.keyPassphrase, { exact: true })
        .fill('synthetic-passphrase');
      await expect(privateKey).toHaveValue('synthetic-private-key');
      await assertEnlargedTextReflow(page);

      if (context.width === 320) {
        await enlargeText(wizard);
        await assertAccountReflow(wizard);
        if (engine.connectionType === 'postgresql-logical') {
          await expect(
            page.getByText(dictionary.databases.edit.connectionStringParsed, { exact: true }),
          ).toBeHidden();
          await assertScreenshot(page, 'connection-ssh-fr-narrow-200.png');
        }
      }
      await wizard
        .getByRole('button', { name: dictionary.databases.actions.testConnection, exact: true })
        .click();
      await wizard
        .getByRole('button', { name: dictionary.common.actions.continue, exact: true })
        .click();
      const limitedTitle = engine.isPhysical
        ? dictionary.databases.readOnlyUser.replicationOnly.title
        : dictionary.databases.readOnlyUser.readOnly.title;
      await expect(wizard.getByText(limitedTitle, { exact: true })).toBeVisible();
      if (context.width === 320) {
        for (const name of [
          dictionary.common.actions.back,
          dictionary.databases.readOnlyUser.skip,
        ]) {
          const action = wizard.getByRole('button', { name, exact: true });
          const lineCount = await action.evaluate((button) => {
            const walker = document.createTreeWalker(button, NodeFilter.SHOW_TEXT);
            const lines = new Set<number>();
            let text: Node | null;
            while ((text = walker.nextNode())) {
              if (!text.textContent?.trim()) continue;
              const range = document.createRange();
              range.selectNodeContents(text);
              for (const rect of range.getClientRects()) lines.add(rect.top);
            }
            return lines.size;
          });
          expect(lineCount).toBe(1);
        }
        await assertAccountReflow(wizard);
      }
      await assertEnlargedTextReflow(page);

      await wizard
        .getByRole('button', { name: dictionary.databases.readOnlyUser.skip, exact: true })
        .click();
      const skipTitle = engine.isPhysical
        ? dictionary.databases.readOnlyUser.replicationOnly.skipTitle
        : dictionary.databases.readOnlyUser.readOnly.skipTitle;
      const skip = page.getByRole('dialog', { name: skipTitle, exact: true });
      await expect(skip).toBeVisible();
      await assertEnlargedTextReflow(page);

      await skip
        .getByRole('button', {
          name: dictionary.databases.readOnlyUser.continueSecurely,
          exact: true,
        })
        .focus();
      await page.keyboard.press('Escape');
      await expect(skip).toBeHidden();
      await expect(wizard).toBeVisible();
    });
  }
}
