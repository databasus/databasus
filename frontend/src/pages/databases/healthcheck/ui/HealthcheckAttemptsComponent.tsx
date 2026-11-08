import { Loader, Tooltip } from '@mantine/core';
import dayjs from 'dayjs';
import { useId, useState } from 'react';
import { useTranslation } from 'react-i18next';

import { type Database, HealthStatus } from '@/entities/databases';
import { healthcheckAttemptApi, healthcheckConfigApi } from '@/entities/healthcheck';
import { useFixedQueryPolling, useSessionQuery } from '@/shared/api';
import { useLocale } from '@/shared/i18n';
import { collectCursorPages } from '@/shared/lib/collectCursorPages';
import { getUserShortTimeFormat } from '@/shared/time/getUserTimeFormat';
import { WrappingSelect, LoadFailure } from '@/shared/ui';

interface Props {
  database: Database;
}

const getSinceByPeriod = (period: 'today' | '7d' | '30d' | 'all'): Date => {
  const since = new Date();

  if (period === 'today') {
    return new Date(since.setDate(since.getDate() - 1));
  }

  if (period === '7d') {
    return new Date(since.setDate(since.getDate() - 7));
  }

  if (period === '30d') {
    return new Date(since.setDate(since.getDate() - 30));
  }

  if (period === 'all') {
    return new Date(0);
  }

  return since;
};

export const HealthcheckAttemptsComponent = ({ database }: Props) => {
  const { t } = useTranslation();
  const { formatRelativeTime } = useLocale();
  const periodFieldId = useId();

  const [period, setPeriod] = useState<'today' | '7d' | '30d' | 'all'>('today');
  const configQuery = useSessionQuery({
    queryKey: ['healthcheck-config', database.id, period],
    queryFn: (signal) => healthcheckConfigApi.getHealthcheckConfig(database.id, { signal }),
  });
  const attemptsQuery = useSessionQuery({
    queryKey: ['healthcheck-attempts', database.workspaceId, database.id, period],
    enabled: configQuery.data?.isHealthcheckEnabled === true,
    queryFn: (signal) => {
      const since = getSinceByPeriod(period);

      return collectCursorPages(
        (after: string | undefined) =>
          healthcheckAttemptApi.getAttemptsByDatabase({
            databaseId: database.id,
            since,
            after,
            signal,
          }),
        { signal },
      );
    },
  });
  useFixedQueryPolling({
    query: attemptsQuery,
    intervalMs: 60000,
    isEnabled:
      configQuery.data?.isHealthcheckEnabled === true &&
      period === 'today' &&
      attemptsQuery.data !== undefined,
    shouldRunInBackground: true,
  });
  const isHealthcheckConfigLoading = configQuery.isPending;
  const isShowHealthcheckConfig = configQuery.data?.isHealthcheckEnabled;
  const configLoadError = configQuery.error ?? undefined;
  const healthcheckAttempts = attemptsQuery.data ?? [];
  const isLoading = attemptsQuery.isPending;
  const attemptsLoadError =
    attemptsQuery.data === undefined ? (attemptsQuery.error ?? undefined) : undefined;
  const retryLoad = async () => {
    const config = await configQuery.refetch();
    if (config.data?.isHealthcheckEnabled && attemptsQuery.error !== null) {
      await attemptsQuery.refetch();
    }
  };

  if (isHealthcheckConfigLoading) {
    return (
      <div className="mb-5 flex items-center">
        <Loader role="status" />
      </div>
    );
  }

  if (configLoadError !== undefined) {
    return <LoadFailure className="mb-5" error={configLoadError} onRetry={retryLoad} />;
  }

  if (!isShowHealthcheckConfig) {
    return null;
  }

  return (
    <div className="mb-5 w-full rounded-md bg-surface p-3 shadow sm:p-5">
      <h2 className="text-lg font-bold sm:text-xl">{t('healthcheck.attempts.title')}</h2>

      <div className="mt-3 flex flex-col gap-2 sm:mt-4 sm:flex-row sm:items-center">
        <label className="text-sm font-medium sm:mr-2" htmlFor={periodFieldId}>
          {t('healthcheck.attempts.period')}
        </label>
        <WrappingSelect
          id={periodFieldId}
          size="xs"
          value={period}
          onChange={(value) => setPeriod(value)}
          className="w-full sm:w-[120px]"
          data={[
            { value: 'today', label: t('healthcheck.attempts.periods.today') },
            { value: '7d', label: t('healthcheck.attempts.periods.sevenDays') },
            { value: '30d', label: t('healthcheck.attempts.periods.thirtyDays') },
            { value: 'all', label: t('healthcheck.attempts.periods.allTime') },
          ]}
        />
      </div>

      <div className="mt-4 sm:mt-5" />

      {attemptsLoadError !== undefined ? (
        <LoadFailure error={attemptsLoadError} onRetry={retryLoad} />
      ) : isLoading ? (
        <div className="flex justify-center">
          <Loader size="xs" role="status" />
        </div>
      ) : (
        <div className="flex flex-wrap gap-1">
          {healthcheckAttempts.length > 0 ? (
            healthcheckAttempts.map((healthcheckAttempt) => (
              <Tooltip
                key={healthcheckAttempt.id}
                label={`${dayjs(healthcheckAttempt.createdAt).format(getUserShortTimeFormat().format)} (${formatRelativeTime(healthcheckAttempt.createdAt)})`}
              >
                <div
                  data-status={healthcheckAttempt.status}
                  className={`h-[8px] w-[8px] cursor-pointer rounded-[2px] ${
                    healthcheckAttempt.status === HealthStatus.AVAILABLE
                      ? 'bg-success-fill'
                      : 'bg-danger'
                  }`}
                />
              </Tooltip>
            ))
          ) : (
            <div className="text-xs text-muted">{t('healthcheck.attempts.empty')}</div>
          )}
        </div>
      )}
    </div>
  );
};
