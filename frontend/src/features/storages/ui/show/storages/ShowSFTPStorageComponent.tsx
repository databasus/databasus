import { useTranslation } from 'react-i18next';

import type { SftpStorageSettings } from '../../../../../entities/storages';

interface Props {
  settings: SftpStorageSettings;
}

export function ShowSFTPStorageComponent({ settings }: Props) {
  const { t } = useTranslation();

  const authMethodLabel = settings.privateKey
    ? t('storages.sftpAuthMethods.privateKey')
    : t('storages.sftpAuthMethods.password');

  return (
    <>
      <div className="mb-1 flex items-center">
        <div className="min-w-[110px] pr-2">{t('common.fields.host')}</div>
        {settings.host || '-'}
      </div>

      <div className="mb-1 flex items-center">
        <div className="min-w-[110px] pr-2">{t('common.fields.port')}</div>
        {settings.port || '-'}
      </div>

      <div className="mb-1 flex items-center">
        <div className="min-w-[110px] pr-2">{t('common.fields.username')}</div>
        {settings.username || '-'}
      </div>

      <div className="mb-1 flex items-center">
        <div className="min-w-[110px] pr-2">{t('storages.fields.authMethod')}</div>
        {authMethodLabel}
      </div>

      <div className="mb-1 flex items-center">
        <div className="min-w-[110px] pr-2">{t('storages.fields.credentials')}</div>
        {'*************'}
      </div>

      <div className="mb-1 flex items-center">
        <div className="min-w-[110px] pr-2">{t('storages.fields.path')}</div>
        {settings.path || '-'}
      </div>

      {settings.isSkipHostKeyVerify && (
        <div className="mb-1 flex items-center">
          <div className="min-w-[110px] pr-2">{t('storages.fields.skipHostKey')}</div>
          {t('common.states.enabled')}
        </div>
      )}
    </>
  );
}
