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
    test(`creates a destination inline and scrolls to the mail server in ${context.locale} ${context.theme} ${context.width}`, async ({
      page,
    }) => {
      const dictionary = DICTIONARIES[context.locale];
      const text = dictionary.instanceBackups;
      await page.setViewportSize({ width: context.width, height: 900 });
      await openAccount({ page, ...context });
      await openAccountTab({ page, locale: context.locale, tab: 'databasusSettings' });

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
      await expect(page.getByText(offsiteStorage.name, { exact: true }).first()).toBeVisible();
      await expect(page).toHaveURL(/\/databasus-settings$/);
      await expect(page.getByText(workspace.name).first()).toBeVisible();

      const configure = page.getByRole('button', { name: text.smtp.configure, exact: true });
      await configure.focus();
      await page.keyboard.press('Enter');

      const mailHeading = page.getByRole('heading', {
        name: dictionary.settings.mailServer.title,
        exact: true,
      });
      await expect(mailHeading).toBeFocused();
      await expect(mailHeading).toBeInViewport();
      await expect(page.getByRole('button', { name: text.settings.save, exact: true })).toHaveCount(
        1,
      );

      await enlargeText(page.getByRole('main'));
      await assertEnlargedTextReflow(page);
    });
  }
});

test.describe('Master key onboarding over HTTP', () => {
  test.use({
    httpFixtures: [
      instanceBackupFixtures({
        status: buildInstanceBackupStatus({ isMasterKeyAcknowledged: false }),
      }),
      { scope: 'test' },
    ],
  });

  for (const context of contexts) {
    test(`defers onboarding without requesting key material in ${context.locale} ${context.theme} ${context.width}`, async ({
      page,
      boundary,
    }) => {
      const dictionary = DICTIONARIES[context.locale];
      await page.setViewportSize({ width: context.width, height: 900 });
      await openAccount({ page, ...context });

      const dialog = page.getByRole('dialog', {
        name: dictionary.masterKey.onboarding.title,
        exact: true,
      });
      await expect(
        dialog.getByText(dictionary.masterKey.insecure.title, { exact: true }),
      ).toBeVisible();
      await assertEnlargedTextReflow(page);
      await page.keyboard.press('Escape');

      await expect(dialog).toHaveCount(0);
      await expect(
        page.getByText(dictionary.instanceBackupNotices.masterKey, { exact: true }),
      ).toBeVisible();
      expect(boundary.requests.some((request) => request.path.startsWith('/master-key'))).toBe(
        false,
      );
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

test.describe('Recovery pause', () => {
  test.use({
    httpFixtures: [
      instanceBackupFixtures({
        status: buildInstanceBackupStatus({
          isRecoveryPaused: true,
          recovery: { ...preparedRecovery, status: 'PAUSED' },
        }),
        extra: [
          apiResponse({
            method: 'post',
            path: '/instance-recoveries/{instanceRecoveryId}/activate',
            status: 200,
            body: { ...preparedRecovery, status: 'ACTIVE' },
          }),
        ],
      }),
      { scope: 'test' },
    ],
  });

  for (const context of contexts) {
    test(`activates only after explicit confirmation from the keyboard in ${context.locale} ${context.theme} ${context.width}`, async ({
      page,
      boundary,
    }) => {
      const text = DICTIONARIES[context.locale].instanceRecovery.pause;
      await page.setViewportSize({ width: context.width, height: 900 });
      await openAccount({ page, ...context });

      await page.getByRole('button', { name: text.activate, exact: true }).click();
      const dialog = page.getByRole('dialog', { name: text.confirmTitle, exact: true });
      const submit = dialog.getByRole('button', { name: text.activate, exact: true });
      await expect(submit).toBeDisabled();

      await dialog.getByRole('checkbox', { name: text.oldInstanceStopped, exact: true }).focus();
      await page.keyboard.press('Space');
      await submit.click();

      await expect(dialog).toHaveCount(0);
      expect(boundary.requests.filter((request) => request.path.endsWith('/activate'))).toEqual([
        {
          method: 'POST',
          path: `/instance-recoveries/${preparedRecovery.id}/activate`,
          body: { isOldInstanceStopped: true },
        },
      ]);
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
            status: 400,
            body: {
              type: 'urn:databasus:problem:identity_proof_invalid',
              title: 'Refused',
              detail: 'Synthetic refusal',
              status: 400,
              code: 'identity_proof_invalid',
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

      await page
        .getByRole('button', { name: text.form.upload, exact: true })
        .scrollIntoViewIfNeeded();
      await page.getByRole('checkbox', { name: text.form.insecureAccept, exact: true }).check();
      await page.getByRole('button', { name: text.form.upload, exact: true }).click();
      await expect(page.getByText(text.form.archiveMissing, { exact: true })).toBeVisible();
      await expect(page.getByText(text.form.masterKeyMissing, { exact: true })).toBeVisible();

      await page.getByRole('checkbox', { name: text.form.insecureAccept, exact: true }).uncheck();
      await page.locator('input[type="file"]').setInputFiles({
        name: 'databasus.dbsbackup',
        mimeType: 'application/octet-stream',
        buffer: Buffer.from('encrypted archive'),
      });
      const keyField = page.getByRole('textbox', { name: new RegExp(text.form.masterKey) });
      await keyField.focus();
      await page.keyboard.type('typed-source-key');
      await expect(
        page.getByRole('button', { name: text.form.upload, exact: true }),
      ).toBeDisabled();
      expect(boundary.requests.some((request) => request.path === '/instance-recoveries')).toBe(
        false,
      );

      await page.getByRole('checkbox', { name: text.form.insecureAccept, exact: true }).check();
      await page.getByRole('button', { name: text.form.upload, exact: true }).click();

      await expect(page.getByText(text.form.ready, { exact: true })).toBeVisible();
      const upload = boundary.requests.find((request) => request.path === '/instance-recoveries');
      expect(String(upload?.body)).toContain('typed-source-key');

      await page.getByRole('button', { name: text.form.replace, exact: true }).click();
      const dialog = page.getByRole('dialog', { name: text.apply.title, exact: true });
      await dialog
        .getByRole('checkbox', { name: text.apply.confirmReplacement, exact: true })
        .check();
      await dialog
        .getByLabel(dictionary.masterKey.proof.password, { exact: true })
        .fill('wrong-password');
      await dialog.getByRole('button', { name: text.apply.submit, exact: true }).click();

      await expect(
        dialog.getByText(dictionary.errors.identityProofInvalid, { exact: true }),
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

    await page.locator('input[type="file"]').setInputFiles({
      name: 'databasus.dbsbackup',
      mimeType: 'application/octet-stream',
      buffer: Buffer.from('encrypted archive'),
    });
    // The page is served over HTTP, where the Clipboard API is unavailable, so the paste reaches the
    // rendered field as the event a browser dispatches for Ctrl+V.
    await page.getByRole('textbox', { name: new RegExp(text.form.masterKey) }).evaluate((field) => {
      const clipboardData = new DataTransfer();
      clipboardData.setData('text/plain', 'pasted-key\r\nsecond line');
      field.dispatchEvent(
        new ClipboardEvent('paste', { clipboardData, bubbles: true, cancelable: true }),
      );
    });
    await page.getByRole('checkbox', { name: text.form.insecureAccept, exact: true }).check();
    await page.getByRole('button', { name: text.form.upload, exact: true }).click();
    await expect(page.getByText(text.form.ready, { exact: true })).toBeVisible();

    const upload = boundary.requests.find((request) => request.path === '/instance-recoveries');
    expect(String(upload?.body)).toContain('pasted-key\r\nsecond line');

    await page.getByRole('button', { name: text.form.replace, exact: true }).click();
    const dialog = page.getByRole('dialog', { name: text.apply.title, exact: true });
    await dialog
      .getByRole('checkbox', { name: text.apply.confirmReplacement, exact: true })
      .check();
    await dialog.getByLabel(dictionary.masterKey.proof.password, { exact: true }).fill('root-pass');
    await dialog.getByRole('button', { name: text.apply.submit, exact: true }).click();

    await expect(page.getByText(text.form.applying, { exact: true })).toBeVisible();
    await expect(page.getByText(text.restoredSignIn, { exact: true })).toBeVisible();
  });
});
