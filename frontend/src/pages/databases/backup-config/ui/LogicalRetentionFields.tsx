import { Tooltip } from '@mantine/core';
import { Info } from 'lucide-react';
import { useId } from 'react';
import { useTranslation } from 'react-i18next';

import type { LogicalBackupSettings } from '@/entities/backup-configs';
import { changeLogicalRetentionPolicy } from '@/pages/databases/backup-config/model/changeLogicalRetentionPolicy';
import { isGfsHoursShown } from '@/pages/databases/backup-config/model/isGfsHoursShown';
import {
  LogicalRetentionPolicy,
  LOGICAL_RETENTION_POLICY_LABEL_KEYS,
} from '@/pages/databases/backup-config/model/LogicalRetentionPolicy';
import {
  RetentionPeriod,
  RETENTION_PERIOD_LABEL_KEYS,
} from '@/pages/databases/backup-config/model/RetentionPeriod';
import { GfsRetentionHint } from '@/pages/databases/backup-config/ui/GfsRetentionHint';
import { RetentionNumberInput } from '@/pages/databases/backup-config/ui/RetentionNumberInput';
import { WrappingSelect } from '@/shared/ui';

type LogicalRetention = LogicalBackupSettings['retention'];

interface Props {
  settings: LogicalBackupSettings;
  onChange: (retention: LogicalRetention) => void;
}

interface GfsPeriodFieldProps {
  label: string;
  value: number;
  onChange: (value: number) => void;
}

const RETENTION_POLICY_ORDER = [
  LogicalRetentionPolicy.GFS,
  LogicalRetentionPolicy.TIME_PERIOD,
  LogicalRetentionPolicy.COUNT,
];

const GfsPeriodField = ({ label, value, onChange }: GfsPeriodFieldProps) => {
  const inputId = useId();

  return (
    <div className="flex items-center gap-2">
      <label htmlFor={inputId} className="w-[110px] text-sm text-muted">
        {label}
      </label>
      <RetentionNumberInput id={inputId} minimum={0} value={value} onChange={onChange} />
    </div>
  );
};

export const LogicalRetentionFields = ({ settings, onChange }: Props) => {
  const { t } = useTranslation();
  const countInputId = useId();

  const { retention, schedule } = settings;

  const updateRetention = (patch: Partial<LogicalRetention>) =>
    onChange({ ...retention, ...patch });

  const retentionPolicyOptions = RETENTION_POLICY_ORDER.map((policy) => ({
    value: policy,
    label: t(LOGICAL_RETENTION_POLICY_LABEL_KEYS[policy]),
  }));

  const periodOptions = Object.values(RetentionPeriod).map((period) => ({
    value: period,
    label: t(RETENTION_PERIOD_LABEL_KEYS[period]),
  }));

  return (
    <div className="mt-5 mb-1 flex w-full flex-col items-start sm:flex-row sm:items-start">
      <div className="mt-1 mb-1 min-w-0 wrap-anywhere sm:mb-0 sm:w-[150px] sm:shrink-0 sm:pr-2">
        {t('backups.logical.config.retentionPolicy')}
      </div>
      <div className="flex w-full min-w-0 grow flex-col gap-1 sm:w-auto">
        <WrappingSelect
          value={retention.policy}
          data={retentionPolicyOptions}
          size="xs"
          className="w-full max-w-[200px]"
          aria-label={t('backups.logical.config.retentionPolicy')}
          onChange={(policy) => onChange(changeLogicalRetentionPolicy(settings, policy))}
        />

        {retention.policy === LogicalRetentionPolicy.TIME_PERIOD && (
          <div className="flex w-full min-w-0 items-center">
            <WrappingSelect
              value={retention.period}
              aria-label={t('healthcheck.attempts.period')}
              onChange={(period) => updateRetention({ period })}
              size="xs"
              className="w-full max-w-[200px]"
              data={periodOptions}
            />

            <Tooltip
              className="cursor-pointer"
              label={t('backups.logical.config.retention.timePeriodTooltip')}
            >
              <Info className="ml-2 inline-block shrink-0 align-middle text-muted" size="1em" />
            </Tooltip>
          </div>
        )}

        {retention.policy === LogicalRetentionPolicy.COUNT && (
          <div className="flex min-w-0 flex-wrap items-center gap-y-1">
            <label
              htmlFor={countInputId}
              className="mr-2 min-w-0 basis-full text-sm wrap-anywhere text-muted sm:basis-auto"
            >
              {t('backups.logical.config.retention.count')}
            </label>
            <RetentionNumberInput
              id={countInputId}
              minimum={1}
              value={retention.count}
              onChange={(count) => updateRetention({ count })}
            />

            <Tooltip
              className="cursor-pointer"
              label={t('backups.logical.config.retention.countTooltip')}
            >
              <Info className="ml-2 inline-block shrink-0 align-middle text-muted" size="1em" />
            </Tooltip>
          </div>
        )}

        {retention.policy === LogicalRetentionPolicy.GFS && (
          <>
            <GfsRetentionHint backupMethod="LOGICAL" />

            <div className="flex flex-col gap-1">
              {isGfsHoursShown(schedule) && (
                <GfsPeriodField
                  label={t('backups.logical.config.retention.gfsFields.hourly')}
                  value={retention.gfsHours}
                  onChange={(gfsHours) => updateRetention({ gfsHours })}
                />
              )}

              <GfsPeriodField
                label={t('backups.logical.config.retention.gfsFields.daily')}
                value={retention.gfsDays}
                onChange={(gfsDays) => updateRetention({ gfsDays })}
              />

              <GfsPeriodField
                label={t('backups.logical.config.retention.gfsFields.weekly')}
                value={retention.gfsWeeks}
                onChange={(gfsWeeks) => updateRetention({ gfsWeeks })}
              />

              <GfsPeriodField
                label={t('backups.logical.config.retention.gfsFields.monthly')}
                value={retention.gfsMonths}
                onChange={(gfsMonths) => updateRetention({ gfsMonths })}
              />

              <GfsPeriodField
                label={t('backups.logical.config.retention.gfsFields.yearly')}
                value={retention.gfsYears}
                onChange={(gfsYears) => updateRetention({ gfsYears })}
              />
            </div>
          </>
        )}
      </div>
    </div>
  );
};
