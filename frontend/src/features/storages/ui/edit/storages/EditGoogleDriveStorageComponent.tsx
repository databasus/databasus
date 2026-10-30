import { Button, Input } from 'antd';
import { useTranslation } from 'react-i18next';

import type { GoogleDriveStorageSettings, StorageOauthDto } from '../../../../../entities/storages';
import { getWebsitePageUrl, useLocale } from '../../../../../shared/i18n';
import { toOptionalValue } from '../../../../../shared/lib';

interface Props {
  settings: GoogleDriveStorageSettings;
  onChange: (settings: GoogleDriveStorageSettings) => void;
  // The rest of the form, which the OAuth redirect carries back to the page that saves the storage.
  oauthForm: Omit<StorageOauthDto, 'settings' | 'authCode'>;
}

export function EditGoogleDriveStorageComponent({ settings, onChange, oauthForm }: Props) {
  const { t } = useTranslation();
  const { locale } = useLocale();

  const goToAuthUrl = () => {
    if (!settings.clientId || !settings.clientSecret) {
      return;
    }

    const redirectUri = `${window.location.origin}/storages/google-oauth`;
    const clientId = settings.clientId;
    const scope = 'https://www.googleapis.com/auth/drive.file';

    const oauthDto: StorageOauthDto = { ...oauthForm, settings, authCode: '' };

    // eslint-disable-next-line i18next/no-literal-string -- OAuth request URL, not copy
    const authUrl = `https://accounts.google.com/o/oauth2/v2/auth?client_id=${
      clientId
    }&redirect_uri=${redirectUri}&response_type=code&scope=${encodeURIComponent(scope)}&access_type=offline&prompt=consent&state=${encodeURIComponent(JSON.stringify(oauthDto))}`;

    window.open(authUrl);
  };

  return (
    <>
      <div className="mb-2 flex items-center">
        <div className="hidden min-w-[110px] sm:block" />

        <div className="text-xs text-blue-600">
          <a
            href={getWebsitePageUrl('storagesGoogleDrive', locale)}
            target="_blank"
            rel="noreferrer"
          >
            {t('storages.edit.googleDrive.guideLink')}
          </a>
        </div>
      </div>

      <div className="mb-1 flex w-full flex-col items-start sm:flex-row sm:items-center">
        <div className="mb-1 min-w-[110px] sm:mb-0 sm:pr-2">{t('storages.fields.clientId')}</div>
        <Input
          value={settings.clientId || ''}
          onChange={(e: React.ChangeEvent<HTMLInputElement>) => {
            onChange({ ...settings, clientId: e.target.value.trim() });
          }}
          size="small"
          className="w-full max-w-[250px]"
          // eslint-disable-next-line i18next/no-literal-string -- example client ID
          placeholder="my-client-id"
          disabled={!!settings.tokenJson}
        />
      </div>

      <div className="mb-1 flex w-full flex-col items-start sm:flex-row sm:items-center">
        <div className="mb-1 min-w-[110px] sm:mb-0 sm:pr-2">
          {t('storages.fields.clientSecret')}
        </div>
        <Input
          value={settings.clientSecret || ''}
          onChange={(e: React.ChangeEvent<HTMLInputElement>) => {
            onChange({ ...settings, clientSecret: toOptionalValue(e.target.value.trim()) });
          }}
          size="small"
          className="w-full max-w-[250px]"
          // eslint-disable-next-line i18next/no-literal-string -- example client secret
          placeholder="my-client-secret"
          disabled={!!settings.tokenJson}
        />
      </div>

      {settings.tokenJson && (
        <>
          <div className="mb-1 flex w-full flex-col items-start sm:flex-row sm:items-center">
            <div className="mb-1 min-w-[110px] sm:mb-0 sm:pr-2">
              {t('storages.fields.userToken')}
            </div>
            <Input
              value={settings.tokenJson || ''}
              disabled
              size="small"
              className="w-full max-w-[250px]"
              // eslint-disable-next-line i18next/no-literal-string -- example token
              placeholder="my-user-token"
            />
          </div>
        </>
      )}

      {!settings.tokenJson && (
        <Button
          type="primary"
          disabled={!settings.clientId || !settings.clientSecret}
          onClick={goToAuthUrl}
        >
          {t('storages.edit.googleDrive.authorize')}
        </Button>
      )}
    </>
  );
}
