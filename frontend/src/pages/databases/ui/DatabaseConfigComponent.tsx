import {
  ArrowRightOutlined,
  CloseOutlined,
  CopyOutlined,
  DeleteOutlined,
  InfoCircleOutlined,
} from '@ant-design/icons';
import { Alert, Button, Input } from 'antd';
import { useEffect, useState } from 'react';
import { useTranslation } from 'react-i18next';

import {
  EditPhysicalBackupConfigComponent,
  ShowPhysicalBackupConfigComponent,
} from '@/features/backups/physical';
import {
  EditVerificationConfigComponent,
  ShowVerificationConfigComponent,
} from '@/features/verification/config';

import { backupConfigApi } from '../../../entities/backup-configs';
import { type Database, DatabaseType, databaseApi } from '../../../entities/databases';
import { translateApiError } from '../../../shared/i18n';
import { ToastHelper } from '../../../shared/toast';
import { ConfirmationComponent } from '../../../shared/ui';
import { EditLogicalBackupConfigComponent } from './backup-config/EditLogicalBackupConfigComponent';
import { ShowLogicalBackupConfigComponent } from './backup-config/ShowLogicalBackupConfigComponent';
import { BackupStorageSelection } from './BackupStorageSelection';
import { DatabaseSettingsSection } from './DatabaseSettingsSection';
import { DatabaseTransferDialogComponent } from './DatabaseTransferDialogComponent';
import { EditDatabaseConnectionSection } from './EditDatabaseConnectionSection';
import { EditDatabaseNotifiersComponent } from './EditDatabaseNotifiersComponent';
import { EditHealthcheckConfigComponent, ShowHealthcheckConfigComponent } from './healthcheck';
import { ShowDatabaseNotifiersComponent } from './show/ShowDatabaseNotifiersComponent';
import { ShowDatabaseSpecificDataComponent } from './show/ShowDatabaseSpecificDataComponent';

interface Props {
  database: Database;
  onDatabaseChanged: (database: Database) => void;
  onDatabaseDeleted: () => void;
  onDatabaseCopied: (copiedDatabase: Database) => void;
  onDatabaseTransferred: () => void;

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
  const [editingSection, setEditingSection] = useState<DatabaseSettingsSectionId>();
  const [nameDraft, setNameDraft] = useState('');
  const [isSavingName, setIsSavingName] = useState(false);

  const [isTestingConnection, setIsTestingConnection] = useState(false);
  const [isCopying, setIsCopying] = useState(false);
  const [isShowRemoveConfirm, setIsShowRemoveConfirm] = useState(false);
  const [isRemoving, setIsRemoving] = useState(false);
  const [isShowCopyConfirm, setIsShowCopyConfirm] = useState(false);
  const [isShowTransferDialog, setIsShowTransferDialog] = useState(false);
  const [currentStorageId, setCurrentStorageId] = useState<string | undefined>();
  const [backupConfigError, setBackupConfigError] = useState<unknown>();
  const [backupConfigLoadAttempt, setBackupConfigLoadAttempt] = useState(0);

  const copyDatabase = () => {
    setIsCopying(true);
    setIsShowCopyConfirm(false);

    databaseApi
      .copyDatabase(database.id)
      .then((copiedDatabase) => {
        ToastHelper.showToast({
          title: t('databases.config.copied.title'),
          description: t('databases.config.copied.description', { name: copiedDatabase.name }),
        });
        onDatabaseCopied(copiedDatabase);
      })
      .catch((e: unknown) => {
        alert(translateApiError(e, t));
      })
      .finally(() => {
        setIsCopying(false);
      });
  };

  const testConnection = () => {
    setIsTestingConnection(true);
    databaseApi
      .testDatabase(database.id)
      .then((testResult) => {
        if (!testResult.isConnected) {
          alert(t('databases.edit.connectionTestFailed', { error: testResult.error }));
          return;
        }

        ToastHelper.showToast({
          title: t('databases.config.connectionTestSucceeded.title'),
          description: t('databases.config.connectionTestSucceeded.description'),
        });
      })
      .catch((e: unknown) => {
        alert(translateApiError(e, t));
      })
      .finally(() => {
        setIsTestingConnection(false);
      });
  };

  const remove = () => {
    setIsShowRemoveConfirm(false);
    setIsRemoving(true);
    databaseApi
      .deleteDatabase(database.id)
      .then(() => {
        onDatabaseDeleted();
      })
      .catch((e: unknown) => {
        alert(translateApiError(e, t));
      })
      .finally(() => {
        setIsRemoving(false);
      });
  };

