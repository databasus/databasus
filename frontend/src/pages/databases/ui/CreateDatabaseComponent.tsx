import { Alert, App, Button } from 'antd';
import { useEffect, useEffectEvent, useState } from 'react';
import { useTranslation } from 'react-i18next';

import { type LogicalBackupConfigInput, backupConfigApi } from '@/entities/backup-configs';
import {
  type Database,
  type DatabaseCreateDraft,
  DatabaseType,
  type PhysicalDatabaseBackupType,
  changeDatabaseCreateDraftType,
  createDefaultDatabaseSettings,
  databaseApi,
  isPostgresType,
} from '@/entities/databases';
import { DatabaseConnectionForm } from '@/features/database-connection';
import { hasSameNotifierIds } from '@/pages/databases/model/hasSameNotifierIds';
import type { PhysicalBackupConfigDraft } from '@/pages/databases/model/physical-backups/PhysicalBackupConfigDraft';
import { toPhysicalBackupConfigInputFromDraft } from '@/pages/databases/model/physical-backups/toPhysicalBackupConfigInputFromDraft';
import { startFirstBackup } from '@/pages/databases/model/startFirstBackup';
import { CreateLogicalBackupConfigStep } from '@/pages/databases/ui/backup-config/CreateLogicalBackupConfigStep';
import { CreatePhysicalBackupConfigStep } from '@/pages/databases/ui/backup-config/CreatePhysicalBackupConfigStep';
import { ChoosePostgresBackupTypeComponent } from '@/pages/databases/ui/ChoosePostgresBackupTypeComponent';
import { CreateDatabaseNotifiersStep } from '@/pages/databases/ui/CreateDatabaseNotifiersStep';
import { CreateLimitedUserComponent } from '@/pages/databases/ui/CreateLimitedUserComponent';
import { EditDatabaseBaseInfoComponent } from '@/pages/databases/ui/EditDatabaseBaseInfoComponent';
import { useSessionGuard } from '@/shared/auth';
import { translateApiError } from '@/shared/i18n';

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

const getPhysicalBackupType = (
  draft: DatabaseCreateDraft,
): PhysicalDatabaseBackupType | undefined =>
  draft.settings.type === DatabaseType.POSTGRES_PHYSICAL ? draft.settings.backupType : undefined;

