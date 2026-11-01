import { useEffect, useState } from 'react';
import { useTranslation } from 'react-i18next';

import { EditLogicalBackupConfigComponent } from '@/features/backups/logical';
import { EditPhysicalBackupConfigComponent } from '@/features/backups/physical';
import {
  ChoosePostgresBackupTypeComponent,
  CreateReadOnlyComponent,
  EditDatabaseBaseInfoComponent,
  EditDatabaseSpecificDataComponent,
} from '@/features/databases';

import {
  type LogicalBackupConfigInput,
  type PhysicalBackupConfigInput,
  backupConfigApi,
} from '../../../entities/backups/config';
import { logicalBackupApi } from '../../../entities/backups/logical';
import { physicalBackupsApi } from '../../../entities/backups/physical';
import {
  type Database,
  DatabaseType,
  changeDatabaseType,
  createDefaultDatabaseSettings,
  databaseApi,
  isPostgresType,
} from '../../../entities/databases';
import { translateApiError } from '../../../shared/i18n';
import { BackupStorageSelection } from './BackupStorageSelection';
import { EditDatabaseNotifiersComponent } from './EditDatabaseNotifiersComponent';

interface Props {
  workspaceId: string;
  onCreated: (database: Database) => void;
  onClose: () => void;
  onConnectionErrorChange?: (hasConnectionError: boolean) => void;
  onWideStepChange?: (isWideStep: boolean) => void;
}

// The wizard fills a Database before it exists; an empty ID marks it as not created yet.
const createInitialDatabase = (workspaceId: string): Database => ({
  id: '',
  name: '',
  workspaceId,
  settings: createDefaultDatabaseSettings(DatabaseType.POSTGRES_LOGICAL),
  notifierIds: [],
});

