import { InfoCircleOutlined } from '@ant-design/icons';
import { Alert, Button, Input, Spin, Switch, Tooltip } from 'antd';
import { useEffect, useId, useState } from 'react';
import { useTranslation } from 'react-i18next';

import { healthcheckConfigApi } from '../../../../entities/healthcheck';
import type { HealthcheckConfig } from '../../../../entities/healthcheck';
import { useSessionGuard } from '../../../../shared/auth';
import { useLatestRef } from '../../../../shared/hooks';
import { translateApiError } from '../../../../shared/i18n';
import { LoadFailure } from '../../../../shared/ui';

interface Props {
  databaseId: string;
  onClose: () => void;
}

export const EditHealthcheckConfigComponent = ({ databaseId, onClose }: Props) => {
  const { t } = useTranslation();
  const isActive = useSessionGuard();
  const latestDatabaseId = useLatestRef(databaseId);
  const enableFieldId = useId();
  const notifyFieldId = useId();
  const intervalFieldId = useId();
  const attemptsFieldId = useId();
  const retentionFieldId = useId();

  const [isSaving, setIsSaving] = useState(false);
  const [isUnsaved, setIsUnsaved] = useState(false);
  const [saveError, setSaveError] = useState<unknown>();
  const [healthcheckConfig, setHealthcheckConfig] = useState<HealthcheckConfig | undefined>(
    undefined,
  );
  const [loadError, setLoadError] = useState<unknown>();
  const [loadAttempt, setLoadAttempt] = useState(0);

  const handleSave = async () => {
    if (!healthcheckConfig) return;

    const isCurrentSave = () => isActive() && latestDatabaseId.current === databaseId;

    setIsSaving(true);
    setSaveError(undefined);

    try {
      await healthcheckConfigApi.saveHealthcheckConfig(healthcheckConfig);

      if (!isCurrentSave()) return;

      setIsUnsaved(false);
      onClose();
    } catch (e) {
      if (isCurrentSave()) setSaveError(e);
    } finally {
      if (isCurrentSave()) setIsSaving(false);
    }
  };

  useEffect(() => {
    let isCurrentLoad = true;

    setHealthcheckConfig(undefined);
    setLoadError(undefined);
    setSaveError(undefined);
    setIsUnsaved(false);
    setIsSaving(false);

    healthcheckConfigApi
      .getHealthcheckConfig(databaseId)
      .then((config) => {
        if (isCurrentLoad && isActive()) setHealthcheckConfig(config);
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

  if (!healthcheckConfig) {
    return <Spin size="small" />;
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
          onChange={(checked) => {
            setHealthcheckConfig({
              ...healthcheckConfig,
              isHealthcheckEnabled: checked,
            });
            setIsUnsaved(true);
          }}
          size="small"
        />

        <Tooltip className="cursor-pointer" title={t('healthcheck.config.tooltips.enable')}>
          <InfoCircleOutlined className="ml-2 text-muted" />
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
              onChange={(checked) => {
                setHealthcheckConfig({
                  ...healthcheckConfig,
                  isNotificationEnabled: checked,
                });
                setIsUnsaved(true);
              }}
              size="small"
            />

            <Tooltip
              className="cursor-pointer"
              title={t('healthcheck.config.tooltips.notifyWhenUnavailable')}
            >
              <InfoCircleOutlined className="ml-2 text-muted" />
            </Tooltip>
          </div>

          <div className="mb-1 flex items-center">
            <label className="min-w-[180px] pr-2" htmlFor={intervalFieldId}>
              {t('healthcheck.config.checkIntervalMinutes')}
            </label>

            <Input
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
              size="small"
              className="w-full max-w-[250px]"
              placeholder="5"
              min={1}
            />

            <Tooltip
              className="cursor-pointer"
              title={t('healthcheck.config.tooltips.checkIntervalMinutes')}
            >
              <InfoCircleOutlined className="ml-2 text-muted" />
            </Tooltip>
          </div>

          <div className="mb-1 flex items-center">
            <label className="min-w-[180px] pr-2" htmlFor={attemptsFieldId}>
              {t('healthcheck.config.attemptsBeforeDown')}
            </label>

            <Input
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
              size="small"
              className="w-full max-w-[250px]"
              placeholder="3"
              min={1}
            />

            <Tooltip
              className="cursor-pointer"
              title={t('healthcheck.config.tooltips.attemptsBeforeDown')}
            >
              <InfoCircleOutlined className="ml-2 text-muted" />
            </Tooltip>
          </div>

          <div className="mb-1 flex items-center">
            <label className="min-w-[180px] pr-2" htmlFor={retentionFieldId}>
              {t('healthcheck.config.attemptRetentionDays')}
            </label>

            <Input
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
              size="small"
              className="w-full max-w-[250px]"
              placeholder="30"
              min={1}
            />

            <Tooltip
              className="cursor-pointer"
              title={t('healthcheck.config.tooltips.attemptRetentionDays')}
            >
              <InfoCircleOutlined className="ml-2 text-muted" />
            </Tooltip>
          </div>
        </>
      )}

      {saveError !== undefined && (
        <Alert type="error" showIcon message={translateApiError(saveError, t)} />
      )}

      <div className="mt-6 flex justify-end space-x-2">
        <Button onClick={onClose} disabled={isSaving}>
          {t('common.actions.cancel')}
        </Button>

        {isUnsaved && (
          <Button type="primary" onClick={handleSave} loading={isSaving}>
            {t('common.actions.save')}
          </Button>
        )}
      </div>
    </div>
  );
};
