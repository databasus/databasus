import { App, Button } from 'antd';
import { useEffect, useState } from 'react';
import { useTranslation } from 'react-i18next';

import { EditPhysicalBackupConfigComponent } from '@/features/backups/physical';
import { DatabaseConnectionForm } from '@/features/database-connection';

import {
  type LogicalBackupConfigInput,
  type PhysicalBackupConfigInput,
  backupConfigApi,
} from '../../../entities/backup-configs';
import { physicalBackupsApi } from '../../../entities/backups/physical';
import {
  type Database,
  type DatabaseCreateDraft,
  DatabaseType,
  changeDatabaseCreateDraftType,
  createDefaultDatabaseSettings,
  databaseApi,
  isPostgresType,
} from '../../../entities/databases';
import { translateApiError } from '../../../shared/i18n';
import { logicalBackupApi } from '../api/logicalBackupApi';
import { CreateLogicalBackupConfigStep } from './backup-config/CreateLogicalBackupConfigStep';
import { BackupStorageSelection } from './BackupStorageSelection';
import { ChoosePostgresBackupTypeComponent } from './ChoosePostgresBackupTypeComponent';
import { CreateLimitedUserComponent } from './CreateLimitedUserComponent';
import { EditDatabaseBaseInfoComponent } from './EditDatabaseBaseInfoComponent';
import { EditDatabaseNotifiersComponent } from './EditDatabaseNotifiersComponent';

interface Props {
  workspaceId: string;
  onCreated: (database: Database) => void;
  onClose: () => void;
  onConnectionErrorChange?: (hasConnectionError: boolean) => void;
  onWideStepChange?: (isWideStep: boolean) => void;
}

const createInitialDraft = (workspaceId: string): DatabaseCreateDraft => ({
  scenario: 'create',
  workspaceId,
  name: '',
  notifierIds: [],
  settings: createDefaultDatabaseSettings(DatabaseType.POSTGRES_LOGICAL),
  hasUserChosenSslMode: false,
});

// The physical backup config and notifier editors take a Database and read an empty ID as not
// created yet.
const toUnsavedDatabase = (draft: DatabaseCreateDraft): Database => ({
  id: '',
  name: draft.name,
  workspaceId: draft.workspaceId,
  settings: draft.settings,
  notifierIds: draft.notifierIds,
});

export const CreateDatabaseComponent = ({
  workspaceId,
  onCreated,
  onClose,
  onConnectionErrorChange,
  onWideStepChange,
}: Props) => {
  const { t } = useTranslation();
  const { message } = App.useApp();
  const [isCreating, setIsCreating] = useState(false);
  const [backupConfig, setBackupConfig] = useState<LogicalBackupConfigInput | undefined>();
  const [physicalBackupConfig, setPhysicalBackupConfig] = useState<
    PhysicalBackupConfigInput | undefined
  >();
  const [draft, setDraft] = useState(() => createInitialDraft(workspaceId));

  const [step, setStep] = useState<
    | 'base-info'
    | 'postgres-backup-type'
    | 'db-settings'
    | 'create-limited-user'
    | 'backup-config'
    | 'notifiers'
  >('base-info');

  // Settings and a backup configuration belong to one database type, so only a type that differs
  // when the user continues starts them over.
  const acceptDatabaseType = (type: DatabaseType) => {
    if (type === draft.settings.type) return draft;

    setBackupConfig(undefined);
    setPhysicalBackupConfig(undefined);

    return changeDatabaseCreateDraftType(draft, type);
  };

  const createDatabase = async (draftToCreate: DatabaseCreateDraft) => {
    if (isPhysical ? !physicalBackupConfig : !backupConfig) return;

    setIsCreating(true);

    try {
      const createdDatabase = await databaseApi.createDatabase(draftToCreate);

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
      message.error(translateApiError(error, t));
    }

    setIsCreating(false);
  };

  // The read-only user step explains its choice in long sentences and puts three buttons in one
  // row, which the narrow wizard width cannot hold.
  useEffect(() => {
    onWideStepChange?.(step === 'create-limited-user');
  }, [step]);

  const isPhysical = draft.settings.type === DatabaseType.POSTGRES_PHYSICAL;
  const isPostgres = isPostgresType(draft.settings.type);

  if (step === 'base-info') {
    return (
      <div>
        <EditDatabaseBaseInfoComponent
          name={draft.name}
          type={draft.settings.type}
          continueButtonText={t('common.actions.continue')}
          onContinue={(name, type) => {
            setDraft({ ...acceptDatabaseType(type), name });
            setStep(isPostgresType(type) ? 'postgres-backup-type' : 'db-settings');
          }}
        />
      </div>
    );
  }

  if (step === 'postgres-backup-type') {
    return (
      <ChoosePostgresBackupTypeComponent
        type={draft.settings.type}
        continueButtonText={t('common.actions.continue')}
        onBack={() => setStep('base-info')}
        onSelected={(type) => {
          setDraft(acceptDatabaseType(type));
          setStep('db-settings');
        }}
      />
    );
  }

  if (step === 'db-settings') {
    return (
      <DatabaseConnectionForm
        draft={draft}
        onDraftChange={setDraft}
        submitLabel={t('common.actions.continue')}
        secondaryActions={
          <Button
            className="mr-auto"
            type="primary"
            ghost
            onClick={() => setStep(isPostgres ? 'postgres-backup-type' : 'base-info')}
          >
            {t('common.actions.back')}
          </Button>
        }
        onConnectionErrorChange={onConnectionErrorChange}
        onSubmit={() => setStep('create-limited-user')}
      />
    );
  }

  if (step === 'create-limited-user') {
    return (
      <CreateLimitedUserComponent
        draft={draft}
        onLimitedUserCreated={(limitedUser) => {
          setDraft({
            ...draft,
            settings: {
              ...draft.settings,
              username: limitedUser.username,
              password: limitedUser.password,
            },
          });
          setStep('backup-config');
        }}
        onGoBack={() => setStep('db-settings')}
        onSkipped={() => setStep('backup-config')}
        onLimitedUserNotSuggested={() => setStep('backup-config')}
      />
    );
  }

  if (step === 'backup-config') {
    if (isPhysical) {
      return (
        <EditPhysicalBackupConfigComponent
          renderStorageSelection={(options) => <BackupStorageSelection {...options} />}
          database={toUnsavedDatabase(draft)}
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
      <CreateLogicalBackupConfigStep
        workspaceId={draft.workspaceId}
        initialConfig={backupConfig}
        onBack={(config) => {
          setBackupConfig(config);
          setStep('db-settings');
        }}
        onContinue={(config) => {
          setBackupConfig(config);
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
        database={toUnsavedDatabase(draft)}
        isShowCancelButton={false}
        workspaceId={workspaceId}
        onCancel={() => onClose()}
        isShowBackButton
        onBack={(notifierIds) => {
          setDraft({ ...draft, notifierIds });
          setStep('backup-config');
        }}
        isShowSaveOnlyForUnsaved={false}
        saveButtonText={t('databases.create.complete')}
        isSaveToApi={false}
        onSaved={({ notifierIds }) => {
          if (isCreating) return;

          const completedDraft = { ...draft, notifierIds };
          setDraft(completedDraft);
          createDatabase(completedDraft);
        }}
      />
    );
  }
};
