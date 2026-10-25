import { Modal, Spin } from 'antd';
import { useEffect, useState } from 'react';
import { useTranslation } from 'react-i18next';

import { type Storage, StorageType } from '../entity/storages';
import type { StorageOauthDto } from '../entity/storages/models/StorageOauthDto';
import type { User } from '../entity/users';
import { userApi } from '../entity/users';
import { EditStorageComponent } from '../features/storages/ui/edit/EditStorageComponent';

export function OauthStorageComponent() {
  const { t } = useTranslation();
  const [storage, setStorage] = useState<Storage | undefined>();
  const [user, setUser] = useState<User | undefined>();

  const exchangeGoogleOauthCode = async (oauthDto: StorageOauthDto) => {
    if (!oauthDto.storage.googleDriveStorage) {
      alert(t('app.oauthStorage.configNotFound'));
      return;
    }

    const { clientId, clientSecret } = oauthDto.storage.googleDriveStorage;
    const { authCode } = oauthDto;

    const redirectUri = `${window.location.origin}/storages/google-oauth`;

    try {
      // Exchange authorization code for access token
      const response = await fetch('https://oauth2.googleapis.com/token', {
        method: 'POST',
        headers: {
          'Content-Type': 'application/x-www-form-urlencoded',
        },
        body: new URLSearchParams({
          code: authCode,
          client_id: clientId,
          client_secret: clientSecret,
          redirect_uri: redirectUri,
          grant_type: 'authorization_code',
        }),
      });

      if (!response.ok) {
        const errorData = await response.json();
        throw new Error(
          errorData.error_description || `OAuth exchange failed: ${response.statusText}`,
        );
      }

      const tokenData = await response.json();

      oauthDto.storage.googleDriveStorage.tokenJson = JSON.stringify(tokenData);
      setStorage(oauthDto.storage);
    } catch (error) {
      // Google's own error description is English and technical: it goes to the console only
      console.error('Failed to exchange OAuth code:', error);
      alert(t('app.oauthStorage.exchangeFailed'));
      // Return to home if exchange fails
      setTimeout(() => {
        window.location.href = '/';
      }, 3000);
    }
  };

  /**
   * Helper to validate the DTO and start the exchange process
   */
  const processOauthDto = (oauthDto: StorageOauthDto) => {
    if (oauthDto.storage.type === StorageType.GOOGLE_DRIVE) {
      if (!oauthDto.storage.googleDriveStorage) {
        alert(t('app.oauthStorage.configNotFound'));
        return;
      }

      exchangeGoogleOauthCode(oauthDto);
    } else {
      alert(t('app.oauthStorage.unsupportedType'));
    }
  };

  useEffect(() => {
    userApi
      .getCurrentUser()
      .then(setUser)
      .catch(() => {
        window.location.href = '/';
      });

    const urlParams = new URLSearchParams(window.location.search);
    const code = urlParams.get('code');
    const state = urlParams.get('state');

    if (code && state) {
      try {
        const decodedState = decodeURIComponent(state);
        const oauthDto: StorageOauthDto = JSON.parse(decodedState);

        oauthDto.authCode = code;

        processOauthDto(oauthDto);
        return;
      } catch (e) {
        console.error('Error parsing OAuth state:', e);
        alert(t('app.oauthStorage.invalidState'));
        return;
      }
    }

    alert(t('app.oauthStorage.paramNotFound'));
  }, []);

  if (!storage || !user) {
    return (
      <div className="mt-20 flex justify-center">
        <Spin />
      </div>
    );
  }

  return (
    <div>
      <Modal
        title={t('app.oauthStorage.addStorageTitle')}
        footer={<div />}
        open
        onCancel={() => {
          window.location.href = '/';
        }}
      >
        <div className="my-3 max-w-[250px] text-gray-500 dark:text-gray-400">
          {t('app.oauthStorage.storageDescription')}
        </div>

        <EditStorageComponent
          workspaceId={storage.workspaceId}
          isShowClose={false}
          onClose={() => {}}
          isShowName={false}
          editingStorage={storage}
          onChanged={() => {
            window.location.href = '/';
          }}
        />
      </Modal>
    </div>
  );
}
