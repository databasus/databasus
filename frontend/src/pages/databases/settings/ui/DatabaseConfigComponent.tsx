import { ActionIcon, Alert, Button, Loader, TextInput } from '@mantine/core';
import { ArrowRight, CircleAlert, Copy, Info, Trash2, X } from 'lucide-react';
import { useId, useState } from 'react';
import { useTranslation } from 'react-i18next';

import {
  type BackupConfig,
  toLogicalBackupConfigInput,
  toPhysicalBackupConfigInput,
} from '@/entities/backup-configs';
import { type Database, DatabaseType, databaseApi } from '@/entities/databases';
import { EditLogicalBackupConfigSection } from '@/pages/databases/backup-config/ui/EditLogicalBackupConfigSection';
import { EditPhysicalBackupConfigSection } from '@/pages/databases/backup-config/ui/EditPhysicalBackupConfigSection';
import { ShowLogicalBackupConfigComponent } from '@/pages/databases/backup-config/ui/ShowLogicalBackupConfigComponent';
import { ShowPhysicalBackupConfigComponent } from '@/pages/databases/backup-config/ui/ShowPhysicalBackupConfigComponent';
import { StorageCleanupNotice } from '@/pages/databases/backup-config/ui/StorageCleanupNotice';
import { useBackupCleanup } from '@/pages/databases/backup-config/ui/useBackupCleanup';
import { useBackupConfigLoad } from '@/pages/databases/backup-config/ui/useBackupConfigLoad';
import {
  EditHealthcheckConfigComponent,
  ShowHealthcheckConfigComponent,
} from '@/pages/databases/healthcheck/ui';
import { DatabaseSettingsSection } from '@/pages/databases/settings/ui/DatabaseSettingsSection';
import { DatabaseTransferDialogComponent } from '@/pages/databases/settings/ui/DatabaseTransferDialogComponent';
import { EditDatabaseConnectionSection } from '@/pages/databases/settings/ui/EditDatabaseConnectionSection';
import { EditDatabaseNotifiersSection } from '@/pages/databases/settings/ui/EditDatabaseNotifiersSection';
import { ShowDatabaseNotifiersComponent } from '@/pages/databases/settings/ui/show/ShowDatabaseNotifiersComponent';
import { ShowDatabaseSpecificDataComponent } from '@/pages/databases/settings/ui/show/ShowDatabaseSpecificDataComponent';
import { EditIconButton } from '@/pages/databases/ui/EditIconButton';
import { EditVerificationConfigComponent } from '@/pages/databases/verification/ui/EditVerificationConfigComponent';
import { ShowVerificationConfigComponent } from '@/pages/databases/verification/ui/ShowVerificationConfigComponent';
import { useSessionMutation } from '@/shared/api';
import { useSessionGuard } from '@/shared/auth';
import { translateApiError } from '@/shared/i18n';
import { useFeedback } from '@/shared/lib/feedback';
import { ConfirmationComponent, LoadFailure } from '@/shared/ui';

interface Props {
  database: Database;
  onDatabaseChanged: (database: Database) => void;
  onDatabaseDeleted: (deletedDatabaseId: string) => void;
  onDatabaseCopied: (copiedDatabase: Database, sourceDatabaseId: string) => void;
  onDatabaseTransferred: (transferredDatabaseId: string) => void;

  isCanManageDBs: boolean;
}

type DatabaseSettingsSectionId =
  | 'name'
  | 'connection'
  | 'backup-config'
  | 'healthcheck'
  | 'verification'
  | 'notifiers';

