import { useTranslation } from 'react-i18next';

import { SFTP_AUTH_METHOD_LABEL_KEYS, type SftpStorageSettings } from '@/entities/storages';

interface Props {
  settings: SftpStorageSettings;
}

export function ShowSFTPStorageComponent({ settings }: Props) {
  const { t } = useTranslation();

  return (
    <>
      <div className="mb-1 flex min-w-0 flex-wrap items-center gap-y-1 break-words">
        <div className="min-w-27.5 pr-2">{t('common.fields.host')}</div>
        {settings.host || '-'}
      </div>

      <div className="mb-1 flex min-w-0 flex-wrap items-center gap-y-1 break-words">
        <div className="min-w-27.5 pr-2">{t('common.fields.port')}</div>
        {settings.port || '-'}
      </div>

      <div className="mb-1 flex min-w-0 flex-wrap items-center gap-y-1 break-words">
        <div className="min-w-27.5 pr-2">{t('common.fields.username')}</div>
        {settings.username || '-'}
      </div>

      <div className="mb-1 flex min-w-0 flex-wrap items-center gap-y-1 break-words">
        <div className="min-w-27.5 pr-2">{t('storages.fields.authMethod')}</div>
        {settings.authMethods.map((method) => (
          <div key={method}>{t(SFTP_AUTH_METHOD_LABEL_KEYS[method])}</div>
        ))}
      </div>

      <div className="mb-1 flex min-w-0 flex-wrap items-center gap-y-1 break-words">
        <div className="min-w-27.5 pr-2">{t('storages.fields.credentials')}</div>
        {'*************'}
      </div>

      <div className="mb-1 flex min-w-0 flex-wrap items-center gap-y-1 break-words">
        <div className="min-w-27.5 pr-2">{t('storages.fields.path')}</div>
        {settings.path || '-'}
      </div>

      {settings.isSkipHostKeyVerify && (
        <div className="mb-1 flex min-w-0 flex-wrap items-center gap-y-1 break-words">
          <div className="min-w-27.5 pr-2">{t('storages.fields.skipHostKey')}</div>
          {t('common.states.enabled')}
        </div>
      )}
    </>
  );
}
