import { expect } from '@playwright/test';

import { DICTIONARIES } from '../../src/shared/i18n/dictionaries';
import { workspace } from '../applicationFixtures';
import { assertEnlargedTextReflow } from '../assertEnlargedTextReflow';
import { test } from '../browserTest';
import { enlargeText } from '../enlargeText';
import { apiResponse } from '../HttpFixture';
import {
  buildInstanceBackupStatus,
  instanceBackupFixtures,
  offsiteStorage,
  preparedRecovery,
} from '../instanceBackupFixtures';
import { openAccount } from '../openAccount';
import { openAccountTab } from '../openAccountTab';

const stoppedIncident = {
  id: 'incident-id',
  kind: 'STOPPED' as const,
  startedAt: '2026-10-18T11:00:00Z',
  mailStatus: 'SENT' as const,
};

const contexts = [
  { locale: 'en' as const, theme: 'light', width: 1280 },
  { locale: 'ru' as const, theme: 'dark', width: 320 },
];

test.describe('Databasus backups visibility', () => {
  test.use({ httpFixtures: [instanceBackupFixtures({ isRootAdmin: false }), { scope: 'test' }] });

  test('hides the section from an administrator who is not root admin', async ({
    page,
    boundary,
  }) => {
    await openAccount({ page });
    await openAccountTab({ page, locale: 'en', tab: 'databasusSettings' });

    await expect(
      page.getByRole('heading', { name: DICTIONARIES.en.settings.mailServer.title }),
    ).toBeVisible();
    await expect(
      page.getByRole('heading', { name: DICTIONARIES.en.instanceBackups.title }),
    ).toHaveCount(0);
    expect(boundary.requests.some((request) => request.path.startsWith('/instance-backup'))).toBe(
      false,
    );
  });
});

test.describe('Databasus backups section', () => {
  test.use({
    httpFixtures: [
      instanceBackupFixtures({
        status: buildInstanceBackupStatus({ isEmailConfigured: false }),
        extra: [
          apiResponse({
            method: 'post',
            path: '/storage-tests',
            status: 200,
            body: { isConnected: true },
          }),
          apiResponse({ method: 'post', path: '/storages', status: 201, body: offsiteStorage }),
        ],
      }),
      { scope: 'test' },
    ],
  });

  for (const context of contexts) {
    test(`creates a destination from the settings dialog and scrolls to the mail server in ${context.locale} ${context.theme} ${context.width}`, async ({
      page,
    }) => {
      const dictionary = DICTIONARIES[context.locale];
      const text = dictionary.instanceBackups;
      await page.setViewportSize({ width: context.width, height: 900 });
      await openAccount({ page, ...context });
      await openAccountTab({ page, locale: context.locale, tab: 'databasusSettings' });

      await page.getByRole('button', { name: text.setup.start, exact: true }).click();
      await page.getByRole('combobox', { name: text.destinations.picker, exact: true }).click();
      await page.getByRole('button', { name: text.destinations.create, exact: true }).click();
      const dialog = page.getByRole('dialog', { name: text.createDestination.title, exact: true });
      await dialog
        .getByRole('button', { name: text.createDestination.continue, exact: true })
        .click();
      await dialog
        .getByRole('textbox', { name: dictionary.common.fields.name, exact: true })
        .fill(offsiteStorage.name);
      await dialog
        .getByRole('button', { name: dictionary.storages.actions.testConnection, exact: true })
        .click();
      await dialog
        .getByRole('button', { name: dictionary.common.actions.save, exact: true })
        .click();

      await expect(dialog).toHaveCount(0);
      const settingsDialog = page.getByRole('dialog', { name: text.settings.title, exact: true });
      await expect(
        settingsDialog.getByText(offsiteStorage.name, { exact: true }).first(),
      ).toBeVisible();
      await expect(page).toHaveURL(/\/databasus-settings$/);
      await expect(settingsDialog.getByText(workspace.name).first()).toBeVisible();
      await settingsDialog
        .getByRole('button', { name: dictionary.common.actions.cancel, exact: true })
        .click();
      await expect(settingsDialog).toHaveCount(0);

      const configure = page.getByRole('button', { name: text.smtp.configure, exact: true });
      await configure.focus();
      await page.keyboard.press('Enter');

      const mailHeading = page.getByRole('heading', {
        name: dictionary.settings.mailServer.title,
        exact: true,
      });
      await expect(mailHeading).toBeFocused();
      await expect(mailHeading).toBeInViewport();

      await enlargeText(page.getByRole('main'));
      await assertEnlargedTextReflow(page);
    });
  }
});

