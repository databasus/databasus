import { Select, Spin, Tooltip } from 'antd';
import dayjs from 'dayjs';
import { useEffect, useId, useState } from 'react';
import { useTranslation } from 'react-i18next';

import { type Database, HealthStatus } from '../../../../entities/databases';
import {
  type HealthcheckAttempt,
  healthcheckAttemptApi,
  healthcheckConfigApi,
} from '../../../../entities/healthcheck';
import { useSessionGuard } from '../../../../shared/auth';
import { useLocale } from '../../../../shared/i18n';
import { getUserShortTimeFormat } from '../../../../shared/time/getUserTimeFormat';
import { LoadFailure } from '../../../../shared/ui';

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
  const isSessionActive = useSessionGuard();
  const periodFieldId = useId();

  const [isHealthcheckConfigLoading, setIsHealthcheckConfigLoading] = useState(false);
  const [isShowHealthcheckConfig, setIsShowHealthcheckConfig] = useState(false);
  const [configLoadError, setConfigLoadError] = useState<unknown>();

  const [isLoading, setIsLoading] = useState(false);
  const [healthcheckAttempts, setHealthcheckAttempts] = useState<HealthcheckAttempt[]>([]);
  const [attemptsLoadError, setAttemptsLoadError] = useState<unknown>();
  const [period, setPeriod] = useState<'today' | '7d' | '30d' | 'all'>('today');
  const [loadAttempt, setLoadAttempt] = useState(0);

  const retryLoad = () => setLoadAttempt((attempt) => attempt + 1);

  useEffect(() => {
    let interval: ReturnType<typeof setInterval> | undefined;
    let isCancelled = false;
    let loadSequence = 0;

    const isCurrentLoad = (sequence: number) =>
      !isCancelled && sequence === loadSequence && isSessionActive();

    const fetchAllAttempts = async (sequence: number) => {
      const since = getSinceByPeriod(period);
      const attempts: HealthcheckAttempt[] = [];
      let after: string | undefined;

      do {
        const page = await healthcheckAttemptApi.getAttemptsByDatabase(database.id, since, after);
        if (!isCurrentLoad(sequence)) return undefined;

        attempts.push(...page.items);
        after = page.nextCursor;
      } while (after);

      return attempts;
    };

    const refreshHealthcheckAttempts = async () => {
      const sequence = ++loadSequence;

      try {
        const attempts = await fetchAllAttempts(sequence);
        if (attempts) setHealthcheckAttempts(attempts);
      } catch {
        // A failed refresh keeps the shown attempts, and the next tick repairs them on its own.
      }
    };

    const loadHealthcheckAttempts = async () => {
      const sequence = ++loadSequence;
      setIsLoading(true);

      try {
        const attempts = await fetchAllAttempts(sequence);
        if (!attempts) return;

        setHealthcheckAttempts(attempts);

        // Polling starts only after a successful load, so a tick cannot replace a shown failure.
        if (period === 'today') {
          interval = setInterval(() => {
            void refreshHealthcheckAttempts();
          }, 60_000);
        }
      } catch (e) {
        if (isCurrentLoad(sequence)) setAttemptsLoadError(e);
      } finally {
        if (isCurrentLoad(sequence)) setIsLoading(false);
      }
    };

    setIsHealthcheckConfigLoading(true);
    setIsShowHealthcheckConfig(false);
    setConfigLoadError(undefined);
    setAttemptsLoadError(undefined);
    setHealthcheckAttempts([]);

    healthcheckConfigApi
      .getHealthcheckConfig(database.id)
      .then((config) => {
        if (isCancelled || !isSessionActive()) return;

        setIsShowHealthcheckConfig(config.isHealthcheckEnabled);

        if (config.isHealthcheckEnabled) void loadHealthcheckAttempts();
      })
      .catch((e: unknown) => {
        if (isCancelled || !isSessionActive()) return;

        setConfigLoadError(e);
      })
      .finally(() => {
        if (!isCancelled && isSessionActive()) setIsHealthcheckConfigLoading(false);
      });

    return () => {
      isCancelled = true;
      if (interval) clearInterval(interval);
    };
  }, [database.id, period, loadAttempt, isSessionActive]);

  if (isHealthcheckConfigLoading) {
    return (
      <div className="mb-5 flex items-center">
        <Spin />
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
        <Select
          id={periodFieldId}
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

      {attemptsLoadError !== undefined ? (
        <LoadFailure error={attemptsLoadError} onRetry={retryLoad} />
      ) : isLoading ? (
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
