import { Tooltip } from '@mantine/core';
import { Info } from 'lucide-react';
import { useId } from 'react';
import { useTranslation } from 'react-i18next';

import type { PhysicalDatabaseBackupType } from '@/entities/databases';
import {
  FullBackupsRetentionPolicy,
  FULL_BACKUPS_RETENTION_POLICY_LABEL_KEYS,
} from '@/pages/databases/backup-config/model/FullBackupsRetentionPolicy';
import { hasIncrementalBackups } from '@/pages/databases/backup-config/model/hasIncrementalBackups';
import type {
  FullBackupsRetention,
  PhysicalBackupConfigDraft,
} from '@/pages/databases/backup-config/model/PhysicalBackupConfigDraft';
import { PhysicalRetentionPolicy } from '@/pages/databases/backup-config/model/PhysicalRetentionPolicy';
import { GfsRetentionHint } from '@/pages/databases/backup-config/ui/GfsRetentionHint';
import { RetentionNumberInput } from '@/pages/databases/backup-config/ui/RetentionNumberInput';
import type { TranslationKey } from '@/shared/i18n';
import { WrappingSelect } from '@/shared/ui';

type PhysicalRetentionDraft = PhysicalBackupConfigDraft['settings']['retention'];

interface Props {
  retention: PhysicalRetentionDraft;
  backupType: PhysicalDatabaseBackupType;
  onChange: (retention: PhysicalRetentionDraft) => void;
}

interface NumberFieldProps {
  label: string;
  minimum: number;
  value: number;
  onChange: (value: number) => void;
}

const GFS_FIELDS: {
  field: 'gfsHours' | 'gfsDays' | 'gfsWeeks' | 'gfsMonths' | 'gfsYears';
  labelKey: TranslationKey;
}[] = [
  { field: 'gfsHours', labelKey: 'backups.physical.config.gfsFields.hourly' },
  { field: 'gfsDays', labelKey: 'backups.physical.config.gfsFields.daily' },
  { field: 'gfsWeeks', labelKey: 'backups.physical.config.gfsFields.weekly' },
  { field: 'gfsMonths', labelKey: 'backups.physical.config.gfsFields.monthly' },
  { field: 'gfsYears', labelKey: 'backups.physical.config.gfsFields.yearly' },
];

const NumberField = ({ label, minimum, value, onChange }: NumberFieldProps) => {
  const inputId = useId();

  return (
    <div className="flex items-center gap-2">
      <label htmlFor={inputId} className="w-[110px] shrink-0 text-sm leading-normal text-muted">
        {label}
      </label>
      <RetentionNumberInput id={inputId} minimum={minimum} value={value} onChange={onChange} />
    </div>
  );
};

export const PhysicalRetentionFields = ({ retention, backupType, onChange }: Props) => {
  const { t } = useTranslation();
  const chainsCountId = useId();

  const { policy, fullBackups } = retention;

  const updateFullBackups = (patch: Partial<FullBackupsRetention>) =>
    onChange({ ...retention, fullBackups: { ...fullBackups, ...patch } });

  const fullBackupsPolicyOptions = Object.values(FullBackupsRetentionPolicy).map((option) => ({
    value: option,
    label: t(FULL_BACKUPS_RETENTION_POLICY_LABEL_KEYS[option]),
  }));

  return (
    <div className="mt-5 mb-1 flex w-full flex-col items-start sm:flex-row sm:items-start">
      <div className="mt-1 mb-1 min-w-0 wrap-anywhere sm:mb-0 sm:w-[150px] sm:shrink-0 sm:pr-2">
        {t('backups.physical.config.retention')}
      </div>
      <div className="flex w-full min-w-0 grow flex-col gap-2 sm:w-auto">
        {hasIncrementalBackups(backupType) ? (
          <div className="flex w-full min-w-0 items-center">
            <WrappingSelect
              value={policy}
              onChange={(chosenPolicy) => onChange({ ...retention, policy: chosenPolicy })}
              size="xs"
              className="w-full max-w-[200px]"
              aria-label={t('backups.physical.config.retention')}
              data={[
                {
                  label: t('backups.physical.config.retentionOptions.chains'),
                  value: PhysicalRetentionPolicy.CHAINS,
                },
                {
                  label: t('backups.physical.config.retentionOptions.chainsAndFullBackups'),
                  value: PhysicalRetentionPolicy.CHAINS_AND_FULL_BACKUPS,
                },
              ]}
            />

            <Tooltip
              className="cursor-pointer"
              label={
                <div>
                  <div>{t('backups.physical.config.retentionHelp.chain')}</div>
                  <div className="mt-2 font-bold">
                    {t('backups.physical.config.retentionOptions.chains')}
                  </div>
                  <div>{t('backups.physical.config.retentionHelp.chains')}</div>
                  <div className="mt-2 font-bold">
                    {t('backups.physical.config.retentionOptions.chainsAndFullBackups')}
                  </div>
                  <div>{t('backups.physical.config.retentionHelp.chainsAndFullBackups')}</div>
                </div>
              }
            >
              <Info className="ml-2 inline-block shrink-0 align-middle text-muted" size="1em" />
            </Tooltip>
          </div>
        ) : (
          <div className="max-w-[320px] text-xs text-muted">
            {t('backups.physical.config.fullBackupsOnlyRetention')}
          </div>
        )}

        {policy !== PhysicalRetentionPolicy.FULL_BACKUPS && (
          <div className="flex items-center gap-2">
            <div className="flex w-[110px] shrink-0 items-center text-sm text-muted">
              <label htmlFor={chainsCountId}>{t('backups.physical.config.chainsCount')}</label>
              <Tooltip
                className="cursor-pointer"
                label={t('backups.physical.config.chainsCountTooltip')}
              >
                <Info className="ml-1 inline-block shrink-0 align-middle text-muted" size="1em" />
              </Tooltip>
            </div>
            <RetentionNumberInput
              id={chainsCountId}
              minimum={1}
              value={retention.chainsCount}
              onChange={(chainsCount) => onChange({ ...retention, chainsCount })}
            />
          </div>
        )}

        {policy !== PhysicalRetentionPolicy.CHAINS && (
          <div className="mt-1 flex flex-col gap-2">
            <div className="flex items-center gap-2">
              <span className="w-[110px] shrink-0 text-sm leading-normal text-muted">
                {t('backups.physical.config.fullBackups')}
              </span>
              <WrappingSelect
                value={fullBackups.policy}
                onChange={(fullBackupsPolicy) => updateFullBackups({ policy: fullBackupsPolicy })}
                size="xs"
                className="w-[80px] max-w-[80px]"
                aria-label={t('backups.physical.config.fullBackups')}
                data={fullBackupsPolicyOptions}
              />
            </div>

            {fullBackups.policy === FullBackupsRetentionPolicy.LAST_N && (
              <NumberField
                label={t('backups.physical.config.fullBackupsCount')}
                minimum={1}
                value={fullBackups.count}
                onChange={(count) => updateFullBackups({ count })}
              />
            )}

            {fullBackups.policy === FullBackupsRetentionPolicy.GFS && (
              <>
                <GfsRetentionHint backupMethod="PHYSICAL" />

                <div className="flex flex-col gap-1">
                  {GFS_FIELDS.map(({ field, labelKey }) => (
                    <NumberField
                      key={field}
                      label={t(labelKey)}
                      minimum={0}
                      value={fullBackups[field]}
                      onChange={(period) => updateFullBackups({ [field]: period })}
                    />
                  ))}
                </div>
              </>
            )}
          </div>
        )}
      </div>
    </div>
  );
};
