import { Button } from 'antd';
import { useState } from 'react';
import { useTranslation } from 'react-i18next';

import { type Database, DatabaseType } from '../../../entities/databases';
import type { TranslationKey } from '../../../shared/i18n';

interface Props {
  database: Database;
  saveButtonText?: string;
  onBack: () => void;
  onSelected: (type: DatabaseType) => void;
}

const backupTypeOptions: {
  type: DatabaseType;
  titleKey: TranslationKey;
  descriptionKey: TranslationKey;
}[] = [
  {
    type: DatabaseType.POSTGRES_LOGICAL,
    titleKey: 'databases.create.backupType.logical.title',
    descriptionKey: 'databases.create.backupType.logical.description',
  },
  {
    type: DatabaseType.POSTGRES_PHYSICAL,
    titleKey: 'databases.create.backupType.physical.title',
    descriptionKey: 'databases.create.backupType.physical.description',
  },
];

export const ChoosePostgresBackupTypeComponent = ({
  database,
  saveButtonText,
  onBack,
  onSelected,
}: Props) => {
  const { t } = useTranslation();
  const [selectedType, setSelectedType] = useState<DatabaseType>(
    database.settings.type === DatabaseType.POSTGRES_PHYSICAL
      ? DatabaseType.POSTGRES_PHYSICAL
      : DatabaseType.POSTGRES_LOGICAL,
  );

  return (
    <div>
      <div className="my-3 text-center text-lg">{t('databases.create.backupType.title')}</div>

      <div className="grid grid-cols-2 gap-3">
        {backupTypeOptions.map((option) => {
          const isSelected = selectedType === option.type;

          return (
            <div
              key={option.type}
              onClick={() => setSelectedType(option.type)}
              className={`flex cursor-pointer flex-col gap-2 rounded-xl border p-3 transition hover:border-blue-400 ${
                isSelected
                  ? 'border-blue-500 bg-blue-50 dark:bg-blue-900/40'
                  : 'border-gray-200 bg-white dark:border-gray-700 dark:bg-gray-800'
              }`}
            >
              <div className="flex items-center gap-2">
                <span className="font-semibold">{t(option.titleKey)}</span>
              </div>

              <span className="text-sm leading-snug text-gray-500 dark:text-gray-400">
                {t(option.descriptionKey)}
              </span>
            </div>
          );
        })}
      </div>

      <div className="mt-5 flex">
        <Button className="mr-auto" type="primary" ghost onClick={onBack}>
          {t('common.actions.back')}
        </Button>

        <Button type="primary" onClick={() => onSelected(selectedType)}>
          {saveButtonText || t('common.actions.continue')}
        </Button>
      </div>
    </div>
  );
};
