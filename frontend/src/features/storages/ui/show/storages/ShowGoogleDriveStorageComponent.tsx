import { useTranslation } from 'react-i18next';

import type { GoogleDriveStorageSettings } from '../../../../../entity/storages';

interface Props {
  settings: GoogleDriveStorageSettings;
}

export function ShowGoogleDriveStorageComponent({ settings }: Props) {
  const { t } = useTranslation();

  return (
    <>
      <div className="mb-1 flex items-center">
        <div className="min-w-[110px] pr-2">{t('storages.fields.clientId')}</div>
        {settings.clientId ? `${settings.clientId.slice(0, 10)}***` : '-'}
      </div>

      <div className="mb-1 flex items-center">
        <div className="min-w-[110px] pr-2">{t('storages.fields.clientSecret')}</div>
        {`*************`}
      </div>

      <div className="mb-1 flex items-center">
        <div className="min-w-[110px] pr-2">{t('storages.fields.userToken')}</div>
        {`*************`}
      </div>
    </>
  );
}
