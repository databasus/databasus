import { useTranslation } from 'react-i18next';

import type { FtpStorageSettings } from '../../../../../entities/storages';

interface Props {
  settings: FtpStorageSettings;
}

export function ShowFTPStorageComponent({ settings }: Props) {
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
        <div className="min-w-27.5 pr-2">{t('common.fields.password')}</div>
        {'*************'}
      </div>

      <div className="mb-1 flex min-w-0 flex-wrap items-center gap-y-1 break-words">
        <div className="min-w-27.5 pr-2">{t('storages.fields.path')}</div>
        {settings.path || '-'}
      </div>

      <div className="mb-1 flex min-w-0 flex-wrap items-center gap-y-1 break-words">
        <div className="min-w-27.5 pr-2">{t('storages.fields.useSslTls')}</div>
        {settings.isSsl ? t('common.answers.yes') : t('common.answers.no')}
      </div>

      {settings.isSsl && settings.isSkipTlsVerify && (
        <div className="mb-1 flex min-w-0 flex-wrap items-center gap-y-1 break-words">
          <div className="min-w-27.5 pr-2">{t('storages.fields.skipTls')}</div>
          {t('common.states.enabled')}
        </div>
      )}
    </>
  );
}
