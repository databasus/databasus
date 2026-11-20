import { Alert, Button, Loader, NumberInput } from '@mantine/core';
import { CircleAlert } from 'lucide-react';
import { useEffect, useEffectEvent, useId, useRef, useState } from 'react';
import { useTranslation } from 'react-i18next';

import {
  type InstanceBackupDestinationChoice,
  type InstanceBackupDestinationHealth,
  type InstanceBackupSettings,
  instanceBackupApi,
} from '@/entities/instance-backups';
import { ScheduleEditor, isScheduleComplete } from '@/entities/schedules';
import { STORAGE_TYPE_LABEL_KEYS, type Storage } from '@/entities/storages';
import type { Workspace } from '@/entities/workspaces';
import { acknowledgeInstanceBackupSettings } from '@/pages/instance-settings/model/acknowledgeInstanceBackupSettings';
import { isInstanceBackupSettingsChanged } from '@/pages/instance-settings/model/isInstanceBackupSettingsChanged';
import { CreateDestinationDialog } from '@/pages/instance-settings/ui/instance-backups/CreateDestinationDialog';
import { InstanceBackupDestinationList } from '@/pages/instance-settings/ui/instance-backups/InstanceBackupDestinationList';
import { useSessionMutation, useSessionQuery } from '@/shared/api';
import { useSessionGuard } from '@/shared/auth';
import { useLatestRef } from '@/shared/hooks';
import { translateApiError } from '@/shared/i18n';
import { useFeedback } from '@/shared/lib/feedback';
import { LoadFailure, ResourcePicker } from '@/shared/ui';

const MIN_RETENTION_DAYS = 1;
const MAX_RETENTION_DAYS = 3650;

interface Props {
  destinationHealth: InstanceBackupDestinationHealth[];
  workspaces: Workspace[];
  onWorkspaceCreated: (workspace: Workspace) => void;
  onSaved: (settings: InstanceBackupSettings) => Promise<void>;
  onClose: () => void;
}

