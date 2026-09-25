import { Select, Spin } from 'antd';
import { useEffect, useState } from 'react';
import { useTranslation } from 'react-i18next';

import type { Database } from '../../../entity/databases';
import {
  type HealthcheckAttempt,
  HealthcheckAttemptsStripComponent,
  healthcheckAttemptApi,
  healthcheckConfigApi,
} from '../../../entity/healthcheck';
import { translateApiError } from '../../../shared/i18n';

interface Props {
  database: Database;
  onVisibilityChange?: (isVisible: boolean) => void;
}

let lastLoadTime = 0;

const getAfterDateByPeriod = (period: 'today' | '7d' | '30d' | 'all'): Date => {
  const afterDate = new Date();

  if (period === 'today') {
    return new Date(afterDate.setDate(afterDate.getDate() - 1));
  }

  if (period === '7d') {
    return new Date(afterDate.setDate(afterDate.getDate() - 7));
  }

  if (period === '30d') {
    return new Date(afterDate.setDate(afterDate.getDate() - 30));
  }

  if (period === 'all') {
    return new Date(0);
  }

  return afterDate;
};

export const HealthckeckAttemptsComponent = ({ database, onVisibilityChange }: Props) => {
  const { t } = useTranslation();

  const [isHealthcheckConfigLoading, setIsHealthcheckConfigLoading] = useState(false);
  const [isShowHealthcheckConfig, setIsShowHealthcheckConfig] = useState(false);

  const [isLoading, setIsLoading] = useState(false);
  const [healthcheckAttempts, setHealthcheckAttempts] = useState<HealthcheckAttempt[]>([]);
  const [period, setPeriod] = useState<'today' | '7d' | '30d' | 'all'>('today');

  const loadHealthcheckAttempts = async (isShowLoading = true) => {
    if (isShowLoading) {
      setIsLoading(true);
    }

    try {
      const currentTime = Date.now();
      lastLoadTime = currentTime;

      const afterDate = getAfterDateByPeriod(period);

      const healthcheckAttempts = await healthcheckAttemptApi.getAttemptsByDatabase(
        database.id,
        afterDate,
      );

      if (currentTime != lastLoadTime) {
        return;
      }

      setHealthcheckAttempts(healthcheckAttempts);
    } catch (e) {
      alert(translateApiError(e, t));
    }

    if (isShowLoading) {
      setIsLoading(false);
    }
  };

  useEffect(() => {
    let interval: number | null = null;
    let isCancelled = false;

    setIsHealthcheckConfigLoading(true);
    healthcheckConfigApi.getHealthcheckConfig(database.id).then((healthcheckConfig) => {
      if (isCancelled) return;

      setIsHealthcheckConfigLoading(false);

      if (!healthcheckConfig.isHealthcheckEnabled) {
        onVisibilityChange?.(false);
      }

      if (healthcheckConfig.isHealthcheckEnabled) {
        setIsShowHealthcheckConfig(true);
        onVisibilityChange?.(true);
        loadHealthcheckAttempts();

        // Set up interval only if healthcheck
        // is enabled and period is 'today'
        if (period === 'today') {
          interval = setInterval(() => {
            loadHealthcheckAttempts(false);
          }, 60_000);
        }
      }
    });

    return () => {
      isCancelled = true;
      if (interval) {
        clearInterval(interval);
      }
    };
  }, [database.id, period]);

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
        <>
          {healthcheckAttempts.length > 0 ? (
            <HealthcheckAttemptsStripComponent attempts={healthcheckAttempts} />
          ) : (
            <div className="text-xs text-gray-400">{t('healthcheck.attempts.empty')}</div>
          )}
        </>
      )}
    </div>
  );
};