  const startEdit = (section: DatabaseSettingsSectionId) => {
    setNameDraft(database.name);
    setEditingSection(section);
  };

  const closeSection = () => setEditingSection(undefined);

  const acceptSavedDatabase = (savedDatabase: Database) => {
    closeSection();
    onDatabaseChanged(savedDatabase);
  };

  const acceptSavedBackupConfig = () => {
    closeSection();
    setBackupConfigLoadAttempt((attempt) => attempt + 1);
    onDatabaseChanged(database);
  };

  const saveName = () => {
    setIsSavingName(true);
    databaseApi
      .updateDatabase(database.id, { ...database, name: nameDraft })
      .then(acceptSavedDatabase)
      .catch((e: unknown) => {
        alert(translateApiError(e, t));
      })
      .finally(() => {
        setIsSavingName(false);
      });
  };

  useEffect(() => {
    let isActive = true;
    setBackupConfigError(undefined);
    backupConfigApi
      .getBackupConfig(database.id)
      .then((loadedBackupConfig) => {
        if (isActive) setCurrentStorageId(loadedBackupConfig.storageId ?? undefined);
      })
      .catch((error) => {
        if (isActive) setBackupConfigError(error);
      });
    return () => {
      isActive = false;
    };
  }, [database.id, backupConfigLoadAttempt]);

  const isPhysicalDatabase = database.settings.type === DatabaseType.POSTGRES_PHYSICAL;
  const isLogicalDatabase = database.settings.type === DatabaseType.POSTGRES_LOGICAL;
  const isSectionEditable = isCanManageDBs && editingSection === undefined;
  const isNameUnsaved = nameDraft !== database.name;

  return (
    <div className="relative w-full rounded-tr-md rounded-br-md rounded-bl-md bg-surface p-3 shadow sm:p-5">
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
          {database.name}

          {isSectionEditable && (
            <div className="ml-2 cursor-pointer" onClick={() => startEdit('name')}>
              <img src="/icons/pen-gray.svg" />
            </div>
          )}
        </div>
      ) : (
        <div>
          <div className="flex items-center">
            <Input
              className="max-w-full sm:max-w-[250px]"
              value={nameDraft}
              onChange={(e) => setNameDraft(e.target.value)}
              placeholder={t('databases.config.namePlaceholder')}
              size="large"
            />

            <div className="ml-1 flex flex-shrink-0 items-center">
              <Button
                type="text"
                className="flex h-6 w-6 items-center justify-center p-0"
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
        </div>
      )}

      {backupConfigError !== undefined && (
        <Alert
          className="mb-4"
          type="error"
          showIcon
          message={translateApiError(backupConfigError, t)}
          action={
            <Button onClick={() => setBackupConfigLoadAttempt((attempt) => attempt + 1)}>
              {t('common.actions.retry')}
            </Button>
          }
        />
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
              onDatabaseSaved={onDatabaseChanged}
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
              <EditPhysicalBackupConfigComponent
                renderStorageSelection={(options) => <BackupStorageSelection {...options} />}
                database={database}
                isShowCancelButton
                onCancel={closeSection}
                isSaveToApi={true}
                onSaved={acceptSavedBackupConfig}
                isShowBackButton={false}
                onBack={() => {}}
              />
            ) : (
              <EditLogicalBackupConfigComponent
                database={database}
                isShowCancelButton
                onCancel={closeSection}
                isSaveToApi={true}
                onSaved={acceptSavedBackupConfig}
                isShowBackButton={false}
                onBack={() => {}}
              />
            )
          ) : isPhysicalDatabase ? (
            <ShowPhysicalBackupConfigComponent database={database} />
          ) : (
            <ShowLogicalBackupConfigComponent database={database} />
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
            <EditDatabaseNotifiersComponent
              workspaceId={database.workspaceId}
              database={database}
              isShowCancelButton
              isShowBackButton={false}
              isShowSaveOnlyForUnsaved={true}
              onBack={() => {}}
              onCancel={closeSection}
              isSaveToApi={true}
              saveButtonText={t('common.actions.save')}
              onSaved={acceptSavedDatabase}
            />
          ) : (
            <ShowDatabaseNotifiersComponent database={database} />
          )}
        </DatabaseSettingsSection>
      </div>

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
                onClick={() => setIsShowTransferDialog(true)}
                className="sm:mr-1"
              />

              <Button
                type="primary"
                ghost
                icon={<CopyOutlined />}
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
          currentStorageId={currentStorageId}
          onClose={() => setIsShowTransferDialog(false)}
          onTransferred={() => {
            setIsShowTransferDialog(false);
            onDatabaseTransferred();
          }}
        />
      )}
    </div>
  );
};
