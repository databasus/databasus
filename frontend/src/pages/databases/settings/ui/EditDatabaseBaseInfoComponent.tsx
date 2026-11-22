import { Button, TextInput } from '@mantine/core';
import { useId, useState } from 'react';
import { useTranslation } from 'react-i18next';

import {
  DATABASE_TYPE_LABEL_KEYS,
  DatabaseType,
  MANAGED_POSTGRESQL_PROVIDERS,
  type ManagedPostgresqlProvider,
  getDatabaseLogoFromType,
  isPostgresType,
} from '@/entities/databases';

interface Props {
  name: string;
  type: DatabaseType;
  managedProvider?: ManagedPostgresqlProvider;
  continueButtonText: string;
  onContinue: (
    name: string,
    type: DatabaseType,
    managedProvider?: ManagedPostgresqlProvider,
  ) => void;
}

// PostgreSQL maps to the logical type as a tentative default; the backup-type
// (logical vs physical) is chosen on the next step.
const databaseEngineTypes = [
  DatabaseType.POSTGRES_LOGICAL,
  DatabaseType.MYSQL,
  DatabaseType.MARIADB,
  DatabaseType.MONGODB,
];

const managedProviders: ManagedPostgresqlProvider[] = ['neon', 'supabase'];

const EngineCard = ({
  label,
  logo,
  isSelected,
  onSelect,
}: {
  label: string;
  logo: string;
  isSelected: boolean;
  onSelect: () => void;
}) => (
  <button
    type="button"
    aria-pressed={isSelected}
    onClick={onSelect}
    className={`flex min-h-24 min-w-0 cursor-pointer flex-col items-center justify-center gap-2 rounded-xl border text-center text-xs transition hover:border-accent ${isSelected ? 'border-accent bg-selected' : 'border-border bg-surface'}`}
  >
    <img src={logo} alt="" className="h-7 w-7" />

    <span className="px-1 leading-tight [overflow-wrap:anywhere]">{label}</span>
  </button>
);

export const EditDatabaseBaseInfoComponent = ({
  name,
  type,
  managedProvider,
  continueButtonText,
  onContinue,
}: Props) => {
  const { t } = useTranslation();
  const nameFieldId = useId();
  const [editingName, setEditingName] = useState(name);
  const [selectedType, setSelectedType] = useState(type);
  const [selectedManagedProvider, setSelectedManagedProvider] = useState(managedProvider);

  // Choosing PostgreSQL again returns to the backup type the database already has, so a selection
  // the user reversed changes nothing.
  const selectEngine = (engineType: DatabaseType) => {
    const isPlainPostgresSelected = isPostgresType(selectedType) && !selectedManagedProvider;

    setSelectedManagedProvider(undefined);

    if (isPostgresType(engineType) && isPlainPostgresSelected) return;

    setSelectedType(isPostgresType(engineType) && isPostgresType(type) ? type : engineType);
  };

  // Managed providers accept no physical replication connections, so they always back up logically.
  const selectManagedProvider = (provider: ManagedPostgresqlProvider) => {
    setSelectedManagedProvider(provider);
    setSelectedType(DatabaseType.POSTGRES_LOGICAL);
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
        {databaseEngineTypes.map((engineType) => (
          <EngineCard
            key={engineType}
            label={t(DATABASE_TYPE_LABEL_KEYS[engineType])}
            logo={getDatabaseLogoFromType(engineType)}
            isSelected={
              isPostgresType(engineType)
                ? isPostgresType(selectedType) && !selectedManagedProvider
                : selectedType === engineType
            }
            onSelect={() => selectEngine(engineType)}
          />
        ))}

        {managedProviders.map((provider) => (
          <EngineCard
            key={provider}
            label={MANAGED_POSTGRESQL_PROVIDERS[provider].name}
            logo={MANAGED_POSTGRESQL_PROVIDERS[provider].logo}
            isSelected={selectedManagedProvider === provider}
            onSelect={() => selectManagedProvider(provider)}
          />
        ))}
      </div>

      <div className="mt-5 flex">
        <Button
          variant="filled"
          className="ml-auto"
          onClick={() => onContinue(editingName.trim(), selectedType, selectedManagedProvider)}
          disabled={!isAllFieldsFilled}
        >
          {continueButtonText}
        </Button>
      </div>
    </div>
  );
};