export function InstanceBackupSettingsForm({
  destinationHealth,
  workspaces,
  onWorkspaceCreated,
  onSaved,
  onClose,
}: Props) {
  const { t, i18n } = useTranslation();
  const notify = useFeedback();
  const isCurrent = useSessionGuard();
  const pickerId = useId();
  const retentionId = useId();
  const storagesHeadingId = useId();

  const settingsQuery = useSessionQuery({
    queryKey: ['instance-backup-settings'],
    queryFn: (signal) => instanceBackupApi.getSettings({ signal }),
  });
  const choicesQuery = useSessionQuery({
    queryKey: ['instance-backup-destination-choices'],
    queryFn: (signal) => instanceBackupApi.listAllDestinationChoices({ signal }),
  });
  const updateSettingsMutation = useSessionMutation({
    mutationFn: (settings: InstanceBackupSettings) => instanceBackupApi.updateSettings(settings),
  });

  const [acknowledged, setAcknowledged] = useState<InstanceBackupSettings>();
  const [draft, setDraft] = useState<InstanceBackupSettings>();
  const draftRef = useLatestRef(draft);
  const [saveError, setSaveError] = useState<unknown>();
  const [isPickerOpen, setIsPickerOpen] = useState(false);
  const [isCreatingDestination, setIsCreatingDestination] = useState(false);
  const [createdChoices, setCreatedChoices] = useState<InstanceBackupDestinationChoice[]>([]);
  const pickerRef = useRef<HTMLButtonElement | HTMLInputElement>(null);
  const isSaving = updateSettingsMutation.isPending;
  const loadedChoices = choicesQuery.data ?? [];
  const choices = [
    ...loadedChoices,
    ...createdChoices.filter(
      (created) => !loadedChoices.some((choice) => choice.storageId === created.storageId),
    ),
  ];

  const returnFocusToPicker = useEffectEvent(() => {
    pickerRef.current?.focus();
  });

  useEffect(() => {
    if (!isCreatingDestination) return;

    return () => {
      returnFocusToPicker();
    };
  }, [isCreatingDestination]);

  const acceptCreatedStorage = (storage: Storage, workspace: Workspace) => {
    if (!isCurrent()) return;

    setCreatedChoices((current) => [
      ...current,
      {
        storageId: storage.id,
        storageName: storage.name,
        storageType: storage.settings.type,
        workspaceId: workspace.id,
        workspaceName: workspace.name,
        isSelected: false,
      },
    ]);
    setDraft((current) =>
      current && !current.storageIds.includes(storage.id)
        ? { ...current, storageIds: [...current.storageIds, storage.id] }
        : current,
    );
    void choicesQuery.refetch();
  };

  useEffect(() => {
    if (!settingsQuery.data) return;

    const loadedSettings = settingsQuery.data;

    // The form is opened for one edit, so the first read is its baseline. Later cache updates, such
    // as the owner storing a saved result, must not overwrite an edit made while saving.
    setAcknowledged((current) => current ?? loadedSettings);
    setDraft((current) => current ?? loadedSettings);
  }, [settingsQuery.data]);

  const updateDraft = (change: Partial<InstanceBackupSettings>) =>
    setDraft((current) => (current ? { ...current, ...change } : current));

  const toggleDestination = (storageId: string) =>
    setDraft((current) =>
      current
        ? {
            ...current,
            storageIds: current.storageIds.includes(storageId)
              ? current.storageIds.filter((id) => id !== storageId)
              : [...current.storageIds, storageId],
          }
        : current,
    );

  const removeDestination = (storageId: string) =>
    setDraft((current) =>
      current
        ? { ...current, storageIds: current.storageIds.filter((id) => id !== storageId) }
        : current,
    );

  const save = async () => {
    if (!draft || !acknowledged || isSaving) return;

    const submitted = draft;
    const wasProtected = acknowledged.storageIds.length > 0;
    setSaveError(undefined);

    try {
      const saved = await updateSettingsMutation.mutateAsync(submitted);
      if (!isCurrent()) return;

      const nextDraft = draftRef.current
        ? acknowledgeInstanceBackupSettings({
            current: draftRef.current,
            submitted,
            acknowledged: saved,
          })
        : saved;
      setAcknowledged(saved);
      setDraft(nextDraft);

      if (wasProtected && saved.storageIds.length === 0) {
        notify({ kind: 'warning', title: i18n.t('instanceBackups.settings.stopped') });
      } else if (!wasProtected && saved.storageIds.length > 0) {
        notify({ kind: 'success', title: i18n.t('instanceBackups.settings.firstBackupRequested') });
      } else {
        notify({ kind: 'success', title: i18n.t('instanceBackups.settings.saved') });
      }

      await onSaved(saved);
      if (!isCurrent()) return;

      // An edit made while the save was in flight stays open instead of being dropped with the form.
      if (!isInstanceBackupSettingsChanged({ acknowledged: saved, shown: nextDraft })) onClose();
    } catch (error: unknown) {
      if (isCurrent()) setSaveError(error);
    } finally {
      if (isCurrent()) updateSettingsMutation.reset();
    }
  };

  if (settingsQuery.error) {
    return (
      <LoadFailure
        className="mt-3"
        error={settingsQuery.error}
        onRetry={() => void settingsQuery.refetch()}
      />
    );
  }

  if (!draft || !acknowledged) {
    return <Loader className="mt-3" role="status" />;
  }

  const isChanged = isInstanceBackupSettingsChanged({ acknowledged, shown: draft });
  const isComplete =
    isScheduleComplete(draft.schedule) &&
    Number.isInteger(draft.retentionDays) &&
    draft.retentionDays >= MIN_RETENTION_DAYS &&
    draft.retentionDays <= MAX_RETENTION_DAYS;

  return (
    <div>
      <p className="text-muted">{t('instanceBackups.retainSeparately')}</p>

      <section className="mt-5" aria-labelledby={storagesHeadingId}>
        <h3 id={storagesHeadingId} className="font-medium">
          {t('instanceBackups.settings.storages')}
        </h3>
        <p className="mt-0.5 text-xs text-muted">{t('instanceBackups.destinations.description')}</p>

        {choicesQuery.error ? (
          <LoadFailure
            className="mt-2"
            error={choicesQuery.error}
            onRetry={() => void choicesQuery.refetch()}
          />
        ) : (
          <ResourcePicker
            id={pickerId}
            className="mt-2"
            size="sm"
            label={t('instanceBackups.destinations.picker')}
            placeholder={t('instanceBackups.destinations.pickerPlaceholder')}
            searchPlaceholder={t('storages.list.searchPlaceholder')}
            data={choices.map((choice) => ({
              value: choice.storageId,
              label: t('instanceBackups.destinations.choice', {
                storage: choice.storageName,
                workspace: choice.workspaceName,
                provider: t(STORAGE_TYPE_LABEL_KEYS[choice.storageType]),
              }),
            }))}
            selectedValues={draft.storageIds}
            onSelect={toggleDestination}
            onRemove={removeDestination}
            isMultiple
            isSearchable
            isLoading={choicesQuery.isLoading}
            isOpen={isPickerOpen}
            onOpenChange={setIsPickerOpen}
            targetRef={pickerRef}
            createLabel={t('instanceBackups.destinations.create')}
            onCreate={() => {
              setIsPickerOpen(false);
              setIsCreatingDestination(true);
            }}
          />
        )}

        <InstanceBackupDestinationList
          storageIds={draft.storageIds}
          choices={choices}
          destinationHealth={destinationHealth}
          isDisabled={isSaving}
          onRemove={removeDestination}
        />
      </section>

      {isCreatingDestination && (
        <CreateDestinationDialog
          workspaces={workspaces}
          onWorkspaceCreated={onWorkspaceCreated}
          onStorageCreated={acceptCreatedStorage}
          onClose={() => setIsCreatingDestination(false)}
        />
      )}

      <section className="mt-5 border-t border-border">
        <ScheduleEditor
          label={t('instanceBackups.settings.schedule')}
          schedule={draft.schedule}
          size="sm"
          onChange={(schedule) => updateDraft({ schedule })}
        />

        <div className="mt-2 flex w-full flex-col items-start sm:flex-row sm:items-center">
          <label
            htmlFor={retentionId}
            className="mb-1 min-w-0 wrap-anywhere sm:mb-0 sm:w-[150px] sm:shrink-0 sm:pr-2"
          >
            {t('instanceBackups.settings.retentionDays')}
          </label>
          <NumberInput
            id={retentionId}
            size="sm"
            className="w-full max-w-[200px]"
            min={MIN_RETENTION_DAYS}
            max={MAX_RETENTION_DAYS}
            allowDecimal={false}
            allowNegative={false}
            value={draft.retentionDays || ''}
            onChange={(value) =>
              updateDraft({ retentionDays: typeof value === 'number' ? value : 0 })
            }
          />
        </div>
      </section>

      {saveError !== undefined && (
        <Alert
          className="mt-3"
          color="danger"
          icon={<CircleAlert aria-hidden size={16} />}
          title={translateApiError(saveError, t)}
        />
      )}

      <div className="mt-5 flex flex-wrap justify-end gap-2 border-t border-border pt-4">
        <Button
          className="!h-auto min-h-8 max-w-full !whitespace-normal"
          variant="default"
          disabled={isSaving}
          onClick={onClose}
        >
          {t('common.actions.cancel')}
        </Button>
        <Button
          className="!h-auto min-h-8 max-w-full !whitespace-normal"
          variant="filled"
          onClick={() => void save()}
          loading={isSaving}
          disabled={isSaving || !isChanged || !isComplete}
          aria-busy={isSaving}
        >
          {t('instanceBackups.settings.save')}
        </Button>
      </div>
    </div>
  );
}