test.describe('Master key onboarding', () => {
  const installedKey = 'k3Jd-installed-master-key-9fQa';
  const unsavedStatus = buildInstanceBackupStatus({ isMasterKeyAcknowledged: false });

  test.use({
    httpFixtures: [
      instanceBackupFixtures({
        status: unsavedStatus,
        extra: [
          apiResponse({
            method: 'post',
            path: '/master-key-exports',
            status: 201,
            body: {
              token: 'key-token',
              fileName: 'databasus-master-key.txt',
              expiresAt: '2999-01-01T00:00:00Z',
            },
          }),
          apiResponse({
            method: 'put',
            path: '/master-key-acknowledgments',
            status: 200,
            body: { isSaved: true },
          }),
        ],
      }),
      { scope: 'test' },
    ],
  });

  for (const context of contexts) {
    test(`replaces the application until the key is copied and confirmed in ${context.locale} ${context.theme} ${context.width}`, async ({
      page,
      boundary,
    }) => {
      const dictionary = DICTIONARIES[context.locale];
      // The key file is plain text, which the JSON boundary fixtures cannot serve.
      await page.route('**/api/v1/master-key-downloads?*', (route) =>
        route.fulfill({ status: 200, contentType: 'application/octet-stream', body: installedKey }),
      );
      // Over plain HTTP the Clipboard API is missing, and the copy must still work there.
      await page.addInitScript(() => {
        Object.defineProperty(navigator, 'clipboard', { configurable: true, value: undefined });
        const copiedTexts: string[] = [];
        Object.defineProperty(window, 'copiedTexts', { value: copiedTexts });
        document.addEventListener(
          'copy',
          () => {
            const field = document.activeElement;
            copiedTexts.push(
              field instanceof HTMLTextAreaElement
                ? field.value.slice(field.selectionStart, field.selectionEnd)
                : (document.getSelection()?.toString() ?? ''),
            );
          },
          true,
        );
      });
      // The confirmation unlocks 15 seconds after the screen appears; the test runs that time.
      await page.clock.install({ time: new Date('2026-10-18T12:00:00Z') });
      await page.setViewportSize({ width: context.width, height: 900 });
      await openAccount({ page, ...context });

      const onboarding = page.getByRole('heading', {
        name: dictionary.masterKey.onboarding.title,
        exact: true,
        level: 1,
      });
      const closeButton = page.getByRole('button', { name: dictionary.masterKey.onboarding.close });
      await expect(page.getByText('k3Jd••••••••••••9fQa', { exact: true })).toBeVisible();
      await expect(onboarding).toBeVisible();
      await expect(closeButton).toBeDisabled();
      await expect(page.getByRole('navigation')).toHaveCount(0);
      await assertEnlargedTextReflow(page);

      await page.getByRole('button', { name: dictionary.masterKey.copy, exact: true }).click();

      await expect(
        page.getByRole('button', { name: dictionary.masterKey.copied, exact: true }),
      ).toBeVisible();
      await expect(closeButton).toBeDisabled();
      await page.clock.runFor(15_000);
      await expect(closeButton).toHaveText(dictionary.masterKey.onboarding.close);
      await expect(closeButton).toBeEnabled();
      expect(
        await page.evaluate(() => (window as unknown as { copiedTexts: string[] }).copiedTexts),
      ).toEqual([installedKey]);

      await page.route('**/api/v1/instance-backup-statuses', (route) =>
        route.fulfill({
          status: 200,
          contentType: 'application/json',
          body: JSON.stringify({ ...unsavedStatus, isMasterKeyAcknowledged: true }),
        }),
      );
      await closeButton.click();

      await expect(onboarding).toBeHidden();
      expect(
        boundary.requests.filter((request) => request.path === '/master-key-acknowledgments'),
      ).toEqual([{ method: 'PUT', path: '/master-key-acknowledgments', body: { isSaved: true } }]);
    });
  }
});

test.describe('Floating incident notice', () => {
  test.use({
    httpFixtures: [
      instanceBackupFixtures({
        status: buildInstanceBackupStatus({ health: 'STOPPED', incident: stoppedIncident }),
      }),
      { scope: 'test' },
    ],
  });

  for (const context of contexts) {
    test(`keeps the notice clear of controls in ${context.locale} ${context.theme} ${context.width}`, async ({
      page,
    }) => {
      const dictionary = DICTIONARIES[context.locale];
      await page.setViewportSize({ width: context.width, height: 900 });
      await openAccount({ page, ...context });

      const notice = page.getByRole('status', {
        name: dictionary.instanceBackupNotices.label,
        exact: true,
      });
      await expect(notice).toBeVisible();
      await expect(page.locator(':focus')).toHaveCount(0);

      const position = await notice.evaluate((element) => getComputedStyle(element).position);
      expect(position).toBe(context.width === 320 ? 'static' : 'fixed');

      await notice
        .getByRole('button', { name: dictionary.instanceBackupNotices.open, exact: true })
        .click();

      await expect(page).toHaveURL(/\/databasus-settings$/);
      await expect(
        page.getByRole('heading', { name: dictionary.instanceBackups.title, exact: true }),
      ).toBeFocused();
      await expect(notice).toHaveCount(0);
    });
  }
});