export const DatabaseConfigComponent = ({
  database,
  onDatabaseChanged,
  onDatabaseDeleted,
  onDatabaseCopied,
  onDatabaseTransferred,
  isCanManageDBs,
}: Props) => {
  const copyDatabaseMutation = useSessionMutation({
    mutationFn: (parameters: Parameters<typeof databaseApi.copyDatabase>) =>
      databaseApi.copyDatabase(...parameters),
  });
  const testDatabaseMutation = useSessionMutation({
    mutationFn: (parameters: Parameters<typeof databaseApi.testDatabase>) =>
      databaseApi.testDatabase(...parameters),
  });
  const deleteDatabaseMutation = useSessionMutation({
    mutationFn: (parameters: Parameters<typeof databaseApi.deleteDatabase>) =>
      databaseApi.deleteDatabase(...parameters),
  });
  const updateDatabaseMutation = useSessionMutation({
    mutationFn: (parameters: Parameters<typeof databaseApi.updateDatabase>) =>
      databaseApi.updateDatabase(...parameters),
  });

  const { t } = useTranslation();
  const notify = useFeedback();
  const isActive = useSessionGuard();
  const databaseNameId = useId();

  const [editingSection, setEditingSection] = useState<DatabaseSettingsSectionId>();
  const [nameDraft, setNameDraft] = useState('');
  const isSavingName = updateDatabaseMutation.isPending;

  const isTestingConnection = testDatabaseMutation.isPending;
  const isCopying = copyDatabaseMutation.isPending;
  const [isShowRemoveConfirm, setIsShowRemoveConfirm] = useState(false);
  const isRemoving = deleteDatabaseMutation.isPending;
  const [isShowCopyConfirm, setIsShowCopyConfirm] = useState(false);
  const [isShowTransferDialog, setIsShowTransferDialog] = useState(false);
  const {
    backupConfig,
    backupConfigRevision,
    backupConfigError,
    reloadBackupConfig,
    acknowledgeBackupConfig,
  } = useBackupConfigLoad({
    databaseId: database.id,
    workspaceId: database.workspaceId,
  });
  const [actionError, setActionError] = useState<unknown>();
  const [connectionFailureReason, setConnectionFailureReason] = useState<string>();

  const clearActionFailure = () => {
    setActionError(undefined);
    setConnectionFailureReason(undefined);
  };

  const copyDatabase = () => {
    clearActionFailure();
    setIsShowCopyConfirm(false);

    copyDatabaseMutation
      .mutateAsync([database.id])
      .then((copiedDatabase) => {
        if (isActive()) {
          notify({
            kind: 'success',
            title: t('databases.config.copied.title'),
            description: t('databases.config.copied.description', { name: copiedDatabase.name }),
            lifetimeMs: 4500,
            position: 'top-right',
          });
        }

        onDatabaseCopied(copiedDatabase, database.id);
      })
      .catch((e: unknown) => {
        if (isActive()) setActionError(e);
      })
      .finally(() => {
        if (isActive()) copyDatabaseMutation.reset();
      });
  };

  const testConnection = () => {
    clearActionFailure();
    testDatabaseMutation
      .mutateAsync([database.id])
      .then((testResult) => {
        if (!isActive()) return;

        if (!testResult.isConnected) {
          setConnectionFailureReason(testResult.error ?? '');
          return;
        }

        notify({
          kind: 'success',
          title: t('databases.config.connectionTestSucceeded.title'),
          description: t('databases.config.connectionTestSucceeded.description'),
          lifetimeMs: 4500,
          position: 'top-right',
        });
      })
      .catch((e: unknown) => {
        if (isActive()) setActionError(e);
      })
      .finally(() => {
        if (isActive()) testDatabaseMutation.reset();
      });
  };

  const remove = () => {
    clearActionFailure();
    setIsShowRemoveConfirm(false);
    deleteDatabaseMutation
      .mutateAsync([database.id])
      .then(() => {
        onDatabaseDeleted(database.id);
      })
      .catch((e: unknown) => {
        if (isActive()) setActionError(e);
      })
      .finally(() => {
        if (isActive()) deleteDatabaseMutation.reset();
      });
  };

  const startEdit = (section: DatabaseSettingsSectionId) => {
    clearActionFailure();
    setNameDraft(database.name);
    setEditingSection(section);
  };

  const closeSection = () => setEditingSection(undefined);

  const acceptSavedConnection = (savedDatabase: Database) => {
    if (isActive()) clearActionFailure();

    onDatabaseChanged(savedDatabase);
  };

  const acceptSavedDatabase = (savedDatabase: Database) => {
    if (isActive()) {
      clearActionFailure();
      closeSection();
    }

    onDatabaseChanged(savedDatabase);
  };

  const acceptSavedBackupConfig = (savedConfig: BackupConfig) => {
    clearActionFailure();
    closeSection();
    acknowledgeBackupConfig(savedConfig);
    onDatabaseChanged(database);
  };

  const saveName = () => {
    clearActionFailure();
    updateDatabaseMutation
      .mutateAsync([database.id, { ...database, name: nameDraft }])
      .then(acceptSavedDatabase)
      .catch((e: unknown) => {
        if (isActive()) setActionError(e);
      })
      .finally(() => {
        if (isActive()) updateDatabaseMutation.reset();
      });
  };

  const { cleanup, cleanupError, retryCleanup } = useBackupCleanup({
    databaseId: database.id,
    workspaceId: database.workspaceId,
    initial: backupConfig?.cleanup,
    revision: backupConfigRevision,
  });

  const isPhysicalDatabase = database.settings.type === DatabaseType.POSTGRES_PHYSICAL;
  const isLogicalDatabase = database.settings.type === DatabaseType.POSTGRES_LOGICAL;
  const isSectionEditable = isCanManageDBs && editingSection === undefined;
  const isNameUnsaved = nameDraft !== database.name;
  const physicalBackupConfig =
    isPhysicalDatabase && backupConfig ? toPhysicalBackupConfigInput(backupConfig) : undefined;
  const logicalBackupConfig =
    !isPhysicalDatabase && backupConfig ? toLogicalBackupConfigInput(backupConfig) : undefined;
  const actionFailureMessage =
    connectionFailureReason !== undefined
      ? t('databases.edit.connectionTestFailed', { error: connectionFailureReason })
      : actionError !== undefined
        ? translateApiError(actionError, t)
        : undefined;
  const actionFailureAlert = actionFailureMessage !== undefined && (
    <Alert
      className="mt-4"
      color="danger"
      icon={<CircleAlert aria-hidden size={16} className="inline-block shrink-0 align-middle" />}
      title={actionFailureMessage}
      withCloseButton
      closeButtonLabel={t('common.actions.close')}
      onClose={clearActionFailure}
    />
  );

  return (
    <div className="relative w-full rounded-md bg-surface p-3 shadow sm:p-5">
      {isRemoving && (
        <div className="absolute inset-0 z-10 flex items-center justify-center rounded-tr-md rounded-br-md rounded-bl-md bg-surface/80">
          <div className="flex flex-col items-center gap-3">
            <div className="h-8 w-8 animate-spin rounded-full border-4 border-border border-t-accent" />
            <span className="text-sm font-medium text-muted">{t('databases.config.removing')}</span>
          </div>
        </div>
      )}

      {editingSection !== 'name' ? (
        <div className="mb-5 flex items-center text-xl font-bold sm:text-2xl">
          <span id={databaseNameId}>{database.name}</span>

          {isSectionEditable && (
            <EditIconButton
              subjectId={databaseNameId}
              className="ml-2 !h-[18px] !w-[18px]"
              onClick={() => startEdit('name')}
            />
          )}
        </div>
      ) : (
        <div>
          <div className="flex items-center">
            <TextInput
              className="max-w-full sm:max-w-[250px]"
              aria-label={t('common.fields.name')}
              value={nameDraft}
              onChange={(e) => setNameDraft(e.target.value)}
              placeholder={t('databases.config.namePlaceholder')}
              size="md"
            />

            <div className="ml-1 flex flex-shrink-0 items-center">
              <Button
                variant="subtle"
                className="flex h-6 w-6 items-center justify-center p-0"
                aria-label={t('common.actions.close')}
                onClick={closeSection}
                disabled={isSavingName}
              >
                <X className="inline-block shrink-0 align-middle text-muted" size="1em" />
              </Button>
            </div>
          </div>

          {isNameUnsaved && (
            <Button
              className="mt-1"
              variant="filled"
              onClick={saveName}
              loading={isSavingName}
              disabled={!nameDraft}
              aria-busy={isSavingName}
            >
              {t('common.actions.save')}
            </Button>
          )}

          {actionFailureAlert}
        </div>
      )}

      {backupConfigError !== undefined && editingSection !== 'backup-config' && (
        <LoadFailure className="mb-4" error={backupConfigError} onRetry={reloadBackupConfig} />
      )}

      {database.lastBackupErrorMessage && (
        <div className="mb-4 max-w-full rounded border border-danger px-3 py-3 sm:max-w-[400px]">
          <div className="mt-1 flex items-center text-sm font-bold text-danger">
            <Info className="mr-2 inline-block shrink-0 align-middle" size="1em" />
            {t('databases.config.lastBackupError.title')}
          </div>

          <div className="mt-3 text-sm break-words whitespace-pre-wrap">
            {t('databases.config.lastBackupError.errorLabel')}
            <br />
            {database.lastBackupErrorMessage}
          </div>

          <div className="mt-3 text-sm text-muted">
            {t('databases.config.lastBackupError.clearedByNextBackup')}
          </div>
        </div>
      )}

      <div className="flex flex-col gap-6 lg:flex-row lg:flex-wrap lg:gap-10">
        <DatabaseSettingsSection
          title={t('databases.config.sections.database')}
          isEditable={isSectionEditable}
          onEdit={() => startEdit('connection')}
        >
          {editingSection === 'connection' ? (
            <EditDatabaseConnectionSection
              key={database.id}
              database={database}
              onDatabaseSaved={acceptSavedConnection}
              onClose={closeSection}
            />
          ) : (
            <ShowDatabaseSpecificDataComponent database={database} />
          )}
        </DatabaseSettingsSection>

        <DatabaseSettingsSection
          title={t('databases.config.sections.backupConfig')}
          isEditable={isSectionEditable}
          onEdit={() => startEdit('backup-config')}
        >
          {editingSection === 'backup-config' ? (
            isPhysicalDatabase ? (
              <EditPhysicalBackupConfigSection
                key={database.id}
                database={database}
                onCancel={closeSection}
                onSaved={acceptSavedBackupConfig}
                storageCleanupNotice={
                  <StorageCleanupNotice
                    cleanup={cleanup}
                    error={cleanupError}
                    onRetry={retryCleanup}
                  />
                }
              />
            ) : (
              <EditLogicalBackupConfigSection
                key={database.id}
                database={database}
                onCancel={closeSection}
                onSaved={acceptSavedBackupConfig}
                storageCleanupNotice={
                  <StorageCleanupNotice
                    cleanup={cleanup}
                    error={cleanupError}
                    onRetry={retryCleanup}
                  />
                }
              />
            )
          ) : physicalBackupConfig ? (
            <ShowPhysicalBackupConfigComponent
              database={database}
              backupConfig={physicalBackupConfig}
              cleanup={cleanup}
              cleanupError={cleanupError}
              onRetryCleanup={retryCleanup}
            />
          ) : logicalBackupConfig ? (
            <ShowLogicalBackupConfigComponent
              database={database}
              backupConfig={logicalBackupConfig}
              cleanup={cleanup}
              cleanupError={cleanupError}
              onRetryCleanup={retryCleanup}
            />
          ) : (
            backupConfigError === undefined && <Loader role="status" />
          )}
        </DatabaseSettingsSection>
      </div>

      <div className="flex flex-col gap-6 lg:flex-row lg:flex-wrap lg:gap-10">
        <DatabaseSettingsSection
          title={t('databases.config.sections.healthcheck')}
          isEditable={isSectionEditable}
          onEdit={() => startEdit('healthcheck')}
        >
          {editingSection === 'healthcheck' ? (
            <EditHealthcheckConfigComponent databaseId={database.id} onClose={closeSection} />
          ) : (
            <ShowHealthcheckConfigComponent databaseId={database.id} />
          )}
        </DatabaseSettingsSection>

        {isLogicalDatabase && (
          <DatabaseSettingsSection
            title={t('databases.config.sections.restoreVerification')}
            isEditable={isSectionEditable}
            onEdit={() => startEdit('verification')}
          >
            {editingSection === 'verification' ? (
              <EditVerificationConfigComponent databaseId={database.id} onClose={closeSection} />
            ) : (
              <ShowVerificationConfigComponent databaseId={database.id} />
            )}
          </DatabaseSettingsSection>
        )}

        <DatabaseSettingsSection
          title={t('databases.config.sections.notifiers')}
          isEditable={isSectionEditable}
          onEdit={() => startEdit('notifiers')}
        >
          {editingSection === 'notifiers' ? (
            <EditDatabaseNotifiersSection
              key={database.id}
              database={database}
              onCancel={closeSection}
              onSaved={acceptSavedDatabase}
            />
          ) : (
            <ShowDatabaseNotifiersComponent database={database} />
          )}
        </DatabaseSettingsSection>
      </div>

      {editingSection !== 'name' && actionFailureAlert}

      {editingSection !== 'connection' && (
        <div className="mt-10 flex flex-col gap-2 sm:flex-row sm:gap-0">
          <Button
            variant="filled"
            className="w-full sm:mr-1 sm:w-auto"
            onClick={testConnection}
            loading={isTestingConnection}
            disabled={isTestingConnection}
            aria-busy={isTestingConnection}
          >
            {t('databases.actions.testConnection')}
          </Button>

          {isCanManageDBs && (
            <>
              <ActionIcon
                variant="outline"
                color="accent"
                aria-label={t('common.actions.transfer')}
                onClick={() => setIsShowTransferDialog(true)}
                className="sm:mr-1"
              >
                <ArrowRight size="1em" className="inline-block shrink-0 align-middle" />
              </ActionIcon>

              <ActionIcon
                variant="outline"
                color="accent"
                aria-label={t('common.actions.copy')}
                onClick={() => setIsShowCopyConfirm(true)}
                loading={isCopying}
                disabled={isCopying}
                className="sm:mr-1"
                aria-busy={isCopying}
              >
                <Copy size="1em" className="inline-block shrink-0 align-middle" />
              </ActionIcon>

              <ActionIcon
                variant="outline"
                color="danger"
                aria-label={t('common.actions.delete')}
                onClick={() => setIsShowRemoveConfirm(true)}
                loading={isRemoving}
                disabled={isRemoving}
                aria-busy={isRemoving}
              >
                <Trash2 size="1em" className="inline-block shrink-0 align-middle" />
              </ActionIcon>
            </>
          )}
        </div>
      )}

      {isShowCopyConfirm && (
        <ConfirmationComponent
          onConfirm={copyDatabase}
          onDecline={() => setIsShowCopyConfirm(false)}
          description={t('databases.config.copyConfirmation')}
          actionText={t('common.actions.copy')}
          isDestructive={false}
        />
      )}

      {isShowRemoveConfirm && (
        <ConfirmationComponent
          onConfirm={remove}
          onDecline={() => setIsShowRemoveConfirm(false)}
          description={t('databases.config.removeConfirmation')}
          actionText={t('common.actions.remove')}
          isDestructive
        />
      )}

      {isShowTransferDialog && (
        <DatabaseTransferDialogComponent
          database={database}
          currentStorageId={backupConfig?.storageId ?? undefined}
          onClose={() => setIsShowTransferDialog(false)}
          onTransferred={() => {
            if (isActive()) setIsShowTransferDialog(false);
            onDatabaseTransferred(database.id);
          }}
        />
      )}
    </div>
  );
};
