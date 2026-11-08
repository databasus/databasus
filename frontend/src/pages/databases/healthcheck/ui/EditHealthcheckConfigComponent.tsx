import { Alert, Button, Loader, Switch, TextInput, Tooltip } from '@mantine/core';
import { CircleAlert, Info } from 'lucide-react';
import { useEffect, useId, useState } from 'react';
import { useTranslation } from 'react-i18next';

import { healthcheckConfigApi, type HealthcheckConfig } from '@/entities/healthcheck';
import { useSessionMutation, useSessionQuery } from '@/shared/api';
import { useSessionGuard } from '@/shared/auth';
import { useLatestRef } from '@/shared/hooks';
import { translateApiError } from '@/shared/i18n';
import { LoadFailure } from '@/shared/ui';

interface Props {
  databaseId: string;
  onClose: () => void;
}

export const EditHealthcheckConfigComponent = ({ databaseId, onClose }: Props) => {
  const saveHealthcheckConfigMutation = useSessionMutation({
    mutationFn: (parameters: Parameters<typeof healthcheckConfigApi.saveHealthcheckConfig>) =>
      healthcheckConfigApi.saveHealthcheckConfig(...parameters),
  });
  const { reset: resetRequest } = saveHealthcheckConfigMutation;

  const { t } = useTranslation();
  const isActive = useSessionGuard();
  const latestDatabaseId = useLatestRef(databaseId);
  const enableFieldId = useId();
  const notifyFieldId = useId();
  const intervalFieldId = useId();
  const attemptsFieldId = useId();
  const retentionFieldId = useId();

  const isSaving = saveHealthcheckConfigMutation.isPending;
  const [isUnsaved, setIsUnsaved] = useState(false);
  const [saveError, setSaveError] = useState<unknown>();
  const [healthcheckConfig, setHealthcheckConfig] = useState<HealthcheckConfig | undefined>(
    undefined,
  );
  const query = useSessionQuery({
    queryKey: ['healthcheck-config', databaseId],
    queryFn: (signal) => healthcheckConfigApi.getHealthcheckConfig(databaseId, { signal }),
  });
  const loadError = query.error ?? undefined;

  const handleSave = async () => {
    if (!healthcheckConfig) return;

    const isCurrentSave = () => isActive() && latestDatabaseId.current === databaseId;

    setSaveError(undefined);

    try {
      await saveHealthcheckConfigMutation.mutateAsync([healthcheckConfig]);

      if (!isCurrentSave()) return;

      setIsUnsaved(false);
      onClose();
    } catch (e) {
      if (isCurrentSave()) setSaveError(e);
    } finally {
      if (isCurrentSave()) resetRequest();
    }
  };

  useEffect(() => {
    setHealthcheckConfig(undefined);
    setIsUnsaved(false);
    setSaveError(undefined);
    resetRequest();
  }, [databaseId, resetRequest]);
  useEffect(() => {
    if (isActive() && query.data !== undefined) {
      setHealthcheckConfig(query.data);
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

  if (!healthcheckConfig) {
    return <Loader size="xs" role="status" />;
  }

  return (
    <div className="space-y-4">
      <div className="mb-1 flex items-center">
        <label className="min-w-[180px] pr-2" htmlFor={enableFieldId}>
          {t('healthcheck.config.enable')}
        </label>
        <Switch
          id={enableFieldId}
          checked={healthcheckConfig.isHealthcheckEnabled}
          onChange={(event) => {
            setHealthcheckConfig({
              ...healthcheckConfig,
              isHealthcheckEnabled: event.currentTarget.checked,
            });
            setIsUnsaved(true);
          }}
          size="xs"
        />

        <Tooltip className="cursor-pointer" label={t('healthcheck.config.tooltips.enable')}>
          <Info className="ml-2 inline-block shrink-0 align-middle text-muted" size="1em" />
        </Tooltip>
      </div>

      {healthcheckConfig.isHealthcheckEnabled && (
        <>
          <div className="mb-1 flex items-center">
            <label className="min-w-[180px] pr-2" htmlFor={notifyFieldId}>
              {t('healthcheck.config.notifyWhenUnavailable')}
            </label>

            <Switch
              id={notifyFieldId}
              checked={healthcheckConfig.isNotificationEnabled}
              onChange={(event) => {
                setHealthcheckConfig({
                  ...healthcheckConfig,
                  isNotificationEnabled: event.currentTarget.checked,
                });
                setIsUnsaved(true);
              }}
              size="xs"
            />

            <Tooltip
              className="cursor-pointer"
              label={t('healthcheck.config.tooltips.notifyWhenUnavailable')}
            >
              <Info className="ml-2 inline-block shrink-0 align-middle text-muted" size="1em" />
            </Tooltip>
          </div>

          <div className="mb-1 flex items-center">
            <label className="min-w-[180px] pr-2" htmlFor={intervalFieldId}>
              {t('healthcheck.config.checkIntervalMinutes')}
            </label>

            <TextInput
              id={intervalFieldId}
              type="number"
              value={healthcheckConfig.intervalMinutes}
              onChange={(e) => {
                const value = Number(e.target.value);
                if (value > 0) {
                  setHealthcheckConfig({
                    ...healthcheckConfig,
                    intervalMinutes: value,
                  });
                  setIsUnsaved(true);
                }
              }}
              size="xs"
              className="w-full max-w-[250px]"
              placeholder="5"
              min={1}
            />

            <Tooltip
              className="cursor-pointer"
              label={t('healthcheck.config.tooltips.checkIntervalMinutes')}
            >
              <Info className="ml-2 inline-block shrink-0 align-middle text-muted" size="1em" />
            </Tooltip>
          </div>

          <div className="mb-1 flex items-center">
            <label className="min-w-[180px] pr-2" htmlFor={attemptsFieldId}>
              {t('healthcheck.config.attemptsBeforeDown')}
            </label>

            <TextInput
              id={attemptsFieldId}
              type="number"
              value={healthcheckConfig.failedAttemptsBeforeUnavailable}
              onChange={(e) => {
                const value = Number(e.target.value);
                if (value > 0) {
                  setHealthcheckConfig({
                    ...healthcheckConfig,
                    failedAttemptsBeforeUnavailable: value,
                  });
                  setIsUnsaved(true);
                }
              }}
              size="xs"
              className="w-full max-w-[250px]"
              placeholder="3"
              min={1}
            />

            <Tooltip
              className="cursor-pointer"
              label={t('healthcheck.config.tooltips.attemptsBeforeDown')}
            >
              <Info className="ml-2 inline-block shrink-0 align-middle text-muted" size="1em" />
            </Tooltip>
          </div>

          <div className="mb-1 flex items-center">
            <label className="min-w-[180px] pr-2" htmlFor={retentionFieldId}>
              {t('healthcheck.config.attemptRetentionDays')}
            </label>

            <TextInput
              id={retentionFieldId}
              type="number"
              value={healthcheckConfig.attemptRetentionDays}
              onChange={(e) => {
                const value = Number(e.target.value);
                if (value > 0) {
                  setHealthcheckConfig({
                    ...healthcheckConfig,
                    attemptRetentionDays: value,
                  });
                  setIsUnsaved(true);
                }
              }}
              size="xs"
              className="w-full max-w-[250px]"
              placeholder="30"
              min={1}
            />

            <Tooltip
              className="cursor-pointer"
              label={t('healthcheck.config.tooltips.attemptRetentionDays')}
            >
              <Info className="ml-2 inline-block shrink-0 align-middle text-muted" size="1em" />
            </Tooltip>
          </div>
        </>
      )}

      {saveError !== undefined && (
        <Alert
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

        {isUnsaved && (
          <Button variant="filled" onClick={handleSave} loading={isSaving} aria-busy={isSaving}>
            {t('common.actions.save')}
          </Button>
        )}
      </div>
    </div>
  );
};
