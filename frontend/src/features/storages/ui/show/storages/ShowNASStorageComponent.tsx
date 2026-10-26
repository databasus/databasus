import { useTranslation } from 'react-i18next';

import type { NasStorageSettings } from '../../../../../entity/storages';

interface Props {
  settings: NasStorageSettings;
}

export function ShowNASStorageComponent({ settings }: Props) {
  const { t } = useTranslation();

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
        <div className="min-w-[110px] pr-2">{t('storages.fields.share')}</div>
        {settings.share || '-'}
      </div>

      <div className="mb-1 flex items-center">
        <div className="min-w-[110px] pr-2">{t('common.fields.username')}</div>
        {settings.username || '-'}
      </div>

      <div className="mb-1 flex items-center">
        <div className="min-w-[110px] pr-2">{t('common.fields.password')}</div>
        {'*************'}
      </div>

      <div className="mb-1 flex items-center">
        <div className="min-w-[110px] pr-2">{t('storages.fields.useSsl')}</div>
        {settings.isSsl ? t('common.answers.yes') : t('common.answers.no')}
      </div>

      <div className="mb-1 flex items-center">
        <div className="min-w-[110px] pr-2">{t('storages.fields.domain')}</div>
        {settings.domain || '-'}
      </div>

      <div className="mb-1 flex items-center">
        <div className="min-w-[110px] pr-2">{t('storages.fields.path')}</div>
        {settings.path || '-'}
      </div>
    </>
  );
}
