import { Select, Spin, Tooltip } from 'antd';
import dayjs from 'dayjs';
import { useEffect, useState } from 'react';
import { useTranslation } from 'react-i18next';

import { type Database, HealthStatus } from '../../../../entity/databases';
import {
  type HealthcheckAttempt,
  healthcheckAttemptApi,
  healthcheckConfigApi,
} from '../../../../entity/healthcheck';
import { translateApiError, useLocale } from '../../../../shared/i18n';
import { getUserShortTimeFormat } from '../../../../shared/time/getUserTimeFormat';

interface Props {
  database: Database;
  onVisibilityChange?: (isVisible: boolean) => void;
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

export const HealthcheckAttemptsComponent = ({ database, onVisibilityChange }: Props) => {
  const { t } = useTranslation();
  const { formatRelativeTime } = useLocale();

  const [isHealthcheckConfigLoading, setIsHealthcheckConfigLoading] = useState(false);
  const [isShowHealthcheckConfig, setIsShowHealthcheckConfig] = useState(false);

  const [isLoading, setIsLoading] = useState(false);
  const [healthcheckAttempts, setHealthcheckAttempts] = useState<HealthcheckAttempt[]>([]);
  const [period, setPeriod] = useState<'today' | '7d' | '30d' | 'all'>('today');

  useEffect(() => {
    let interval: ReturnType<typeof setInterval> | undefined;
    let isCancelled = false;
    let loadSequence = 0;

    const loadHealthcheckAttempts = async (shouldShowLoading = true) => {
      const sequence = ++loadSequence;
      if (shouldShowLoading) setIsLoading(true);

      try {
        const since = getSinceByPeriod(period);
        const attempts: HealthcheckAttempt[] = [];
        let after: string | undefined;
        do {
          const page = await healthcheckAttemptApi.getAttemptsByDatabase(database.id, since, after);
          if (isCancelled || sequence !== loadSequence) return;
          attempts.push(...page.items);
          after = page.nextCursor;
        } while (after);
        setHealthcheckAttempts(attempts);
      } catch (e) {
        if (!isCancelled && sequence === loadSequence) alert(translateApiError(e, t));
      } finally {
        if (!isCancelled && sequence === loadSequence) setIsLoading(false);
      }
    };

    setIsHealthcheckConfigLoading(true);
    setIsShowHealthcheckConfig(false);
    healthcheckConfigApi
      .getHealthcheckConfig(database.id)
      .then((config) => {
        if (isCancelled) return;
        setIsShowHealthcheckConfig(config.isHealthcheckEnabled);
        onVisibilityChange?.(config.isHealthcheckEnabled);
        if (config.isHealthcheckEnabled) {
          void loadHealthcheckAttempts();
          if (period === 'today') {
            interval = setInterval(() => {
              void loadHealthcheckAttempts(false);
            }, 60_000);
          }
        }
      })
      .catch((e) => {
        if (!isCancelled) alert(translateApiError(e, t));
      })
      .finally(() => {
        if (!isCancelled) setIsHealthcheckConfigLoading(false);
      });

    return () => {
      isCancelled = true;
      if (interval) clearInterval(interval);
    };
  }, [database.id, period, onVisibilityChange, t]);

  if (isHealthcheckConfigLoading) {
    return (
      <div className="mb-5 flex items-center">
        <Spin />
      </div>
    );
  }

  if (!isShowHealthcheckConfig) {
    return null;
  }

  return (
    <div className="mb-5 w-full rounded-tr-md rounded-br-md rounded-bl-md bg-white p-3 shadow sm:p-5 dark:bg-gray-800">
      <h2 className="text-lg font-bold sm:text-xl">{t('healthcheck.attempts.title')}</h2>

      <div className="mt-3 flex flex-col gap-2 sm:mt-4 sm:flex-row sm:items-center">
        <span className="text-sm font-medium sm:mr-2">{t('healthcheck.attempts.period')}</span>
        <Select
          size="small"
          value={period}
          onChange={(value) => setPeriod(value)}
          className="w-full sm:w-[120px]"
          options={[
            { value: 'today', label: t('healthcheck.attempts.periods.today') },
            { value: '7d', label: t('healthcheck.attempts.periods.sevenDays') },
            { value: '30d', label: t('healthcheck.attempts.periods.thirtyDays') },
            { value: 'all', label: t('healthcheck.attempts.periods.allTime') },
          ]}
        />
      </div>

      <div className="mt-4 sm:mt-5" />

      {isLoading ? (
        <div className="flex justify-center">
          <Spin size="small" />
        </div>
      ) : (
        <div className="flex flex-wrap gap-1">
          {healthcheckAttempts.length > 0 ? (
            healthcheckAttempts.map((healthcheckAttempt) => (
              <Tooltip
                key={healthcheckAttempt.id}
                title={`${dayjs(healthcheckAttempt.createdAt).format(getUserShortTimeFormat().format)} (${formatRelativeTime(healthcheckAttempt.createdAt)})`}
              >
                <div
                  data-status={healthcheckAttempt.status}
                  className={`h-[8px] w-[8px] cursor-pointer rounded-[2px] ${
                    healthcheckAttempt.status === HealthStatus.AVAILABLE
                      ? 'bg-green-500'
                      : 'bg-red-500'
                  }`}
                />
              </Tooltip>
            ))
          ) : (
            <div className="text-xs text-gray-400">{t('healthcheck.attempts.empty')}</div>
          )}
        </div>
      )}
    </div>
  );
};
