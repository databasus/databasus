import { CircleAlert } from 'lucide-react';
import { useTranslation } from 'react-i18next';

export function EditLocalStorageComponent() {
  const { t } = useTranslation();

  return (
    <>
      <div className="max-w-[360px] text-warning">
        <CircleAlert size={14} /> {t('storages.edit.local.diskSpaceWarning')}
      </div>

      <div className="mb-5" />
    </>
  );
}
