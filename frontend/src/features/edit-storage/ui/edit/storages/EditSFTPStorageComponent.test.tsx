// @vitest-environment happy-dom
import { isInaccessible, screen } from '@testing-library/react';
import { describe, expect, it } from 'vitest';

import { EditSFTPStorageComponent } from '@/features/edit-storage/ui/edit/storages/EditSFTPStorageComponent';
import { DICTIONARIES, SUPPORTED_LOCALES, type Locale } from '@/shared/i18n';
import { renderScreen } from '@/shared/lib/test/renderScreen';

const renderSftpEditor = ({
  locale = 'en',
  isSkipHostKeyVerify = false,
}: {
  locale?: Locale;
  isSkipHostKeyVerify?: boolean;
} = {}) =>
  renderScreen(
    <EditSFTPStorageComponent
      settings={{
        type: 'SFTP',
        host: 'sftp.example.com',
        port: 22,
        username: 'backup',
        authMethods: ['PASSWORD'],
        password: 'password',
        isSkipHostKeyVerify,
      }}
      savedAuthMethods={[]}
      isAddressChanged={false}
      onChange={() => {}}
    />,
    { locale },
  );

const assertSecurityWarning = ({
  locale,
  isSkipHostKeyVerify,
}: {
  locale: Locale;
  isSkipHostKeyVerify: boolean;
}) => {
  const sftpCopy = DICTIONARIES[locale].storages.edit.sftp;
  const warning = screen.getByText(sftpCopy.skipHostKeyTooltip);

  expect(isInaccessible(warning)).toBe(false);

  const skipVerificationCheckbox = screen.getByRole('checkbox', {
    name: sftpCopy.skipHostKeyVerification,
    description: sftpCopy.skipHostKeyTooltip,
  }) as HTMLInputElement;

  expect(skipVerificationCheckbox.checked).toBe(isSkipHostKeyVerify);
};

describe('EditSFTPStorageComponent', () => {
  it.each([
    {
      state: 'saved secrets are omitted',
      hasSavedSecrets: true,
      isAddressChanged: false,
      secret: undefined,
      isInvalid: false,
    },
    {
      state: 'new secrets are present',
      hasSavedSecrets: false,
      isAddressChanged: false,
      secret: 'secret',
      isInvalid: false,
    },
    {
      state: 'new secrets are missing',
      hasSavedSecrets: false,
      isAddressChanged: false,
      secret: undefined,
      isInvalid: true,
    },
    {
      state: 'the address changes and secrets are omitted',
      hasSavedSecrets: true,
      isAddressChanged: true,
      secret: undefined,
      isInvalid: true,
    },
  ])(
    'marks only missing required credentials invalid when $state',
    ({ hasSavedSecrets, isAddressChanged, secret, isInvalid }) => {
      renderScreen(
        <EditSFTPStorageComponent
          settings={{
            type: 'SFTP',
            host: 'sftp.example.com',
            port: 22,
            username: 'backup',
            authMethods: ['PASSWORD', 'PRIVATE_KEY'],
            password: secret,
            privateKey: secret,
            isSkipHostKeyVerify: false,
          }}
          savedAuthMethods={hasSavedSecrets ? ['PASSWORD', 'PRIVATE_KEY'] : []}
          isAddressChanged={isAddressChanged}
          onChange={() => {}}
        />,
      );

      const password = screen.getByLabelText(DICTIONARIES.en.common.fields.password);
      const privateKey = screen.getByRole('textbox', {
        name: DICTIONARIES.en.storages.fields.privateKey,
      });

      for (const credential of [password, privateKey]) {
        expect(credential.getAttribute('aria-invalid') === 'true').toBe(isInvalid);
      }

      if (hasSavedSecrets && !isAddressChanged) {
        expect(screen.getByText(DICTIONARIES.en.storages.edit.sftp.savedPassword)).toBeTruthy();
        expect(screen.getByText(DICTIONARIES.en.storages.edit.sftp.savedPrivateKey)).toBeTruthy();
      }
    },
  );

  it('warns that skipping host key verification reduces security when advanced settings are expanded', async () => {
    const { user } = renderSftpEditor();

    await user.click(
      screen.getByRole('button', { name: DICTIONARIES.en.storages.edit.advancedSettings }),
    );

    assertSecurityWarning({ locale: 'en', isSkipHostKeyVerify: false });
  });

  it.each(SUPPORTED_LOCALES)(
    'shows the warning immediately when verification is skipped in %s',
    (locale) => {
      renderSftpEditor({ locale, isSkipHostKeyVerify: true });

      assertSecurityWarning({ locale, isSkipHostKeyVerify: true });
      expect(
        screen.getByRole('button', {
          name: DICTIONARIES[locale].storages.edit.advancedSettings,
          expanded: true,
        }),
      ).toBeTruthy();
    },
  );
});
