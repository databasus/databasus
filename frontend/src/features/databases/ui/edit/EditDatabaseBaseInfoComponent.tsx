import { Button, Input } from 'antd';
import { useEffect, useState } from 'react';
import { useTranslation } from 'react-i18next';

import {
  DATABASE_TYPE_LABEL_KEYS,
  type Database,
  DatabaseType,
  changeDatabaseType,
  databaseApi,
  getDatabaseLogoFromType,
  isPostgresType,
} from '../../../../entities/databases';
import { translateApiError } from '../../../../shared/i18n';

interface Props {
  database: Database;

  isShowName?: boolean;
  isShowEngine?: boolean;
  isShowCancelButton?: boolean;
  onCancel: () => void;

  saveButtonText?: string;
  isSaveToApi: boolean;
  onSaved: (db: Database) => void;
}

// PostgreSQL maps to the logical type as a tentative default; the backup-type
// (logical vs physical) is chosen on the next step.
const databaseEngineTypes = [
  DatabaseType.POSTGRES_LOGICAL,
  DatabaseType.MYSQL,
  DatabaseType.MARIADB,
  DatabaseType.MONGODB,
];

export const EditDatabaseBaseInfoComponent = ({
  database,
  isShowName,
  isShowEngine,
  isShowCancelButton,
  onCancel,
  saveButtonText,
  isSaveToApi,
  onSaved,
}: Props) => {
  const { t } = useTranslation();
  const [editingDatabase, setEditingDatabase] = useState<Database>();
  const [isUnsaved, setIsUnsaved] = useState(false);
  const [isSaving, setIsSaving] = useState(false);

  const updateDatabase = (patch: Partial<Database>) => {
    setEditingDatabase((prev) => (prev ? { ...prev, ...patch } : prev));
    setIsUnsaved(true);
  };

  const handleTypeChange = (newType: DatabaseType) => {
    if (!editingDatabase) return;

    setEditingDatabase(changeDatabaseType(editingDatabase, newType));
    setIsUnsaved(true);
  };

  const saveDatabase = async () => {
    if (!editingDatabase) return;

    const editedDatabase = { ...editingDatabase, name: editingDatabase.name.trim() };

    if (!isSaveToApi) {
      onSaved(editedDatabase);
      return;
    }

    setIsSaving(true);

    try {
      onSaved(await databaseApi.updateDatabase(editedDatabase.id, editedDatabase));
      setIsUnsaved(false);
    } catch (e) {
      alert(translateApiError(e, t));
    }

    setIsSaving(false);
  };

  useEffect(() => {
    setIsSaving(false);
    setIsUnsaved(false);
    setEditingDatabase({ ...database });
  }, [database]);

  if (!editingDatabase) return null;

  const isAllFieldsFilled = !!editingDatabase.name?.trim();

  return (
    <div>
      {isShowName && (
        <div className="mb-3 flex items-center">
          <div className="mr-3">{t('common.fields.name')}</div>
          <Input
            value={editingDatabase.name || ''}
            onChange={(e) => updateDatabase({ name: e.target.value })}
            size="small"
            placeholder={t('databases.create.namePlaceholder')}
            className="grow"
          />
        </div>
      )}

      {isShowEngine && (
        <div className="grid grid-cols-2 gap-3">
          {databaseEngineTypes.map((engineType) => {
            const isSelected = isPostgresType(engineType)
              ? isPostgresType(editingDatabase.settings.type)
              : editingDatabase.settings.type === engineType;
            const engineLabel = t(DATABASE_TYPE_LABEL_KEYS[engineType]);

            return (
              <div
                key={engineType}
                onClick={() => handleTypeChange(engineType)}
                className={`flex h-24 cursor-pointer flex-col items-center justify-center gap-2 rounded-xl border text-center text-xs transition hover:border-blue-400 ${
                  isSelected
                    ? 'border-blue-500 bg-blue-50 dark:bg-blue-900/40'
                    : 'border-gray-200 bg-white dark:border-gray-700 dark:bg-gray-800'
                }`}
              >
                <img
                  src={getDatabaseLogoFromType(engineType)}
                  alt={engineLabel}
                  className="h-7 w-7"
                />

                <span className="px-1 leading-tight">{engineLabel}</span>
              </div>
            );
          })}
        </div>
      )}

      <div className="mt-5 flex">
        {isShowCancelButton && (
          <Button danger ghost className="mr-1" onClick={onCancel}>
            {t('common.actions.cancel')}
          </Button>
        )}

        <Button
          type="primary"
          className={isShowCancelButton ? 'ml-1' : 'ml-auto'}
          onClick={saveDatabase}
          loading={isSaving}
          disabled={(isSaveToApi && !isUnsaved) || !isAllFieldsFilled}
        >
          {saveButtonText || t('common.actions.save')}
        </Button>
      </div>
    </div>
  );
};