test.describe('Recovery import form', () => {
  test.use({
    httpFixtures: [
      instanceBackupFixtures({
        extra: [
          apiResponse({
            method: 'post',
            path: '/instance-recoveries',
            status: 202,
            body: preparedRecovery,
          }),
          apiResponse({
            method: 'post',
            path: '/instance-recoveries/{instanceRecoveryId}/apply',
            status: 409,
            body: {
              type: 'urn:databasus:problem:recovery_not_ready',
              title: 'Refused',
              detail: 'Synthetic refusal',
              status: 409,
              code: 'recovery_not_ready',
            },
          }),
        ],
      }),
      { scope: 'test' },
    ],
  });

  for (const context of contexts) {
    test(`picks an archive, takes a typed key and confirms replacement in ${context.locale} ${context.theme} ${context.width}`, async ({
      page,
      boundary,
    }) => {
      const dictionary = DICTIONARIES[context.locale];
      const text = dictionary.instanceRecovery;
      await page.setViewportSize({ width: context.width, height: 900 });
      await openAccount({ page, ...context });
      await openAccountTab({ page, locale: context.locale, tab: 'databasusSettings' });

      await page.getByRole('button', { name: text.open, exact: true }).click();
      const importDialog = page.getByRole('dialog', { name: text.form.title, exact: true });
      await importDialog.getByRole('button', { name: text.form.upload, exact: true }).click();
      await expect(importDialog.getByText(text.form.archiveMissing, { exact: true })).toBeVisible();
      await expect(
        importDialog.getByText(text.form.masterKeyMissing, { exact: true }),
      ).toBeVisible();

      await page.locator('input[type="file"]').setInputFiles({
        name: 'databasus.dbsbackup',
        mimeType: 'application/octet-stream',
        buffer: Buffer.from('encrypted archive'),
      });
      const keyField = page.getByLabel(text.form.masterKey, { exact: true });
      await keyField.focus();
      await page.keyboard.type('typed-source-key');

      await importDialog.getByRole('button', { name: text.form.upload, exact: true }).click();

      const dialog = page.getByRole('dialog', { name: text.apply.title, exact: true });
      await expect(dialog.getByText(text.form.ready, { exact: true })).toBeVisible();
      const upload = boundary.requests.find((request) => request.path === '/instance-recoveries');
      expect(String(upload?.body)).toContain('typed-source-key');

      await dialog
        .getByRole('checkbox', { name: text.apply.confirmReplacement, exact: true })
        .check();
      await dialog.getByRole('button', { name: text.apply.submit, exact: true }).click();

      await expect(
        dialog.getByText(dictionary.errors.recoveryNotReady, { exact: true }),
      ).toBeVisible();
      await assertEnlargedTextReflow(page);
    });
  }
});

test.describe('Recovery key paste and restored sign-in', () => {
  test.use({
    httpFixtures: [
      instanceBackupFixtures({
        extra: [
          apiResponse({
            method: 'post',
            path: '/instance-recoveries',
            status: 202,
            body: preparedRecovery,
          }),
          apiResponse({
            method: 'post',
            path: '/instance-recoveries/{instanceRecoveryId}/apply',
            status: 202,
            body: { ...preparedRecovery, status: 'APPLYING' },
          }),
          apiResponse({
            method: 'get',
            path: '/instance-recoveries/{instanceRecoveryId}',
            status: 401,
            body: {
              type: 'urn:databasus:problem:unauthenticated',
              title: 'Unauthenticated',
              detail: 'Synthetic session end',
              status: 401,
              code: 'unauthenticated',
            },
          }),
        ],
      }),
      { scope: 'test' },
    ],
  });

  test('keeps the pasted key exact and asks for a restored account after the session ends', async ({
    page,
    boundary,
  }) => {
    const dictionary = DICTIONARIES.en;
    const text = dictionary.instanceRecovery;
    await openAccount({ page });
    await openAccountTab({ page, locale: 'en', tab: 'databasusSettings' });

    await page.getByRole('button', { name: text.open, exact: true }).click();
    await page.locator('input[type="file"]').setInputFiles({
      name: 'databasus.dbsbackup',
      mimeType: 'application/octet-stream',
      buffer: Buffer.from('encrypted archive'),
    });
    // The page is served over HTTP, where the Clipboard API is unavailable, so the paste reaches the
    // rendered field as the event a browser dispatches for Ctrl+V.
    await page.getByLabel(text.form.masterKey, { exact: true }).evaluate((field) => {
      const clipboardData = new DataTransfer();
      clipboardData.setData('text/plain', 'pasted-key\r\nsecond line');
      field.dispatchEvent(
        new ClipboardEvent('paste', { clipboardData, bubbles: true, cancelable: true }),
      );
    });
    await page.getByRole('button', { name: text.form.upload, exact: true }).click();
    const dialog = page.getByRole('dialog', { name: text.apply.title, exact: true });
    await expect(dialog.getByText(text.form.ready, { exact: true })).toBeVisible();

    const upload = boundary.requests.find((request) => request.path === '/instance-recoveries');
    expect(String(upload?.body)).toContain('pasted-key\r\nsecond line');

    await dialog
      .getByRole('checkbox', { name: text.apply.confirmReplacement, exact: true })
      .check();
    await dialog.getByRole('button', { name: text.apply.submit, exact: true }).click();

    await expect(page.getByText(text.form.applying, { exact: true })).toBeVisible();
    await expect(page.getByText(text.restoredSignIn, { exact: true })).toBeVisible();
  });
});
