import { Alert, Button, Spin, Switch } from 'antd';
import { useEffect, useState } from 'react';
import { useTranslation } from 'react-i18next';

import {
  type LogicalBackupConfigInput,
  StorageSwitchHistoryComponent,
  backupConfigApi,
  toLogicalBackupConfigInput,
} from '../../../../entities/backup-configs';
import type { Database } from '../../../../entities/databases';
import { useSessionGuard } from '../../../../shared/auth';
import { translateApiError } from '../../../../shared/i18n';
import { ConfirmationComponent } from '../../../../shared/ui';
import { isLogicalBackupConfigChanged } from '../../model/isLogicalBackupConfigChanged';
import { isLogicalBackupConfigComplete } from '../../model/isLogicalBackupConfigComplete';
import { LogicalBackupConfigFields } from './LogicalBackupConfigFields';

interface Props {
  database: Database;
  onCancel: () => void;
  onSaved: () => void;
}

export const EditLogicalBackupConfigSection = ({ database, onCancel, onSaved }: Props) => {
  const { t } = useTranslation();
  const isActive = useSessionGuard();

  const [acknowledgedConfig, setAcknowledgedConfig] = useState<LogicalBackupConfigInput>();
  const [draft, setDraft] = useState<LogicalBackupConfigInput>();
  const [hasHistory, setHasHistory] = useState<boolean | undefined>();
  const [loadError, setLoadError] = useState<unknown>();
  const [loadAttempt, setLoadAttempt] = useState(0);

  const [isSaving, setIsSaving] = useState(false);
  const [saveError, setSaveError] = useState<unknown>();
  const [isShowStorageSwitchWarning, setIsShowStorageSwitchWarning] = useState(false);

  const saveDraft = async (config: LogicalBackupConfigInput) => {
    setIsSaving(true);
    setSaveError(undefined);

    try {
      await backupConfigApi.updateBackupConfig(database.id, config);
      if (!isActive()) return;

      onSaved();
    } catch (error) {
      if (!isActive()) return;

      setSaveError(error);
      setIsSaving(false);
    }
  };

  const confirmAndSaveDraft = (config: LogicalBackupConfigInput) => {
    const savedStorageId = acknowledgedConfig?.storageId ?? null;
    const isStorageSwitched = config.storageId && config.storageId !== savedStorageId;

    if (isStorageSwitched && (savedStorageId !== null || hasHistory !== false)) {
      setIsShowStorageSwitchWarning(true);
      return;
    }

    void saveDraft(config);
  };

  useEffect(() => {
    let isCurrentLoad = true;

    setLoadError(undefined);
    setAcknowledgedConfig(undefined);
    setDraft(undefined);

    Promise.all([
      backupConfigApi.getBackupConfig(database.id),
      backupConfigApi.hasBackupHistory(database.id, false).catch(() => undefined),
    ])
      .then(([loadedConfig, loadedHasHistory]) => {
        if (!isCurrentLoad) return;

        const loadedLogicalConfig = toLogicalBackupConfigInput(loadedConfig);

        setAcknowledgedConfig(loadedLogicalConfig);
        setDraft(loadedLogicalConfig);
        setHasHistory(loadedHasHistory);
      })
      .catch((error: unknown) => {
        if (isCurrentLoad) setLoadError(error);
      });

    return () => {
      isCurrentLoad = false;
    };
  }, [database.id, loadAttempt]);

  if (loadError !== undefined) {
    return (
      <div>
        <Alert
          type="error"
          showIcon
          message={translateApiError(loadError, t)}
          action={
            <Button onClick={() => setLoadAttempt((attempt) => attempt + 1)}>
              {t('common.actions.retry')}
            </Button>
          }
        />

        <div className="mt-5 flex">
          <Button danger ghost onClick={onCancel}>
            {t('common.actions.cancel')}
          </Button>
        </div>
      </div>
    );
  }

  if (!draft || !acknowledgedConfig) {
    return (
      <div className="mb-5 flex items-center">
        <Spin />
      </div>
    );
  }

  const canSave =
    isLogicalBackupConfigChanged({ acknowledged: acknowledgedConfig, shown: draft }) &&
    isLogicalBackupConfigComplete(draft);

  return (
    <div>
      <div className="mb-1 flex w-full flex-col items-start sm:flex-row sm:items-center">
        <div className="mb-1 min-w-[150px] sm:mb-0 sm:pr-2">
          {t('backups.config.backupsEnabled')}
        </div>
        <Switch
          checked={draft.isBackupsEnabled}
          aria-label={t('backups.config.backupsEnabled')}
          onChange={(isBackupsEnabled) => setDraft({ ...draft, isBackupsEnabled })}
          size="small"
        />
      </div>

      <LogicalBackupConfigFields
        config={draft}
        onChange={setDraft}
        workspaceId={database.workspaceId}
        canClearStorage
      />

      {saveError !== undefined && (
        <Alert className="mt-5" type="error" showIcon message={translateApiError(saveError, t)} />
      )}

      <div className="mt-5 flex">
        <Button danger ghost className="mr-1" disabled={isSaving} onClick={onCancel}>
          {t('common.actions.cancel')}
        </Button>

        <Button
          type="primary"
          className="mr-5 ml-1"
          onClick={() => confirmAndSaveDraft(draft)}
          loading={isSaving}
          disabled={!canSave}
        >
          {t('common.actions.save')}
        </Button>
      </div>

      <StorageSwitchHistoryComponent workspaceId={database.workspaceId} databaseId={database.id} />

      {isShowStorageSwitchWarning && (
        <ConfirmationComponent
          onConfirm={() => {
            setIsShowStorageSwitchWarning(false);
            void saveDraft(draft);
          }}
          onDecline={() => setIsShowStorageSwitchWarning(false)}
          description={t('backups.storageSwitch.warning')}
          isDestructive
          actionText={t('backups.storageSwitch.confirm')}
          cancelText={t('common.actions.cancel')}
        />
      )}
    </div>
  );
};
