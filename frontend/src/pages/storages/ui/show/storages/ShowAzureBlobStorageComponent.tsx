import { useTranslation } from 'react-i18next';

import {
  AZURE_BLOB_AUTH_METHOD_LABEL_KEYS,
  type AzureBlobStorageSettings,
} from '../../../../../entities/storages';

interface Props {
  settings: AzureBlobStorageSettings;
}

export function ShowAzureBlobStorageComponent({ settings }: Props) {
  const { t } = useTranslation();

  return (
    <>
      <div className="mb-1 flex min-w-0 flex-wrap items-center gap-y-1 break-words">
        <div className="min-w-27.5 pr-2">{t('storages.fields.authMethod')}</div>
        {t(AZURE_BLOB_AUTH_METHOD_LABEL_KEYS[settings.authMethod ?? 'ACCOUNT_KEY'])}
      </div>

      {settings.authMethod === 'CONNECTION_STRING' && (
        <div className="mb-1 flex min-w-0 flex-wrap items-center gap-y-1 break-words">
          <div className="min-w-27.5 pr-2">{t('storages.fields.connectionString')}</div>
          {'*************'}
        </div>
      )}

      {settings.authMethod === 'ACCOUNT_KEY' && (
        <>
          <div className="mb-1 flex min-w-0 flex-wrap items-center gap-y-1 break-words">
            <div className="min-w-27.5 pr-2">{t('storages.fields.accountName')}</div>
            {settings.accountName || '-'}
          </div>

          <div className="mb-1 flex min-w-0 flex-wrap items-center gap-y-1 break-words">
            <div className="min-w-27.5 pr-2">{t('storages.fields.accountKey')}</div>
            {'*************'}
          </div>

          <div className="mb-1 flex min-w-0 flex-wrap items-center gap-y-1 break-words">
            <div className="min-w-27.5 pr-2">{t('storages.fields.endpoint')}</div>
            {settings.endpoint || '-'}
          </div>
        </>
      )}

      <div className="mb-1 flex min-w-0 flex-wrap items-center gap-y-1 break-words">
        <div className="min-w-27.5 pr-2">{t('storages.fields.containerName')}</div>
        {settings.containerName || '-'}
      </div>

      {settings.prefix && (
        <div className="mb-1 flex min-w-0 flex-wrap items-center gap-y-1 break-words">
          <div className="min-w-27.5 pr-2">{t('storages.fields.prefix')}</div>
          {settings.prefix}
        </div>
      )}
    </>
  );
}
