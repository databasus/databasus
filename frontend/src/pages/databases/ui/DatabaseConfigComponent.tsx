import {
  ArrowRightOutlined,
  CloseOutlined,
  CopyOutlined,
  DeleteOutlined,
  InfoCircleOutlined,
} from '@ant-design/icons';
import { Alert, App, Button, Input, Spin } from 'antd';
import { useId, useState } from 'react';
import { useTranslation } from 'react-i18next';

import {
  type BackupConfig,
  toLogicalBackupConfigInput,
  toPhysicalBackupConfigInput,
} from '@/entities/backup-configs';
import { type Database, DatabaseType, databaseApi } from '@/entities/databases';
import { EditLogicalBackupConfigSection } from '@/pages/databases/ui/backup-config/EditLogicalBackupConfigSection';
import { EditPhysicalBackupConfigSection } from '@/pages/databases/ui/backup-config/EditPhysicalBackupConfigSection';
import { ShowLogicalBackupConfigComponent } from '@/pages/databases/ui/backup-config/ShowLogicalBackupConfigComponent';
import { ShowPhysicalBackupConfigComponent } from '@/pages/databases/ui/backup-config/ShowPhysicalBackupConfigComponent';
import { StorageCleanupNotice } from '@/pages/databases/ui/backup-config/StorageCleanupNotice';
import { useBackupCleanup } from '@/pages/databases/ui/backup-config/useBackupCleanup';
import { useBackupConfigLoad } from '@/pages/databases/ui/backup-config/useBackupConfigLoad';
import { DatabaseSettingsSection } from '@/pages/databases/ui/DatabaseSettingsSection';
import { DatabaseTransferDialogComponent } from '@/pages/databases/ui/DatabaseTransferDialogComponent';
import { EditDatabaseConnectionSection } from '@/pages/databases/ui/EditDatabaseConnectionSection';
import { EditDatabaseNotifiersSection } from '@/pages/databases/ui/EditDatabaseNotifiersSection';
import { EditIconButton } from '@/pages/databases/ui/EditIconButton';
import {
  EditHealthcheckConfigComponent,
  ShowHealthcheckConfigComponent,
} from '@/pages/databases/ui/healthcheck';
import { ShowDatabaseNotifiersComponent } from '@/pages/databases/ui/show/ShowDatabaseNotifiersComponent';
import { ShowDatabaseSpecificDataComponent } from '@/pages/databases/ui/show/ShowDatabaseSpecificDataComponent';
import { EditVerificationConfigComponent } from '@/pages/databases/ui/verification/EditVerificationConfigComponent';
import { ShowVerificationConfigComponent } from '@/pages/databases/ui/verification/ShowVerificationConfigComponent';
import { useSessionGuard } from '@/shared/auth';
import { translateApiError } from '@/shared/i18n';
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
  const { t } = useTranslation();
  const { notification } = App.useApp();
  const isActive = useSessionGuard();
  const databaseNameId = useId();

  const [editingSection, setEditingSection] = useState<DatabaseSettingsSectionId>();
  const [nameDraft, setNameDraft] = useState('');
  const [isSavingName, setIsSavingName] = useState(false);

  const [isTestingConnection, setIsTestingConnection] = useState(false);
  const [isCopying, setIsCopying] = useState(false);
  const [isShowRemoveConfirm, setIsShowRemoveConfirm] = useState(false);
  const [isRemoving, setIsRemoving] = useState(false);
  const [isShowCopyConfirm, setIsShowCopyConfirm] = useState(false);
  const [isShowTransferDialog, setIsShowTransferDialog] = useState(false);
  const { backupConfig, backupConfigError, reloadBackupConfig, acknowledgeBackupConfig } =
    useBackupConfigLoad({
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
    setIsCopying(true);
    setIsShowCopyConfirm(false);

    databaseApi
      .copyDatabase(database.id)
      .then((copiedDatabase) => {
        if (isActive()) {
          notification.success({
            message: t('databases.config.copied.title'),
            description: t('databases.config.copied.description', { name: copiedDatabase.name }),
          });
        }

        onDatabaseCopied(copiedDatabase, database.id);
      })
      .catch((e: unknown) => {
        if (isActive()) setActionError(e);
      })
      .finally(() => {
        if (isActive()) setIsCopying(false);
      });
  };

  const testConnection = () => {
    clearActionFailure();
    setIsTestingConnection(true);
    databaseApi
      .testDatabase(database.id)
      .then((testResult) => {
        if (!isActive()) return;

        if (!testResult.isConnected) {
          setConnectionFailureReason(testResult.error ?? '');
          return;
        }

        notification.success({
          message: t('databases.config.connectionTestSucceeded.title'),
          description: t('databases.config.connectionTestSucceeded.description'),
        });
      })
      .catch((e: unknown) => {
        if (isActive()) setActionError(e);
      })
      .finally(() => {
        if (isActive()) setIsTestingConnection(false);
      });
  };

  const remove = () => {
    clearActionFailure();
    setIsShowRemoveConfirm(false);
    setIsRemoving(true);
    databaseApi
      .deleteDatabase(database.id)
      .then(() => {
        onDatabaseDeleted(database.id);
      })
      .catch((e: unknown) => {
        if (isActive()) setActionError(e);
      })
      .finally(() => {
        if (isActive()) setIsRemoving(false);
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
    setIsSavingName(true);
    databaseApi
      .updateDatabase(database.id, { ...database, name: nameDraft })
      .then(acceptSavedDatabase)
      .catch((e: unknown) => {
        if (isActive()) setActionError(e);
      })
      .finally(() => {
        if (isActive()) setIsSavingName(false);
      });
  };

  const { cleanup, cleanupError, retryCleanup } = useBackupCleanup({
    databaseId: database.id,
    workspaceId: database.workspaceId,
    initial: backupConfig?.cleanup,
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
      type="error"
      showIcon
      message={actionFailureMessage}
      closable={{ closeIcon: <CloseOutlined />, 'aria-label': t('common.actions.close') }}
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
            <Input
              className="max-w-full sm:max-w-[250px]"
              aria-label={t('common.fields.name')}
              value={nameDraft}
              onChange={(e) => setNameDraft(e.target.value)}
              placeholder={t('databases.config.namePlaceholder')}
              size="large"
            />

            <div className="ml-1 flex flex-shrink-0 items-center">
              <Button
                type="text"
                className="flex h-6 w-6 items-center justify-center p-0"
                aria-label={t('common.actions.close')}
                onClick={closeSection}
                disabled={isSavingName}
              >
                <CloseOutlined className="text-muted" />
              </Button>
            </div>
          </div>

          {isNameUnsaved && (
            <Button
              className="mt-1"
              type="primary"
              onClick={saveName}
              loading={isSavingName}
              disabled={!nameDraft}
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
            <InfoCircleOutlined className="mr-2" />
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
            backupConfigError === undefined && <Spin />
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
            type="primary"
            className="w-full sm:mr-1 sm:w-auto"
            onClick={testConnection}
            loading={isTestingConnection}
            disabled={isTestingConnection}
          >
            {t('databases.actions.testConnection')}
          </Button>

          {isCanManageDBs && (
            <>
              <Button
                type="primary"
                ghost
                icon={<ArrowRightOutlined />}
                aria-label={t('common.actions.transfer')}
                onClick={() => setIsShowTransferDialog(true)}
                className="sm:mr-1"
              />

              <Button
                type="primary"
                ghost
                icon={<CopyOutlined />}
                aria-label={t('common.actions.copy')}
                onClick={() => setIsShowCopyConfirm(true)}
                loading={isCopying}
                disabled={isCopying}
                className="sm:mr-1"
              />

              <Button
                type="primary"
                ghost
                danger
                icon={<DeleteOutlined />}
                aria-label={t('common.actions.delete')}
                onClick={() => setIsShowRemoveConfirm(true)}
                loading={isRemoving}
                disabled={isRemoving}
              />
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
            setIsShowTransferDialog(false);
            onDatabaseTransferred(database.id);
          }}
        />
      )}
    </div>
  );
};
