import { InfoCircleOutlined } from '@ant-design/icons';
import { Alert, Button, Checkbox, Select, Spin, Switch, Tooltip } from 'antd';
import { useEffect, useId, useState } from 'react';
import { Trans, useTranslation } from 'react-i18next';

import {
  SCHEDULE_TYPE_LABEL_KEYS,
  ScheduleDetailsFields,
  ScheduleType,
} from '../../../../entities/schedules';
import { useSessionGuard } from '../../../../shared/auth';
import { useLatestRef } from '../../../../shared/hooks';
import { getWebsitePageUrl, translateApiError, useLocale } from '../../../../shared/i18n';
import { LoadFailure } from '../../../../shared/ui';
import { verificationConfigApi } from '../../api/verificationConfigApi';
import { buildVerificationConfigForSave } from '../../model/verification/buildVerificationConfigForSave';
import { isVerificationConfigChanged } from '../../model/verification/isVerificationConfigChanged';
import { isVerificationConfigComplete } from '../../model/verification/isVerificationConfigComplete';
import { type VerificationConfig } from '../../model/verification/VerificationConfig';
import { VerificationNotificationType } from '../../model/verification/VerificationNotificationType';
import { VERIFICATION_NOTIFICATION_TYPE_LABEL_KEYS } from '../../model/verification/VerificationNotificationTypeLabelKeys';
import { VerificationScheduleType } from '../../model/verification/VerificationScheduleType';

type VerificationScheduleChoice = typeof VerificationScheduleType.AFTER_BACKUP | ScheduleType;

interface Props {
  databaseId: string;
  onClose: () => void;
}

export const EditVerificationConfigComponent = ({ databaseId, onClose }: Props) => {
  const { t } = useTranslation();
  const { locale } = useLocale();
  const isActive = useSessionGuard();
  const shownDatabaseId = useLatestRef(databaseId);
  const scheduleChoiceControlId = useId();

  const [acknowledgedConfig, setAcknowledgedConfig] = useState<VerificationConfig>();
  const [draft, setDraft] = useState<VerificationConfig>();
  const [loadError, setLoadError] = useState<unknown>();
  const [loadAttempt, setLoadAttempt] = useState(0);

  const [isSaving, setIsSaving] = useState(false);
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

    setIsSaving(true);
    setSaveError(undefined);

    try {
      await verificationConfigApi.save(
        databaseId,
        buildVerificationConfigForSave({ acknowledgedConfig, draft }),
      );
      if (!isShownRequest()) return;

      onClose();
    } catch (error) {
      if (!isShownRequest()) return;

      setSaveError(error);
      setIsSaving(false);
    }
  };

  useEffect(() => {
    let isCurrentLoad = true;

    setLoadError(undefined);
    setAcknowledgedConfig(undefined);
    setDraft(undefined);
    setSaveError(undefined);
    setIsSaving(false);

    verificationConfigApi
      .getByDatabaseId(databaseId)
      .then((loadedConfig) => {
        if (!isCurrentLoad || !isActive()) return;

        setAcknowledgedConfig(loadedConfig);
        setDraft(loadedConfig);
      })
      .catch((error: unknown) => {
        if (isCurrentLoad && isActive()) setLoadError(error);
      });

    return () => {
      isCurrentLoad = false;
    };
  }, [databaseId, loadAttempt, isActive]);

  if (loadError !== undefined) {
    return (
      <div>
        <LoadFailure error={loadError} onRetry={() => setLoadAttempt((attempt) => attempt + 1)} />

        <div className="mt-6 flex justify-end">
          <Button onClick={onClose}>{t('common.actions.cancel')}</Button>
        </div>
      </div>
    );
  }

  if (!draft || !acknowledgedConfig) {
    return <Spin size="small" />;
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
          onChange={(isAutomaticVerificationEnabled) =>
            updateDraft({ isAutomaticVerificationEnabled })
          }
          size="small"
        />
        <Tooltip
          className="cursor-pointer"
          title={
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
          <InfoCircleOutlined className="ml-2 text-muted" />
        </Tooltip>
      </div>

      {draft.isAutomaticVerificationEnabled && (
        <>
          <div className="mt-5 mb-1 flex w-full flex-col items-start sm:flex-row sm:items-center">
            <label htmlFor={scheduleChoiceControlId} className="mb-1 min-w-[150px] sm:mb-0 sm:pr-2">
              {t('verification.config.interval')}
            </label>
            <Select<VerificationScheduleChoice>
              id={scheduleChoiceControlId}
              value={isAfterBackup ? VerificationScheduleType.AFTER_BACKUP : draft.schedule.type}
              onChange={selectSchedule}
              size="small"
              className="w-full max-w-[200px] grow"
              options={[
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
            <div className="mt-0 mb-1 min-w-[150px] sm:mt-1 sm:mb-0 sm:pr-2">
              {t('verification.config.notifications')}
            </div>
            <div
              role="group"
              aria-label={t('verification.config.notifications')}
              className="flex flex-col space-y-2"
            >
              {Object.values(VerificationNotificationType).map((notificationType) => (
                <Checkbox
                  key={notificationType}
                  checked={draft.notificationTriggers.includes(notificationType)}
                  onChange={(e) => toggleNotification(notificationType, e.target.checked)}
                >
                  {t(VERIFICATION_NOTIFICATION_TYPE_LABEL_KEYS[notificationType])}
                </Checkbox>
              ))}
            </div>
          </div>
        </>
      )}

      {saveError !== undefined && (
        <Alert className="mt-5" type="error" showIcon message={translateApiError(saveError, t)} />
      )}

      <div className="mt-6 flex justify-end space-x-2">
        <Button onClick={onClose} disabled={isSaving}>
          {t('common.actions.cancel')}
        </Button>

        <Button type="primary" onClick={saveDraft} loading={isSaving} disabled={!canSave}>
          {t('common.actions.save')}
        </Button>
      </div>
    </div>
  );
};
