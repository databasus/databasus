import { Alert, Button, Form, Input } from 'antd';
import { useId } from 'react';
import { useTranslation } from 'react-i18next';

import type { GoogleDriveStorageSettings } from '../../../../../entities/storages';
import { getWebsitePageUrl, useLocale } from '../../../../../shared/i18n';
import { toOptionalValue } from '../../../../../shared/lib';
import { useGoogleDriveAuthorization } from '../../../model/useGoogleDriveAuthorization';

interface Props {
  settings: GoogleDriveStorageSettings;
  onChange: (settings: GoogleDriveStorageSettings) => void;
  workspaceId: string;
  storageId?: string;
}

export function EditGoogleDriveStorageComponent(props: Props) {
  const { settings, onChange, storageId } = props;
  const { t } = useTranslation();
  const { locale } = useLocale();
  const controlId = useId();
  const authorization = useGoogleDriveAuthorization(props);
  const hasAuthorization = !!settings.tokenJson || !!storageId;

  return (
    <div className="min-w-0">
      <a href={getWebsitePageUrl('storagesGoogleDrive', locale)} target="_blank" rel="noreferrer">
        {t('storages.edit.googleDrive.guideLink')}
      </a>
      <Form.Item label={t('storages.fields.clientId')} htmlFor={`${controlId}-client`}>
        <Input
          id={`${controlId}-client`}
          value={settings.clientId}
          size="small"
          onChange={(event) => onChange({ ...settings, clientId: event.target.value.trim() })}
        />
      </Form.Item>
      <Form.Item
        label={t('storages.fields.clientSecret')}
        htmlFor={`${controlId}-secret`}
        help={storageId ? t('storages.edit.googleDrive.reconnectHelp') : undefined}
      >
        <Input.Password
          id={`${controlId}-secret`}
          value={settings.clientSecret || ''}
          size="small"
          onChange={(event) =>
            onChange({ ...settings, clientSecret: toOptionalValue(event.target.value.trim()) })
          }
        />
      </Form.Item>
      {hasAuthorization && <div role="status">{t('storages.edit.googleDrive.authorized')}</div>}
      {authorization.isPending && (
        <div role="status">
          {t(
            authorization.status === 'waiting'
              ? 'storages.edit.googleDrive.waiting'
              : 'storages.edit.googleDrive.exchanging',
          )}
        </div>
      )}
      {authorization.errorKey && (
        <Alert type="error" showIcon message={t(authorization.errorKey)} />
      )}
      <div className="mt-2 flex flex-wrap gap-2">
        <Button
          type="primary"
          disabled={!settings.clientId || !settings.clientSecret}
          onClick={authorization.start}
        >
          {t(
            authorization.isPending || authorization.errorKey
              ? 'storages.edit.googleDrive.restart'
              : hasAuthorization
                ? 'storages.edit.googleDrive.reconnect'
                : 'storages.edit.googleDrive.authorize',
          )}
        </Button>
        {authorization.isPending && (
          <Button onClick={authorization.cancel}>{t('storages.edit.googleDrive.cancel')}</Button>
        )}
      </div>
    </div>
  );
}
