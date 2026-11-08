import { Alert, Button, PasswordInput, TextInput } from '@mantine/core';
import { CircleAlert } from 'lucide-react';
import { useId } from 'react';
import { useTranslation } from 'react-i18next';

import type { GoogleDriveStorageSettings } from '@/entities/storages';
import { useGoogleDriveAuthorization } from '@/features/edit-storage/model/useGoogleDriveAuthorization';
import { getWebsitePageUrl, useLocale } from '@/shared/i18n';
import { toOptionalValue } from '@/shared/lib';

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
      <div className="mb-3 min-w-0">
        <label htmlFor={`${controlId}-client`} className="mb-1 block text-sm text-content">
          {t('storages.fields.clientId')}
        </label>
        <TextInput
          id={`${controlId}-client`}
          value={settings.clientId}
          size="xs"
          onChange={(event) => onChange({ ...settings, clientId: event.target.value.trim() })}
        />
      </div>
      <div className="mb-3 min-w-0">
        <label htmlFor={`${controlId}-secret`} className="mb-1 block text-sm text-content">
          {t('storages.fields.clientSecret')}
        </label>
        <PasswordInput
          id={`${controlId}-secret`}
          value={settings.clientSecret || ''}
          size="xs"
          onChange={(event) =>
            onChange({ ...settings, clientSecret: toOptionalValue(event.target.value.trim()) })
          }
          visibilityToggleButtonProps={{
            'aria-label': t('common.actions.togglePasswordVisibility'),
          }}
          description={storageId ? t('storages.edit.googleDrive.reconnectHelp') : undefined}
          inputWrapperOrder={['label', 'input', 'description', 'error']}
        />
      </div>
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
        <Alert
          color="danger"
          icon={
            <CircleAlert aria-hidden size={16} className="inline-block shrink-0 align-middle" />
          }
          title={t(authorization.errorKey)}
        />
      )}
      <div className="mt-2 flex flex-wrap gap-2">
        <Button
          variant="filled"
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
