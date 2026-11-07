import { Button, TextInput } from '@mantine/core';
import { useId, useState } from 'react';
import { useTranslation } from 'react-i18next';

import {
  DATABASE_TYPE_LABEL_KEYS,
  DatabaseType,
  getDatabaseLogoFromType,
  isPostgresType,
} from '@/entities/databases';

interface Props {
  name: string;
  type: DatabaseType;
  continueButtonText: string;
  onContinue: (name: string, type: DatabaseType) => void;
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
  name,
  type,
  continueButtonText,
  onContinue,
}: Props) => {
  const { t } = useTranslation();
  const nameFieldId = useId();
  const [editingName, setEditingName] = useState(name);
  const [selectedType, setSelectedType] = useState(type);

  // Choosing PostgreSQL again returns to the backup type the database already has, so a selection
  // the user reversed changes nothing.
  const selectEngine = (engineType: DatabaseType) => {
    if (isPostgresType(engineType) && isPostgresType(selectedType)) return;

    setSelectedType(isPostgresType(engineType) && isPostgresType(type) ? type : engineType);
  };

  const isAllFieldsFilled = !!editingName.trim();

  return (
    <div>
      <div className="mb-3 flex items-center">
        <label className="mr-3" htmlFor={nameFieldId}>
          {t('common.fields.name')}
        </label>
        <TextInput
          id={nameFieldId}
          value={editingName}
          onChange={(e) => setEditingName(e.target.value)}
          placeholder={t('databases.create.namePlaceholder')}
          className="grow"
        />
      </div>

      <div className="grid grid-cols-2 gap-3">
        {databaseEngineTypes.map((engineType) => {
          const isSelected = isPostgresType(engineType)
            ? isPostgresType(selectedType)
            : selectedType === engineType;
          const engineLabel = t(DATABASE_TYPE_LABEL_KEYS[engineType]);

          return (
            <button
              type="button"
              key={engineType}
              aria-pressed={isSelected}
              onClick={() => selectEngine(engineType)}
              className={`flex min-h-24 min-w-0 cursor-pointer flex-col items-center justify-center gap-2 rounded-xl border text-center text-xs transition hover:border-accent ${isSelected ? 'border-accent bg-selected' : 'border-border bg-surface'}`}
            >
              <img src={getDatabaseLogoFromType(engineType)} alt="" className="h-7 w-7" />

              <span className="px-1 leading-tight [overflow-wrap:anywhere]">{engineLabel}</span>
            </button>
          );
        })}
      </div>

      <div className="mt-5 flex">
        <Button
          variant="filled"
          className="ml-auto"
          onClick={() => onContinue(editingName.trim(), selectedType)}
          disabled={!isAllFieldsFilled}
        >
          {continueButtonText}
        </Button>
      </div>
    </div>
  );
};
