import { Spin } from 'antd';
import { useEffect, useState } from 'react';
import { useTranslation } from 'react-i18next';

import { healthcheckConfigApi } from '../../../../entity/healthcheck';
import type { HealthcheckConfig } from '../../../../entity/healthcheck';
import { translateApiError } from '../../../../shared/i18n';

interface Props {
  databaseId: string;
}

export const ShowHealthcheckConfigComponent = ({ databaseId }: Props) => {
  const { t } = useTranslation();
  const [isLoading, setIsLoading] = useState(false);
  const [healthcheckConfig, setHealthcheckConfig] = useState<HealthcheckConfig | undefined>(
    undefined,
  );

  useEffect(() => {
    setIsLoading(true);
    healthcheckConfigApi
      .getHealthcheckConfig(databaseId)
      .then((config) => {
        setHealthcheckConfig(config);
      })
      .catch((error) => {
        alert(translateApiError(error, t));
      })
      .finally(() => {
        setIsLoading(false);
      });
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
