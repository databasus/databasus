import { Alert, Button, Checkbox, Loader, Switch, Tooltip } from '@mantine/core';
import { CircleAlert, Info } from 'lucide-react';
import { useEffect, useId, useState } from 'react';
import { Trans, useTranslation } from 'react-i18next';

import {
  SCHEDULE_TYPE_LABEL_KEYS,
  ScheduleDetailsFields,
  ScheduleType,
} from '@/entities/schedules';
import { verificationConfigApi } from '@/pages/databases/verification/api/verificationConfigApi';
import { buildVerificationConfigForSave } from '@/pages/databases/verification/model/buildVerificationConfigForSave';
import { isVerificationConfigChanged } from '@/pages/databases/verification/model/isVerificationConfigChanged';
import { isVerificationConfigComplete } from '@/pages/databases/verification/model/isVerificationConfigComplete';
import { type VerificationConfig } from '@/pages/databases/verification/model/verificationContract';
import {
  VerificationNotificationType,
  VERIFICATION_NOTIFICATION_TYPE_LABEL_KEYS,
} from '@/pages/databases/verification/model/VerificationNotificationType';
import { VerificationScheduleType } from '@/pages/databases/verification/model/VerificationScheduleType';
import { useSessionMutation, useSessionQuery } from '@/shared/api';
import { useSessionGuard } from '@/shared/auth';
import { useLatestRef } from '@/shared/hooks';
import { getWebsitePageUrl, translateApiError, useLocale } from '@/shared/i18n';
import { WrappingSelect, LoadFailure } from '@/shared/ui';

type VerificationScheduleChoice = typeof VerificationScheduleType.AFTER_BACKUP | ScheduleType;

interface Props {
  databaseId: string;
  onClose: () => void;
}

