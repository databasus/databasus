import { InfoCircleOutlined } from '@ant-design/icons';
import { Select, Tooltip } from 'antd';
import { useId } from 'react';
import { useTranslation } from 'react-i18next';

import type { LogicalBackupSettings } from '@/entities/backup-configs';
import { changeLogicalRetentionPolicy } from '@/pages/databases/model/changeLogicalRetentionPolicy';
import { isGfsHoursShown } from '@/pages/databases/model/isGfsHoursShown';
import { LogicalRetentionPolicy } from '@/pages/databases/model/LogicalRetentionPolicy';
import { LOGICAL_RETENTION_POLICY_LABEL_KEYS } from '@/pages/databases/model/LogicalRetentionPolicyLabelKeys';
import { RetentionPeriod } from '@/pages/databases/model/RetentionPeriod';
import { RETENTION_PERIOD_LABEL_KEYS } from '@/pages/databases/model/RetentionPeriodLabelKeys';
import { GfsRetentionHint } from '@/pages/databases/ui/backup-config/GfsRetentionHint';
import { RetentionNumberInput } from '@/pages/databases/ui/backup-config/RetentionNumberInput';

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
      <div className="mt-1 mb-1 min-w-[150px] sm:mb-0 sm:pr-2">
        {t('backups.logical.config.retentionPolicy')}
      </div>
      <div className="flex min-w-0 grow flex-col gap-1">
        <Select
          value={retention.policy}
          options={retentionPolicyOptions}
          size="small"
          className="w-[200px]"
          popupMatchSelectWidth={false}
          aria-label={t('backups.logical.config.retentionPolicy')}
          onChange={(policy) => onChange(changeLogicalRetentionPolicy(settings, policy))}
        />

        {retention.policy === LogicalRetentionPolicy.TIME_PERIOD && (
          <div className="flex w-full items-center">
            <Select
              value={retention.period}
              aria-label={t('healthcheck.attempts.period')}
              onChange={(period) => updateRetention({ period })}
              size="small"
              className="min-w-0 grow"
              options={periodOptions}
            />

            <Tooltip
              className="cursor-pointer"
              title={t('backups.logical.config.retention.timePeriodTooltip')}
            >
              <InfoCircleOutlined className="ml-2 text-muted" />
            </Tooltip>
          </div>
        )}

        {retention.policy === LogicalRetentionPolicy.COUNT && (
          <div className="flex items-center">
            <label htmlFor={countInputId} className="mr-2 shrink-0 text-sm text-muted">
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
              title={t('backups.logical.config.retention.countTooltip')}
            >
              <InfoCircleOutlined className="ml-2 text-muted" />
            </Tooltip>
          </div>
        )}

        {retention.policy === LogicalRetentionPolicy.GFS && (
          <>
            <GfsRetentionHint />

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