export const CreateDatabaseComponent = ({
  workspaceId,
  onCreated,
  onClose,
  onConnectionErrorChange,
  onWideStepChange,
}: Props) => {
  const { t } = useTranslation();
  const [isCreating, setIsCreating] = useState(false);
  const [backupConfig, setBackupConfig] = useState<LogicalBackupConfigInput | undefined>();
  const [physicalBackupConfig, setPhysicalBackupConfig] = useState<
    PhysicalBackupConfigInput | undefined
  >();
  const [database, setDatabase] = useState<Database>(createInitialDatabase(workspaceId));

  const [step, setStep] = useState<
    | 'base-info'
    | 'postgres-backup-type'
    | 'db-settings'
    | 'create-limited-user'
    | 'backup-config'
    | 'notifiers'
  >('base-info');

  // A backup configuration belongs to one database type, so a type change starts it over.
  const acceptDatabaseType = (typedDatabase: Database) => {
    if (typedDatabase.settings.type !== database.settings.type) {
      setBackupConfig(undefined);
      setPhysicalBackupConfig(undefined);
    }

    setDatabase(typedDatabase);
  };

  const createDatabase = async (database: Database) => {
    if (isPhysical ? !physicalBackupConfig : !backupConfig) return;

    setIsCreating(true);

    try {
      const createdDatabase = await databaseApi.createDatabase(database);
      setDatabase({ ...createdDatabase });

      if (isPhysical && physicalBackupConfig) {
        await backupConfigApi.updateBackupConfig(createdDatabase.id, physicalBackupConfig);

        if (physicalBackupConfig.isBackupsEnabled) {
          await physicalBackupsApi.triggerPhysicalBackup(createdDatabase.id, 'AUTO');
        }
      } else if (backupConfig) {
        await backupConfigApi.updateBackupConfig(createdDatabase.id, backupConfig);

        if (backupConfig.isBackupsEnabled) {
          await logicalBackupApi.startLogicalBackup(createdDatabase.id);
        }
      }

      onCreated(createdDatabase);
      onClose();
    } catch (error) {
      alert(translateApiError(error, t));
    }

    setIsCreating(false);
  };

  // The read-only user step explains its choice in long sentences and puts three buttons in one
  // row, which the narrow wizard width cannot hold.
  useEffect(() => {
    onWideStepChange?.(step === 'create-limited-user');
  }, [step]);

  const isPhysical = database.settings.type === DatabaseType.POSTGRES_PHYSICAL;
  const isPostgres = isPostgresType(database.settings.type);

  if (step === 'base-info') {
    return (
      <div>
        <EditDatabaseBaseInfoComponent
          database={database}
          isShowName
          isShowEngine
          isSaveToApi={false}
          saveButtonText={t('common.actions.continue')}
          onCancel={() => onClose()}
          onSaved={(db) => {
            acceptDatabaseType(db);
            setStep(isPostgresType(db.settings.type) ? 'postgres-backup-type' : 'db-settings');
          }}
        />
      </div>
    );
  }

  if (step === 'postgres-backup-type') {
    return (
      <ChoosePostgresBackupTypeComponent
        database={database}
        saveButtonText={t('common.actions.continue')}
        onBack={() => setStep('base-info')}
        onSelected={(type) => {
          acceptDatabaseType(changeDatabaseType(database, type));
          setStep('db-settings');
        }}
      />
    );
  }

  if (step === 'db-settings') {
    return (
      <EditDatabaseSpecificDataComponent
        database={database}
        isShowCancelButton={false}
        onCancel={() => onClose()}
        isShowBackButton
        onBack={() => setStep(isPostgres ? 'postgres-backup-type' : 'base-info')}
        saveButtonText={t('common.actions.continue')}
        isSaveToApi={false}
        onConnectionErrorChange={onConnectionErrorChange}
        onSaved={(database) => {
          setDatabase({ ...database });
          setStep('create-limited-user');
        }}
      />
    );
  }

  if (step === 'create-limited-user') {
    return (
      <CreateReadOnlyComponent
        database={database}
        onReadOnlyUserUpdated={(database) => {
          setDatabase({ ...database });
          setStep('backup-config');
        }}
        onGoBack={() => setStep('db-settings')}
        onSkipped={() => setStep('backup-config')}
        onReadOnlyUserNotSuggested={() => setStep('backup-config')}
      />
    );
  }

  if (step === 'backup-config') {
    if (isPhysical) {
      return (
        <EditPhysicalBackupConfigComponent
          renderStorageSelection={(options) => <BackupStorageSelection {...options} />}
          database={database}
          initialConfig={physicalBackupConfig}
          isShowCancelButton={false}
          onCancel={() => onClose()}
          isShowBackButton
          onBack={() => setStep('db-settings')}
          saveButtonText={t('common.actions.continue')}
          isSaveToApi={false}
          onSaved={(physicalBackupConfig) => {
            setPhysicalBackupConfig(physicalBackupConfig);
            setStep('notifiers');
          }}
        />
      );
    }

    return (
      <EditLogicalBackupConfigComponent
        renderStorageSelection={(options) => <BackupStorageSelection {...options} />}
        database={database}
        initialConfig={backupConfig}
        isShowCancelButton={false}
        onCancel={() => onClose()}
        isShowBackButton
        onBack={() => setStep('db-settings')}
        saveButtonText={t('common.actions.continue')}
        isSaveToApi={false}
        onSaved={(backupConfig) => {
          setBackupConfig(backupConfig);
          setStep('notifiers');
        }}
      />
    );
  }

  if (step === 'notifiers') {
    if (isCreating) {
      return <div>{t('databases.create.creating')}</div>;
    }

    return (
      <EditDatabaseNotifiersComponent
        database={database}
        isShowCancelButton={false}
        workspaceId={workspaceId}
        onCancel={() => onClose()}
        isShowBackButton
        onBack={() => setStep('backup-config')}
        isShowSaveOnlyForUnsaved={false}
        saveButtonText={t('databases.create.complete')}
        isSaveToApi={false}
        onSaved={(database) => {
          if (isCreating) return;

          setDatabase({ ...database });
          createDatabase(database);
        }}
      />
    );
  }
};
