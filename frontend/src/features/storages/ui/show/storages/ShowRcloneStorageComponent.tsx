import { useTranslation } from 'react-i18next';

import type { RcloneStorageSettings } from '../../../../../entities/storages';

interface Props {
  settings: RcloneStorageSettings;
}

export function ShowRcloneStorageComponent({ settings }: Props) {
  const { t } = useTranslation();

  return (
    <>
      <div className="mb-1 flex items-center">
        <div className="min-w-[110px] pr-2">{t('storages.fields.remotePath')}</div>
        {settings.remotePath || '-'}
      </div>

      <div className="mb-1 flex items-center">
        <div className="min-w-[110px] pr-2">{t('storages.fields.config')}</div>
        {'*************'}
      </div>
    </>
  );
}