export const EditVerificationConfigComponent = ({ databaseId, onClose }: Props) => {
  const saveMutation = useSessionMutation({
    mutationFn: (parameters: Parameters<typeof verificationConfigApi.save>) =>
      verificationConfigApi.save(...parameters),
  });
  const { reset: resetRequest } = saveMutation;

  const { t } = useTranslation();
  const { locale } = useLocale();
  const isActive = useSessionGuard();
  const shownDatabaseId = useLatestRef(databaseId);
  const scheduleChoiceControlId = useId();

  const [acknowledgedConfig, setAcknowledgedConfig] = useState<VerificationConfig>();
  const [draft, setDraft] = useState<VerificationConfig>();
  const query = useSessionQuery({
    queryKey: ['verification-config', databaseId],
    queryFn: (signal) => verificationConfigApi.getByDatabaseId(databaseId, { signal }),
  });
  const loadError = query.error ?? undefined;

  const isSaving = saveMutation.isPending;
  const [saveError, setSaveError] = useState<unknown>();

  const updateDraft = (patch: Partial<VerificationConfig>) => {
    setDraft((previousDraft) => previousDraft && { ...previousDraft, ...patch });
  };

  const selectSchedule = (choice: VerificationScheduleChoice) => {
    if (choice === VerificationScheduleType.AFTER_BACKUP) {
      updateDraft({ scheduleType: VerificationScheduleType.AFTER_BACKUP });
      return;
    }

    setDraft(
      (previousDraft) =>
        previousDraft && {
          ...previousDraft,
          scheduleType: VerificationScheduleType.INTERVAL,
          schedule: { ...previousDraft.schedule, type: choice },
        },
    );
  };

  const toggleNotification = (type: VerificationNotificationType, isChecked: boolean) => {
    setDraft(
      (previousDraft) =>
        previousDraft && {
          ...previousDraft,
          notificationTriggers: isChecked
            ? [...previousDraft.notificationTriggers, type]
            : previousDraft.notificationTriggers.filter((trigger) => trigger !== type),
        },
    );
  };

  const saveDraft = async () => {
    if (!acknowledgedConfig || !draft) return;

    const isShownRequest = () => isActive() && shownDatabaseId.current === databaseId;

    setSaveError(undefined);

    try {
      await saveMutation.mutateAsync([
        databaseId,
        buildVerificationConfigForSave({ acknowledgedConfig, draft }),
      ]);
      if (!isShownRequest()) return;

      onClose();
    } catch (error) {
      if (!isShownRequest()) return;

      setSaveError(error);
      resetRequest();
    }
  };

  useEffect(() => {
    setAcknowledgedConfig(undefined);
    setDraft(undefined);
    setSaveError(undefined);
    resetRequest();
  }, [databaseId, resetRequest]);
  useEffect(() => {
    if (isActive() && query.data !== undefined) {
      setAcknowledgedConfig(query.data);
      setDraft(query.data);
    }
  }, [query.data, isActive]);

  if (loadError !== undefined) {
    return (
      <div>
        <LoadFailure error={loadError} onRetry={() => void query.refetch()} />

        <div className="mt-6 flex justify-end">
          <Button onClick={onClose}>{t('common.actions.cancel')}</Button>
        </div>
      </div>
    );
  }

  if (!draft || !acknowledgedConfig) {
    return <Loader size="xs" role="status" />;
  }

  const isAfterBackup = draft.scheduleType === VerificationScheduleType.AFTER_BACKUP;

  const canSave =
    isVerificationConfigChanged({ acknowledged: acknowledgedConfig, shown: draft }) &&
    isVerificationConfigComplete(draft);

  return (
    <div className="space-y-3">
      <div className="mb-1 flex w-full flex-col items-start sm:flex-row sm:items-center">
        <div className="mb-1 min-w-[150px] sm:mb-0 sm:pr-2">
          {t('verification.config.scheduledVerification')}
        </div>
        <Switch
          checked={draft.isAutomaticVerificationEnabled}
          aria-label={t('verification.config.scheduledVerification')}
          onChange={(event) =>
            updateDraft({ isAutomaticVerificationEnabled: event.currentTarget.checked })
          }
          size="xs"
        />
        <Tooltip
          className="cursor-pointer"
          label={
            <Trans
              i18nKey="verification.config.scheduledVerificationTooltip"
              components={{
                docsLink: (
                  <a
                    href={getWebsitePageUrl('restoreVerification', locale)}
                    target="_blank"
                    rel="noopener noreferrer"
                    className="underline"
                  />
                ),
              }}
            />
          }
        >
          <Info className="ml-2 inline-block shrink-0 align-middle text-muted" size="1em" />
        </Tooltip>
      </div>

      {draft.isAutomaticVerificationEnabled && (
        <>
          <div className="mt-5 mb-1 flex w-full flex-col items-start sm:flex-row sm:items-center">
            <label htmlFor={scheduleChoiceControlId} className="mb-1 min-w-[150px] sm:mb-0 sm:pr-2">
              {t('verification.config.interval')}
            </label>
            <WrappingSelect<VerificationScheduleChoice>
              id={scheduleChoiceControlId}
              value={isAfterBackup ? VerificationScheduleType.AFTER_BACKUP : draft.schedule.type}
              onChange={selectSchedule}
              size="xs"
              className="w-full max-w-[200px] grow"
              data={[
                {
                  label: t('verification.config.afterBackup'),
                  value: VerificationScheduleType.AFTER_BACKUP,
                },
                ...Object.values(ScheduleType).map((scheduleType) => ({
                  label: t(SCHEDULE_TYPE_LABEL_KEYS[scheduleType]),
                  value: scheduleType,
                })),
              ]}
            />
          </div>

          {isAfterBackup ? (
            <div className="mb-1 flex w-full flex-col items-start text-xs text-muted sm:flex-row sm:items-center">
              <div className="mb-1 min-w-[150px] sm:mb-0 sm:pr-2" />
              <div>{t('verification.config.afterBackupHint')}</div>
            </div>
          ) : (
            <ScheduleDetailsFields
              schedule={draft.schedule}
              onChange={(schedule) => updateDraft({ schedule })}
            />
          )}

          <div className="mt-5 mb-1 flex w-full flex-col items-start sm:flex-row sm:items-start">
            <div className="mb-1 min-w-0 wrap-anywhere sm:mb-0 sm:w-[150px] sm:shrink-0 sm:pr-2">
              {t('verification.config.notifications')}
            </div>
            <div
              role="group"
              aria-label={t('verification.config.notifications')}
              className="flex min-w-0 flex-col space-y-1"
            >
              {Object.values(VerificationNotificationType).map((notificationType) => (
                <Checkbox
                  key={notificationType}
                  checked={draft.notificationTriggers.includes(notificationType)}
                  onChange={(e) => toggleNotification(notificationType, e.target.checked)}
                  label={<>{t(VERIFICATION_NOTIFICATION_TYPE_LABEL_KEYS[notificationType])}</>}
                />
              ))}
            </div>
          </div>
        </>
      )}

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

      <div className="mt-6 flex justify-end space-x-2">
        <Button onClick={onClose} disabled={isSaving}>
          {t('common.actions.cancel')}
        </Button>

        <Button
          variant="filled"
          onClick={saveDraft}
          loading={isSaving}
          disabled={!canSave}
          aria-busy={isSaving}
        >
          {t('common.actions.save')}
        </Button>
      </div>
    </div>
  );
};
