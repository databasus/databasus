import { InfoCircleOutlined } from '@ant-design/icons';
import { Button, Input, Spin, Switch, Tooltip } from 'antd';
import { useEffect, useState } from 'react';
import { useTranslation } from 'react-i18next';

import { healthcheckConfigApi } from '../../../../entities/healthcheck';
import type { HealthcheckConfig } from '../../../../entities/healthcheck';
import { translateApiError } from '../../../../shared/i18n';

interface Props {
  databaseId: string;
  onClose: () => void;
}

export const EditHealthcheckConfigComponent = ({ databaseId, onClose }: Props) => {
  const { t } = useTranslation();
  const [isLoading, setIsLoading] = useState(false);
  const [isSaving, setIsSaving] = useState(false);
  const [isUnsaved, setIsUnsaved] = useState(false);
  const [healthcheckConfig, setHealthcheckConfig] = useState<HealthcheckConfig | undefined>(
    undefined,
  );

  const handleSave = async () => {
    if (!healthcheckConfig) return;

    setIsSaving(true);

    try {
      await healthcheckConfigApi.saveHealthcheckConfig(healthcheckConfig);
      setIsUnsaved(false);
      onClose();
    } catch (e) {
      alert(translateApiError(e, t));
    }

    setIsSaving(false);
  };

  useEffect(() => {
    let isActive = true;
    setIsLoading(true);
    healthcheckConfigApi
      .getHealthcheckConfig(databaseId)
      .then((config) => {
        if (isActive) setHealthcheckConfig(config);
      })
      .catch((error) => {
        if (isActive) alert(translateApiError(error, t));
      })
      .finally(() => {
        if (isActive) setIsLoading(false);
      });

    return () => {
      isActive = false;
    };
  }, [databaseId]);

  if (isLoading) {
    return <Spin size="small" />;
  }

  if (!healthcheckConfig) {
    return <div />;
  }

  return (
    <div className="space-y-4">
      <div className="mb-1 flex items-center">
        <div className="min-w-[180px] pr-2">{t('healthcheck.config.enable')}</div>
        <Switch
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
            <div className="min-w-[180px] pr-2">
              {t('healthcheck.config.notifyWhenUnavailable')}
            </div>

            <Switch
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
            <div className="min-w-[180px] pr-2">{t('healthcheck.config.checkIntervalMinutes')}</div>

            <Input
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
            <div className="min-w-[180px] pr-2">{t('healthcheck.config.attemptsBeforeDown')}</div>

            <Input
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
            <div className="min-w-[180px] pr-2">{t('healthcheck.config.attemptRetentionDays')}</div>

            <Input
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
