import { useTranslation } from 'react-i18next';

import type { RcloneStorageSettings } from '../../../../../entities/storages';

interface Props {
  settings: RcloneStorageSettings;
}

export function ShowRcloneStorageComponent({ settings }: Props) {
  const { t } = useTranslation();

  return (
    <>
      <div className="mb-1 flex min-w-0 flex-wrap items-center gap-y-1 break-words">
        <div className="min-w-27.5 pr-2">{t('storages.fields.remotePath')}</div>
        {settings.remotePath || '-'}
      </div>

      <div className="mb-1 flex min-w-0 flex-wrap items-center gap-y-1 break-words">
        <div className="min-w-27.5 pr-2">{t('storages.fields.config')}</div>
        {'*************'}
      </div>
    </>
  );
}
