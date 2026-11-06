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
