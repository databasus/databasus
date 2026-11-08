import { Alert, Button, Loader, Switch } from '@mantine/core';
import { CircleAlert } from 'lucide-react';
import { type ReactNode, useEffect, useEffectEvent, useState } from 'react';
import { useTranslation } from 'react-i18next';

import {
  type BackupConfig,
  type BackupConfigInput,
  backupConfigApi,
} from '@/entities/backup-configs';
import type { Database } from '@/entities/databases';
import { readConcurrentRequests, useSessionMutation, useSessionQuery } from '@/shared/api';
import { useSessionGuard } from '@/shared/auth';
import { translateApiError } from '@/shared/i18n';
import { ConfirmationComponent, LoadFailure } from '@/shared/ui';

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
  const updateBackupConfigMutation = useSessionMutation({
    mutationFn: (parameters: Parameters<typeof backupConfigApi.updateBackupConfig>) =>
      backupConfigApi.updateBackupConfig(...parameters),
  });

  const { t } = useTranslation();
  const isActive = useSessionGuard();

  const [acknowledgedDraft, setAcknowledgedDraft] = useState<TDraft>();
  const [draft, setDraft] = useState<TDraft>();
  const query = useSessionQuery({
    queryKey: ['backup-config-editor', database.workspaceId, database.id, isPhysical],
    queryFn: (signal) =>
      readConcurrentRequests(signal, async (signal) => {
        const [config, hasHistory] = await Promise.all([
          backupConfigApi.getBackupConfig(database.id, { signal }),
          backupConfigApi
            .hasBackupHistory(database.id, isPhysical, { signal })
            .catch(() => undefined),
        ]);

        return { config, hasHistory };
      }),
  });
  const hasHistory = query.data?.hasHistory;
  const loadError = query.error ?? undefined;

  const isSaving = updateBackupConfigMutation.isPending;
  const [saveError, setSaveError] = useState<unknown>();
  const [isShowStorageSwitchWarning, setIsShowStorageSwitchWarning] = useState(false);

  const saveDraft = async (savedDraft: TDraft) => {
    setSaveError(undefined);

    try {
      const savedConfig = await updateBackupConfigMutation.mutateAsync([
        database.id,
        toInput(savedDraft),
      ]);
      if (!isActive()) return;

      onSaved(savedConfig);
    } catch (error) {
      if (!isActive()) return;

      setSaveError(error);
      updateBackupConfigMutation.reset();
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

  const buildLoadedDraft = useEffectEvent((loadedConfig: BackupConfig) => toDraft(loadedConfig));
  useEffect(() => {
    const loadedDraft = query.data ? buildLoadedDraft(query.data.config) : undefined;
    setAcknowledgedDraft(loadedDraft);
    setDraft(loadedDraft);
  }, [query.data]);

  if (loadError !== undefined) {
    return (
      <div>
        <LoadFailure error={loadError} onRetry={() => void query.refetch()} />

        <div className="mt-5 flex">
          <Button variant="outline" color="danger" onClick={onCancel}>
            {t('common.actions.cancel')}
          </Button>
        </div>
      </div>
    );
  }

  if (!draft || !acknowledgedDraft) {
    return (
      <div className="mb-5 flex items-center">
        <Loader role="status" />
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
          onChange={(event) =>
            setDraft({ ...draft, isBackupsEnabled: event.currentTarget.checked })
          }
          size="xs"
        />
      </div>

      {renderFields(draft, setDraft)}

      {saveError !== undefined && (
        <Alert
          className="mt-5"
          color="danger"
          icon={
            <CircleAlert aria-hidden size={16} className="inline-block shrink-0 align-middle" />
          }
          title={translateApiError(saveError, t)}
        />
      )}

      <div className="mt-5 flex">
        <Button
          variant="outline"
          color="danger"
          className="mr-1"
          disabled={isSaving}
          onClick={onCancel}
        >
          {t('common.actions.cancel')}
        </Button>

        <Button
          variant="filled"
          className="mr-5 ml-1"
          onClick={() => confirmAndSaveDraft(draft)}
          loading={isSaving}
          disabled={!canSave}
          aria-busy={isSaving}
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
