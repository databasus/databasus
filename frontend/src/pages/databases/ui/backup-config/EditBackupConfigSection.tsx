import { Alert, Button, Spin, Switch } from 'antd';
import { type ReactNode, useEffect, useEffectEvent, useState } from 'react';
import { useTranslation } from 'react-i18next';

import {
  type BackupConfig,
  type BackupConfigInput,
  backupConfigApi,
} from '../../../../entities/backup-configs';
import type { Database } from '../../../../entities/databases';
import { useSessionGuard } from '../../../../shared/auth';
import { translateApiError } from '../../../../shared/i18n';
import { ConfirmationComponent, LoadFailure } from '../../../../shared/ui';

interface BackupConfigSectionDraft {
  isBackupsEnabled: boolean;
  storageId: string | null;
}

interface Props<TDraft extends BackupConfigSectionDraft> {
  database: Database;
  isPhysical: boolean;
  toDraft: (config: BackupConfig) => TDraft | undefined;
  toInput: (draft: TDraft) => BackupConfigInput;
  isChanged: (comparison: { acknowledged: TDraft; shown: TDraft }) => boolean;
  isComplete: (draft: TDraft) => boolean;
  renderFields: (draft: TDraft, onChange: (draft: TDraft) => void) => ReactNode;
  onCancel: () => void;
  onSaved: (config: BackupConfig) => void;
}

export const EditBackupConfigSection = <TDraft extends BackupConfigSectionDraft>({
  database,
  isPhysical,
  toDraft,
  toInput,
  isChanged,
  isComplete,
  renderFields,
  onCancel,
  onSaved,
}: Props<TDraft>) => {
  const { t } = useTranslation();
  const isActive = useSessionGuard();

  const [acknowledgedDraft, setAcknowledgedDraft] = useState<TDraft>();
  const [draft, setDraft] = useState<TDraft>();
  const [hasHistory, setHasHistory] = useState<boolean | undefined>();
  const [loadError, setLoadError] = useState<unknown>();
  const [loadAttempt, setLoadAttempt] = useState(0);

  const [isSaving, setIsSaving] = useState(false);
  const [saveError, setSaveError] = useState<unknown>();
  const [isShowStorageSwitchWarning, setIsShowStorageSwitchWarning] = useState(false);

  const saveDraft = async (savedDraft: TDraft) => {
    setIsSaving(true);
    setSaveError(undefined);

    try {
      const savedConfig = await backupConfigApi.updateBackupConfig(
        database.id,
        toInput(savedDraft),
      );
      if (!isActive()) return;

      onSaved(savedConfig);
    } catch (error) {
      if (!isActive()) return;

      setSaveError(error);
      setIsSaving(false);
    }
  };

  const confirmAndSaveDraft = (savedDraft: TDraft) => {
    const savedStorageId = acknowledgedDraft?.storageId ?? null;
    const isStorageSwitched = savedDraft.storageId && savedDraft.storageId !== savedStorageId;

    if (isStorageSwitched && (savedStorageId !== null || hasHistory !== false)) {
      setIsShowStorageSwitchWarning(true);
      return;
    }

    void saveDraft(savedDraft);
  };

  const loadHasHistory = useEffectEvent(() =>
    backupConfigApi.hasBackupHistory(database.id, isPhysical),
  );
  const buildLoadedDraft = useEffectEvent((loadedConfig: BackupConfig) => toDraft(loadedConfig));

  useEffect(() => {
    let isCurrentLoad = true;

    setLoadError(undefined);
    setAcknowledgedDraft(undefined);
    setDraft(undefined);

    Promise.all([
      backupConfigApi.getBackupConfig(database.id),
      loadHasHistory().catch(() => undefined),
    ])
      .then(([loadedConfig, loadedHasHistory]) => {
        if (!isCurrentLoad) return;

        const loadedDraft = buildLoadedDraft(loadedConfig);

        setAcknowledgedDraft(loadedDraft);
        setDraft(loadedDraft);
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
        <LoadFailure error={loadError} onRetry={() => setLoadAttempt((attempt) => attempt + 1)} />

        <div className="mt-5 flex">
          <Button danger ghost onClick={onCancel}>
            {t('common.actions.cancel')}
          </Button>
        </div>
      </div>
    );
  }

  if (!draft || !acknowledgedDraft) {
    return (
      <div className="mb-5 flex items-center">
        <Spin />
      </div>
    );
  }

  const canSave = isChanged({ acknowledged: acknowledgedDraft, shown: draft }) && isComplete(draft);

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

      {renderFields(draft, setDraft)}

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
