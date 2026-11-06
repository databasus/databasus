import { InfoCircleOutlined } from '@ant-design/icons';
import { Select, Tooltip } from 'antd';
import { useId } from 'react';
import { useTranslation } from 'react-i18next';

import type { PhysicalDatabaseBackupType } from '@/entities/databases';
import type { FullBackupsRetention } from '@/pages/databases/model/physical-backups/FullBackupsRetention';
import { FullBackupsRetentionPolicy } from '@/pages/databases/model/physical-backups/FullBackupsRetentionPolicy';
import { FULL_BACKUPS_RETENTION_POLICY_LABEL_KEYS } from '@/pages/databases/model/physical-backups/FullBackupsRetentionPolicyLabelKeys';
import { hasIncrementalBackups } from '@/pages/databases/model/physical-backups/hasIncrementalBackups';
import type { PhysicalBackupConfigDraft } from '@/pages/databases/model/physical-backups/PhysicalBackupConfigDraft';
import { PhysicalRetentionPolicy } from '@/pages/databases/model/physical-backups/PhysicalRetentionPolicy';
import { GfsRetentionHint } from '@/pages/databases/ui/backup-config/GfsRetentionHint';
import { RetentionNumberInput } from '@/pages/databases/ui/backup-config/RetentionNumberInput';
import type { TranslationKey } from '@/shared/i18n';

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
      <label htmlFor={inputId} className="w-[110px] shrink-0 text-sm leading-4 text-muted">
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
      <div className="mt-1 mb-1 min-w-[150px] sm:mb-0 sm:pr-2">
        {t('backups.physical.config.retention')}
      </div>
      <div className="flex min-w-0 grow flex-col gap-2">
        {hasIncrementalBackups(backupType) ? (
          <div className="flex w-full items-center">
            <Select
              value={policy}
              onChange={(chosenPolicy) => onChange({ ...retention, policy: chosenPolicy })}
              size="small"
              className="min-w-0 grow"
              popupMatchSelectWidth={false}
              aria-label={t('backups.physical.config.retention')}
              options={[
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
              title={
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
              <InfoCircleOutlined className="ml-2 text-muted" />
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
                title={t('backups.physical.config.chainsCountTooltip')}
              >
                <InfoCircleOutlined className="ml-1 text-muted" />
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
              <span className="w-[110px] shrink-0 text-sm leading-4 text-muted">
                {t('backups.physical.config.fullBackups')}
              </span>
              <Select
                value={fullBackups.policy}
                onChange={(fullBackupsPolicy) => updateFullBackups({ policy: fullBackupsPolicy })}
                size="small"
                className="w-[80px] max-w-[80px]"
                popupMatchSelectWidth={false}
                aria-label={t('backups.physical.config.fullBackups')}
                options={fullBackupsPolicyOptions}
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
                <GfsRetentionHint />

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
