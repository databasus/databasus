import { useTranslation } from 'react-i18next';

import type { GoogleDriveStorageSettings } from '@/entities/storages';

interface Props {
  settings: GoogleDriveStorageSettings;
}

export function ShowGoogleDriveStorageComponent({ settings }: Props) {
  const { t } = useTranslation();

  return (
    <>
      <div className="mb-1 flex min-w-0 flex-wrap items-center gap-y-1 break-words">
        <div className="min-w-27.5 pr-2">{t('storages.fields.clientId')}</div>
        {settings.clientId ? `${settings.clientId.slice(0, 10)}***` : '-'}
      </div>

      <div className="mb-1 flex min-w-0 flex-wrap items-center gap-y-1 break-words">
        <div className="min-w-27.5 pr-2">{t('storages.fields.clientSecret')}</div>
        {`*************`}
      </div>

      <div className="mb-1 flex min-w-0 flex-wrap items-center gap-y-1 break-words">
        <div className="min-w-27.5 pr-2">{t('storages.fields.userToken')}</div>
        {`*************`}
      </div>
    </>
  );
}
