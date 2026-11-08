import { CircleAlert } from 'lucide-react';
import { useTranslation } from 'react-i18next';

export function EditLocalStorageComponent() {
  const { t } = useTranslation();

  return (
    <>
      <div className="flex max-w-[400px] items-start gap-2 text-warning">
        <CircleAlert aria-hidden size={14} className="mt-1 shrink-0" />
        <span className="min-w-0">{t('storages.edit.local.diskSpaceWarning')}</span>
      </div>

      <div className="mb-5" />
    </>
  );
}