export const CreateDatabaseComponent = ({
  workspaceId,
  onCreated,
  onClose,
  onConnectionErrorChange,
  onWideStepChange,
}: Props) => {
  const { t } = useTranslation();
  const { message } = App.useApp();
  const [isCompleting, setIsCompleting] = useState(false);
  const [createdDatabase, setCreatedDatabase] = useState<Database>();
  const [isBackupConfigAcknowledged, setIsBackupConfigAcknowledged] = useState(false);
  const [completionError, setCompletionError] = useState<unknown>();
  const [backupConfig, setBackupConfig] = useState<LogicalBackupConfigInput | undefined>();
  const [physicalBackupDraft, setPhysicalBackupDraft] = useState<
    PhysicalBackupConfigDraft | undefined
  >();
  const [acceptedPhysicalBackupType, setAcceptedPhysicalBackupType] = useState<
    PhysicalDatabaseBackupType | undefined
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
    setPhysicalBackupDraft(undefined);

    return changeDatabaseCreateDraftType(draft, type);
  };

  // A physical backup configuration holds schedules and retention of one backup type, so only a type
  // that differs when the user continues starts it over.
  const resetPhysicalBackupConfigIfTypeChanged = () => {
    if (getPhysicalBackupType(draft) !== acceptedPhysicalBackupType) {
      setPhysicalBackupDraft(undefined);
    }
  };

  const acceptPhysicalBackupDraft = (acceptedDraft: PhysicalBackupConfigDraft) => {
    setPhysicalBackupDraft(acceptedDraft);
    setAcceptedPhysicalBackupType(getPhysicalBackupType(draft));
  };

  // Every step records its result before the next one starts, so a retry repeats only unfinished
  // work. A closed wizard starts no later step and reports nothing.
  const completeWizard = async (completedDraft: DatabaseCreateDraft) => {
    const backupConfigToSave = isPhysical
      ? physicalBackupDraft && toPhysicalBackupConfigInputFromDraft(physicalBackupDraft)
      : backupConfig;
    if (!backupConfigToSave) return;

    setIsCompleting(true);
    setCompletionError(undefined);

    let database = createdDatabase;

    try {
      if (!database) {
        database = await databaseApi.createDatabase(completedDraft);
        if (!isCurrent()) return;

        setCreatedDatabase(database);
        onCreated(database);
      }

      if (
        !hasSameNotifierIds({
          acknowledgedIds: database.notifierIds,
          shownIds: completedDraft.notifierIds,
        })
      ) {
        database = await databaseApi.updateDatabase(database.id, {
          ...database,
          notifierIds: completedDraft.notifierIds,
        });
        if (!isCurrent()) return;

        setCreatedDatabase(database);
      }

      if (!isBackupConfigAcknowledged) {
        await backupConfigApi.updateBackupConfig(database.id, backupConfigToSave);
        if (!isCurrent()) return;

        setIsBackupConfigAcknowledged(true);
      }

      await startFirstBackup(database.id, backupConfigToSave);
      if (!isCurrent()) return;

      onClose();
    } catch (error) {
      if (!isCurrent()) return;

      if (database) setCompletionError(error);
      else message.error(translateApiError(error, t));
    }

    setIsCompleting(false);
  };

  const isCurrent = useSessionGuard();

  // The read-only user step explains its choice in long sentences and puts three buttons in one
  // row, which the narrow wizard width cannot hold.
  const reportWideStep = useEffectEvent((isWideStep: boolean) => {
    onWideStepChange?.(isWideStep);
  });

  useEffect(() => {
    reportWideStep(step === 'create-limited-user');
  }, [step]);

  const physicalBackupType = getPhysicalBackupType(draft);
  const isPhysical = physicalBackupType !== undefined;
  const isPostgres = isPostgresType(draft.settings.type);
  const completionAlert = completionError !== undefined && (
    <Alert
      className="mb-5"
      type="error"
      showIcon
      message={t('databases.create.createdButNotFinished', {
        reason: translateApiError(completionError, t),
      })}
    />
  );

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
        onSubmit={() => {
          resetPhysicalBackupConfigIfTypeChanged();
          setStep('create-limited-user');
        }}
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
    return (
      <>
        {completionAlert}

        {physicalBackupType ? (
          <CreatePhysicalBackupConfigStep
            workspaceId={draft.workspaceId}
            backupType={physicalBackupType}
            initialDraft={physicalBackupDraft}
            onBack={
              createdDatabase
                ? undefined
                : (physicalDraft) => {
                    acceptPhysicalBackupDraft(physicalDraft);
                    setStep('db-settings');
                  }
            }
            onContinue={(physicalDraft) => {
              acceptPhysicalBackupDraft(physicalDraft);
              setStep('notifiers');
            }}
          />
        ) : (
          <CreateLogicalBackupConfigStep
            workspaceId={draft.workspaceId}
            initialConfig={backupConfig}
            onBack={
              createdDatabase
                ? undefined
                : (config) => {
                    setBackupConfig(config);
                    setStep('db-settings');
                  }
            }
            onContinue={(config) => {
              setBackupConfig(config);
              setStep('notifiers');
            }}
          />
        )}
      </>
    );
  }

  if (step === 'notifiers') {
    if (isCompleting) {
      return (
        <div>{t(createdDatabase ? 'databases.create.finishing' : 'databases.create.creating')}</div>
      );
    }

    return (
      <>
        {completionAlert}

        <CreateDatabaseNotifiersStep
          workspaceId={workspaceId}
          initialNotifierIds={draft.notifierIds}
          onBack={
            isBackupConfigAcknowledged
              ? undefined
              : (notifierIds) => {
                  setDraft({ ...draft, notifierIds });
                  setStep('backup-config');
                }
          }
          onContinue={(notifierIds) => {
            if (isCompleting) return;

            const completedDraft = { ...draft, notifierIds };
            setDraft(completedDraft);
            void completeWizard(completedDraft);
          }}
        />
      </>
    );
  }
};
