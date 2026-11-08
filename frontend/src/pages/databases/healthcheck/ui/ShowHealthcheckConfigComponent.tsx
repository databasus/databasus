import { Loader } from '@mantine/core';
import { useTranslation } from 'react-i18next';

import { healthcheckConfigApi } from '@/entities/healthcheck';
import { useSessionQuery } from '@/shared/api';
import { LoadFailure } from '@/shared/ui';

interface Props {
  databaseId: string;
}

export const ShowHealthcheckConfigComponent = ({ databaseId }: Props) => {
  const { t } = useTranslation();

  const query = useSessionQuery({
    queryKey: ['healthcheck-config', databaseId],
    queryFn: (signal) => healthcheckConfigApi.getHealthcheckConfig(databaseId, { signal }),
  });
  const healthcheckConfig = query.data;
  const loadError = query.error ?? undefined;

  if (loadError !== undefined) {
    return <LoadFailure error={loadError} onRetry={() => void query.refetch()} />;
  }

  if (!healthcheckConfig) {
    return <Loader size="xs" role="status" />;
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
