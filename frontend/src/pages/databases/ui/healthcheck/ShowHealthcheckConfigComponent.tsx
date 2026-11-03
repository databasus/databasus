import { Spin } from 'antd';
import { useEffect, useState } from 'react';
import { useTranslation } from 'react-i18next';

import { healthcheckConfigApi } from '../../../../entities/healthcheck';
import type { HealthcheckConfig } from '../../../../entities/healthcheck';
import { useSessionGuard } from '../../../../shared/auth';
import { LoadFailure } from '../../../../shared/ui';

interface Props {
  databaseId: string;
}

export const ShowHealthcheckConfigComponent = ({ databaseId }: Props) => {
  const { t } = useTranslation();
  const isActive = useSessionGuard();

  const [healthcheckConfig, setHealthcheckConfig] = useState<HealthcheckConfig | undefined>(
    undefined,
  );
  const [loadError, setLoadError] = useState<unknown>();
  const [loadAttempt, setLoadAttempt] = useState(0);

  useEffect(() => {
    let isCurrentLoad = true;

    setHealthcheckConfig(undefined);
    setLoadError(undefined);

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
      <LoadFailure error={loadError} onRetry={() => setLoadAttempt((attempt) => attempt + 1)} />
    );
  }

  if (!healthcheckConfig) {
    return <Spin size="small" />;
  }

  return (
    <div className="space-y-4">
      <div className="mb-1 flex items-center">
        <div className="min-w-[180px] pr-2">{t('healthcheck.config.isEnabled')}</div>
        <div>
          {healthcheckConfig.isHealthcheckEnabled
            ? t('common.answers.yes')
            : t('common.answers.no')}
        </div>
      </div>

      {healthcheckConfig.isHealthcheckEnabled && (
        <>
          <div className="mb-1 flex items-center">
            <div className="min-w-[180px] pr-2">
              {t('healthcheck.config.notifyWhenUnavailable')}
            </div>
            <div className="lg:w-[200px]">
              {healthcheckConfig.isNotificationEnabled
                ? t('common.answers.yes')
                : t('common.answers.no')}
            </div>
          </div>

          <div className="mb-1 flex items-center">
            <div className="min-w-[180px] pr-2">{t('healthcheck.config.checkIntervalMinutes')}</div>
            <div className="lg:w-[200px]">{healthcheckConfig.intervalMinutes}</div>
          </div>

          <div className="mb-1 flex items-center">
            <div className="min-w-[180px] pr-2">{t('healthcheck.config.attemptsBeforeDown')}</div>
            <div className="lg:w-[200px]">{healthcheckConfig.failedAttemptsBeforeUnavailable}</div>
          </div>

          <div className="mb-1 flex items-center">
            <div className="min-w-[180px] pr-2">{t('healthcheck.config.attemptRetentionDays')}</div>
            <div className="lg:w-[200px]">{healthcheckConfig.attemptRetentionDays}</div>
          </div>
        </>
      )}
    </div>
  );
};
