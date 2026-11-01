import { InfoCircleOutlined } from '@ant-design/icons';
import { Button, Checkbox, InputNumber, Select, Spin, Switch, Tooltip } from 'antd';
import { type JSX, useEffect, useState } from 'react';
import { useTranslation } from 'react-i18next';

import {
  BackupSettingsType,
  type PhysicalBackupConfigInput,
  type PhysicalBackupSettings,
  backupConfigApi,
  StorageSwitchHistoryComponent,
  type StorageSwitch,
  toPhysicalBackupConfigInput,
  BACKUP_ENCRYPTION_OPTION_LABEL_KEYS,
  BackupEncryption,
} from '../../../../entities/backup-configs';
import {
  type Database,
  DatabaseType,
  PhysicalDatabaseBackupType,
} from '../../../../entities/databases';
import {
  type Schedule,
  ScheduleEditor,
  ScheduleType,
  isScheduleComplete,
} from '../../../../entities/schedules';
import { useSessionGuard } from '../../../../shared/auth';
import { translateApiError } from '../../../../shared/i18n';
import { ConfirmationComponent } from '../../../../shared/ui';
import { type FullBackupsRetention } from '../../model/physical-backups/FullBackupsRetention';
import { FullBackupsRetentionPolicy } from '../../model/physical-backups/FullBackupsRetentionPolicy';
import { FULL_BACKUPS_RETENTION_POLICY_LABEL_KEYS } from '../../model/physical-backups/FullBackupsRetentionPolicyLabelKeys';
import { PhysicalNotificationTrigger } from '../../model/physical-backups/PhysicalNotificationTrigger';
import { PHYSICAL_NOTIFICATION_TRIGGER_LABEL_KEYS } from '../../model/physical-backups/PhysicalNotificationTriggerLabelKeys';
import { PhysicalRetentionPolicy } from '../../model/physical-backups/PhysicalRetentionPolicy';
import { BackupStorageSelection } from '../BackupStorageSelection';

interface Props {
  database: Database;

  // Seeds the editor in wizard mode (isSaveToApi=false) so going Back restores the
  // in-progress config instead of refetching a default by database id.
  initialConfig?: PhysicalBackupConfigInput;

  isShowBackButton: boolean;
  onBack: () => void;

  isShowCancelButton?: boolean;
  onCancel: () => void;

  saveButtonText?: string;
  isSaveToApi: boolean;
  onSaved: (backupConfig: PhysicalBackupConfigInput) => void;
}

const BYTES_IN_MB = 1024 * 1024;

const isFullBackupsRetentionValid = (retention: FullBackupsRetention): boolean => {
  if (retention.policy === FullBackupsRetentionPolicy.LAST_N) {
    return retention.count > 0;
  }

  return (
    retention.gfsHours > 0 ||
    retention.gfsDays > 0 ||
    retention.gfsWeeks > 0 ||
    retention.gfsMonths > 0 ||
    retention.gfsYears > 0
  );
};

const DEFAULT_FULL_BACKUPS_RETENTION: FullBackupsRetention = {
  policy: FullBackupsRetentionPolicy.GFS,
  count: 0,
  gfsHours: 0,
  gfsDays: 7,
  gfsWeeks: 4,
  gfsMonths: 12,
  gfsYears: 3,
};

// The contract forbids the retention part the selected policy doesn't use: CHAINS carries no full
// backups retention and FULL_BACKUPS keeps no chains. The editor keeps both filled so the values
// survive switching the policy, and drops the unused one only when saving.
const withRetentionForSave = (
  backupConfig: PhysicalBackupConfigInput,
  fullBackupsRetention: FullBackupsRetention,
): PhysicalBackupConfigInput => {
  const { policy, chainsCount } = backupConfig.settings.retention;

  const retention: PhysicalBackupSettings['retention'] = {
    policy,
    chainsCount: policy === PhysicalRetentionPolicy.FULL_BACKUPS ? 0 : chainsCount,
    fullBackups: policy === PhysicalRetentionPolicy.CHAINS ? null : fullBackupsRetention,
  };

  return { ...backupConfig, settings: { ...backupConfig.settings, retention } };
};

export const EditPhysicalBackupConfigComponent = ({
  database,
  initialConfig,

  isShowBackButton,
  onBack,

  isShowCancelButton,
  onCancel,
  saveButtonText,
  isSaveToApi,
  onSaved,
}: Props): JSX.Element => {
  const { t } = useTranslation();
  const isActive = useSessionGuard();
  const [backupConfig, setBackupConfig] = useState<PhysicalBackupConfigInput>();
  const [fullBackupsRetention, setFullBackupsRetention] = useState<FullBackupsRetention>(
    DEFAULT_FULL_BACKUPS_RETENTION,
  );
  const [isUnsaved, setIsUnsaved] = useState(false);
  const [isSaving, setIsSaving] = useState(false);
  const [isLoading, setIsLoading] = useState(true);

  const [savedStorageId, setSavedStorageId] = useState<string | null>(null);
  const [hasHistory, setHasHistory] = useState<boolean | undefined>(undefined);
  const [acceptedSwitch, setAcceptedSwitch] = useState<StorageSwitch>();
  const [isShowWarn, setIsShowWarn] = useState(false);

  // DB-level strategy lives on the database, not the config. It decides which
  // cadences and retention modes the config may use.
  const backupType =
    database.settings.type === DatabaseType.POSTGRES_PHYSICAL
      ? database.settings.backupType
      : PhysicalDatabaseBackupType.FULL;
  const isIncrementalAllowed = backupType !== PhysicalDatabaseBackupType.FULL;
  const isWalStream = backupType === PhysicalDatabaseBackupType.FULL_INCREMENTAL_WAL_STREAM;

  const updateBackupConfig = (patch: Partial<PhysicalBackupConfigInput>) => {
    setBackupConfig((prev) => (prev ? { ...prev, ...patch } : prev));
    setIsUnsaved(true);
  };

  const updateRetention = (patch: Partial<PhysicalBackupSettings['retention']>) => {
    setBackupConfig((prev) =>
      prev
        ? {
            ...prev,
            settings: { ...prev.settings, retention: { ...prev.settings.retention, ...patch } },
          }
        : prev,
    );
    setIsUnsaved(true);
  };

  const updateFullBackupSchedule = (fullBackupSchedule: Schedule) => {
    setBackupConfig((prev) =>
      prev ? { ...prev, settings: { ...prev.settings, fullBackupSchedule } } : prev,
    );
    setIsUnsaved(true);
  };

  const updateIncrementalBackupSchedule = (incrementalBackupSchedule: Schedule) => {
    setBackupConfig((prev) =>
      prev ? { ...prev, settings: { ...prev.settings, incrementalBackupSchedule } } : prev,
    );
    setIsUnsaved(true);
  };

  const updateFullBackupsRetention = (patch: Partial<FullBackupsRetention>) => {
    setFullBackupsRetention((prev) => ({ ...prev, ...patch }));
    setIsUnsaved(true);
  };

  const toggleNotificationTrigger = (trigger: PhysicalNotificationTrigger, isChecked: boolean) => {
    setBackupConfig((prev) => {
      if (!prev) return prev;

      const notificationTriggers = prev.settings.notificationTriggers.filter(
        (selectedTrigger) => selectedTrigger !== trigger,
      );
      if (isChecked) notificationTriggers.push(trigger);

      return { ...prev, settings: { ...prev.settings, notificationTriggers } };
    });
    setIsUnsaved(true);
  };

  const showBackupConfig = (shownBackupConfig: PhysicalBackupConfigInput) => {
    setBackupConfig(shownBackupConfig);
    setSavedStorageId(shownBackupConfig.storageId);
    setFullBackupsRetention(
      shownBackupConfig.settings.retention.fullBackups ?? DEFAULT_FULL_BACKUPS_RETENTION,
    );
  };

  const saveBackupConfig = async () => {
    if (!backupConfig) return;

    const configToSave = withRetentionForSave(backupConfig, fullBackupsRetention);

    if (isSaveToApi) {
      setIsSaving(true);
      try {
        const saved = await backupConfigApi.updateBackupConfig(database.id, configToSave);
        if (!isActive()) return;

        if ('targetStorageId' in saved) setAcceptedSwitch(saved);
        setSavedStorageId(configToSave.storageId);

        const hasSavedHistory = await backupConfigApi
          .hasBackupHistory(database.id, true)
          .catch(() => undefined);
        if (!isActive()) return;

        setHasHistory(hasSavedHistory);
        setIsUnsaved(false);
      } catch (e) {
        if (!isActive()) return;

        alert(translateApiError(e, t));
        setIsSaving(false);
        return;
      }
      setIsSaving(false);
    }

    onSaved(configToSave);
  };

  const confirmSaveBackupConfig = () => {
    if (
      isSaveToApi &&
      backupConfig?.storageId &&
      backupConfig.storageId !== savedStorageId &&
      (savedStorageId !== null || hasHistory !== false)
    ) {
      setIsShowWarn(true);
      return;
    }
    void saveBackupConfig();
  };

  const buildDefaultConfig = (): PhysicalBackupConfigInput => ({
    isBackupsEnabled: true,
    storageId: null,
    encryption: BackupEncryption.NONE,
    settings: {
      type: BackupSettingsType.PHYSICAL,
      fullBackupSchedule: { type: ScheduleType.WEEKLY, timeOfDay: '00:00', weekday: 1 },
      incrementalBackupSchedule: isIncrementalAllowed
        ? { type: ScheduleType.DAILY, timeOfDay: '00:00' }
        : null,
      retention: {
        policy:
          backupType === PhysicalDatabaseBackupType.FULL
            ? PhysicalRetentionPolicy.FULL_BACKUPS
            : PhysicalRetentionPolicy.CHAINS_AND_FULL_BACKUPS,
        chainsCount: 7,
        fullBackups: DEFAULT_FULL_BACKUPS_RETENTION,
      },
      walLagThresholdBytes: isWalStream ? 256 * BYTES_IN_MB : 0,
      notificationTriggers: [PhysicalNotificationTrigger.BACKUP_FAILED],
    },
  });

  useEffect(() => {
    let isCurrentLoad = true;

    const run = async () => {
      setIsLoading(true);

      try {
        // Wizard mode (isSaveToApi=false) is purely in-memory: never refetch by database
        // id, even after the DB row exists - seed from the in-progress config or defaults.
        if (isSaveToApi && database.id) {
          const loadedBackupConfig = toPhysicalBackupConfigInput(
            await backupConfigApi.getBackupConfig(database.id),
          );
          if (!isCurrentLoad) return;

          if (loadedBackupConfig) showBackupConfig(loadedBackupConfig);

          const hasLoadedHistory = await backupConfigApi.hasBackupHistory(database.id, true);
          if (!isCurrentLoad) return;

          setHasHistory(hasLoadedHistory);
          setIsUnsaved(false);
          setIsSaving(false);
        } else {
          showBackupConfig(initialConfig ?? buildDefaultConfig());
        }
      } catch (e) {
        if (isCurrentLoad) alert(translateApiError(e, t));
      } finally {
        if (isCurrentLoad) setIsLoading(false);
      }
    };

    run();

    return () => {
      isCurrentLoad = false;
    };
  }, [database.id]);

  if (isLoading) {
    return (
      <div className="mb-5 flex items-center">
        <Spin />
      </div>
    );
  }

  if (!backupConfig) return <div />;

  const { settings } = backupConfig;
  const retentionPolicy = settings.retention.policy;

  // FULL databases are forced to FULL_BACKUPS retention; the others may pick
  // between CHAINS and CHAINS_AND_FULL_BACKUPS.
  const isShowChainsCount =
    backupType !== PhysicalDatabaseBackupType.FULL &&
    (retentionPolicy === PhysicalRetentionPolicy.CHAINS ||
      retentionPolicy === PhysicalRetentionPolicy.CHAINS_AND_FULL_BACKUPS);

  const isShowFullBackupsEditor =
    backupType === PhysicalDatabaseBackupType.FULL ||
    retentionPolicy === PhysicalRetentionPolicy.CHAINS_AND_FULL_BACKUPS;

  const isChainsCountValid = !isShowChainsCount || settings.retention.chainsCount > 0;

  const isFullRetentionValid =
    !isShowFullBackupsEditor || isFullBackupsRetentionValid(fullBackupsRetention);

  const encryptionOptions = Object.values(BackupEncryption).map((encryption) => ({
    value: encryption,
    label: t(BACKUP_ENCRYPTION_OPTION_LABEL_KEYS[encryption]),
  }));

  const fullBackupsPolicyOptions = Object.values(FullBackupsRetentionPolicy).map((policy) => ({
    value: policy,
    label: t(FULL_BACKUPS_RETENTION_POLICY_LABEL_KEYS[policy]),
  }));

  const isAllFieldsFilled =
    !backupConfig.isBackupsEnabled ||
    ((isSaveToApi || Boolean(backupConfig.storageId)) &&
      Boolean(backupConfig.encryption) &&
      isScheduleComplete(settings.fullBackupSchedule) &&
      (!isIncrementalAllowed || isScheduleComplete(settings.incrementalBackupSchedule)) &&
      isChainsCountValid &&
      isFullRetentionValid);

  return (
    <div>
      {database.id && (
        <div className="mb-1 flex w-full flex-col items-start sm:flex-row sm:items-center">
          <div className="mb-1 min-w-[150px] sm:mb-0 sm:pr-2">
            {t('backups.config.backupsEnabled')}
          </div>
          <Switch
            checked={backupConfig.isBackupsEnabled}
            onChange={(checked) => updateBackupConfig({ isBackupsEnabled: checked })}
            size="small"
          />
        </div>
      )}

      {backupConfig.isBackupsEnabled && (
        <>
          <ScheduleEditor
            label={t('backups.physical.config.fullBackupCadence')}
            schedule={settings.fullBackupSchedule}
            onChange={updateFullBackupSchedule}
          />

          {isIncrementalAllowed && (
            <>
              <ScheduleEditor
                label={t('backups.physical.config.incrementalBackupCadence')}
                schedule={settings.incrementalBackupSchedule}
                onChange={updateIncrementalBackupSchedule}
              />
              <div className="mt-1 mb-3 flex w-full flex-col items-start sm:flex-row sm:items-center">
                <div className="min-w-[150px] pr-2" />
                <div className="max-w-[320px] text-xs text-gray-500 dark:text-gray-400">
                  {t('backups.physical.config.incrementalMoreFrequent')}
                </div>
              </div>
            </>
          )}

          <div className="mb-3" />
        </>
      )}

      <div className="mt-5 mb-1 flex w-full flex-col items-start sm:flex-row sm:items-center">
        <div className="mb-1 min-w-[150px] sm:mb-0 sm:pr-2">{t('backups.config.storage')}</div>
        <div className="flex w-full items-center">
          <BackupStorageSelection
            workspaceId={database.workspaceId}
            selectedStorageId={backupConfig.storageId ?? undefined}
            canClear={isSaveToApi}
            onSelect={(storageId) => updateBackupConfig({ storageId: storageId ?? null })}
          />
        </div>
      </div>

      <div className="mb-1 flex w-full flex-col items-start sm:flex-row sm:items-center">
        <div className="mb-1 min-w-[150px] sm:mb-0 sm:pr-2">{t('backups.config.encryption')}</div>
        <div className="flex w-full items-center">
          <Select
            value={backupConfig.encryption}
            onChange={(v) => updateBackupConfig({ encryption: v })}
            size="small"
            className="min-w-0 grow"
            options={encryptionOptions}
          />

          <Tooltip
            className="cursor-pointer"
            title={t('backups.physical.config.encryptionTooltip')}
          >
            <InfoCircleOutlined className="ml-2" style={{ color: 'gray' }} />
          </Tooltip>
        </div>
      </div>

      <div className="mt-5 mb-1 flex w-full flex-col items-start sm:flex-row sm:items-start">
        <div className="mt-1 mb-1 min-w-[150px] sm:mb-0 sm:pr-2">
          {t('backups.physical.config.retention')}
        </div>
        <div className="flex min-w-0 grow flex-col gap-2">
          {backupType === PhysicalDatabaseBackupType.FULL ? (
            <div className="max-w-[320px] text-xs text-gray-500 dark:text-gray-400">
              {t('backups.physical.config.fullBackupsOnlyRetention')}
            </div>
          ) : (
            <div className="flex w-full items-center">
              <Select
                value={retentionPolicy}
                onChange={(policy) => updateRetention({ policy })}
                size="small"
                className="min-w-0 grow"
                popupMatchSelectWidth={false}
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
                <InfoCircleOutlined className="ml-2" style={{ color: 'gray' }} />
              </Tooltip>
            </div>
          )}

          {isShowChainsCount && (
            <div className="flex items-center gap-2">
              <div className="flex w-[110px] items-center text-sm text-gray-600 dark:text-gray-400">
                <span>{t('backups.physical.config.chainsCount')}</span>
                <Tooltip
                  className="cursor-pointer"
                  title={t('backups.physical.config.chainsCountTooltip')}
                >
                  <InfoCircleOutlined className="ml-1" style={{ color: 'gray' }} />
                </Tooltip>
              </div>
              <InputNumber
                min={1}
                value={settings.retention.chainsCount}
                onChange={(v) => updateRetention({ chainsCount: v ?? 1 })}
                size="small"
                className="w-[80px] max-w-[80px]"
              />
            </div>
          )}

          {isShowFullBackupsEditor && (
            <div className="mt-1 flex flex-col gap-2">
              <div className="flex w-full max-w-[200px] items-center gap-2">
                <span className="w-[110px] text-sm text-gray-600 dark:text-gray-400">
                  {t('backups.physical.config.fullBackups')}
                </span>
                <Select
                  value={fullBackupsRetention.policy}
                  onChange={(policy) => updateFullBackupsRetention({ policy })}
                  size="small"
                  className="mr-5 max-w-[80px] min-w-0 grow"
                  popupMatchSelectWidth={false}
                  options={fullBackupsPolicyOptions}
                />
              </div>

              {fullBackupsRetention.policy === FullBackupsRetentionPolicy.LAST_N && (
                <div className="flex items-center gap-2">
                  <span className="max-w-[110px] shrink-0 text-sm leading-4 text-gray-600 dark:text-gray-400">
                    {t('backups.physical.config.fullBackupsCount')}
                  </span>
                  <InputNumber
                    min={1}
                    value={fullBackupsRetention.count}
                    onChange={(v) => updateFullBackupsRetention({ count: v ?? 1 })}
                    size="small"
                    className="w-[80px] max-w-[80px]"
                  />
                </div>
              )}

              {fullBackupsRetention.policy === FullBackupsRetentionPolicy.GFS && (
                <div className="flex max-w-[200px] flex-col gap-1">
                  <div className="flex items-center gap-2">
                    <span className="w-[110px] text-sm text-gray-600 dark:text-gray-400">
                      {t('backups.physical.config.gfsFields.hourly')}
                    </span>
                    <InputNumber
                      min={0}
                      value={fullBackupsRetention.gfsHours}
                      onChange={(v) => updateFullBackupsRetention({ gfsHours: v ?? 0 })}
                      size="small"
                      className="w-[80px] max-w-[80px]"
                    />
                  </div>
                  <div className="flex items-center gap-2">
                    <span className="w-[110px] text-sm text-gray-600 dark:text-gray-400">
                      {t('backups.physical.config.gfsFields.daily')}
                    </span>
                    <InputNumber
                      min={0}
                      value={fullBackupsRetention.gfsDays}
                      onChange={(v) => updateFullBackupsRetention({ gfsDays: v ?? 0 })}
                      size="small"
                      className="w-[80px] max-w-[80px]"
                    />
                  </div>
                  <div className="flex items-center gap-2">
                    <span className="w-[110px] text-sm text-gray-600 dark:text-gray-400">
                      {t('backups.physical.config.gfsFields.weekly')}
                    </span>
                    <InputNumber
                      min={0}
                      value={fullBackupsRetention.gfsWeeks}
                      onChange={(v) => updateFullBackupsRetention({ gfsWeeks: v ?? 0 })}
                      size="small"
                      className="w-[80px] max-w-[80px]"
                    />
                  </div>
                  <div className="flex items-center gap-2">
                    <span className="w-[110px] text-sm text-gray-600 dark:text-gray-400">
                      {t('backups.physical.config.gfsFields.monthly')}
                    </span>
                    <InputNumber
                      min={0}
                      value={fullBackupsRetention.gfsMonths}
                      onChange={(v) => updateFullBackupsRetention({ gfsMonths: v ?? 0 })}
                      size="small"
                      className="w-[80px] max-w-[80px]"
                    />
                  </div>
                  <div className="flex items-center gap-2">
                    <span className="w-[110px] text-sm text-gray-600 dark:text-gray-400">
                      {t('backups.physical.config.gfsFields.yearly')}
                    </span>
                    <InputNumber
                      min={0}
                      value={fullBackupsRetention.gfsYears}
                      onChange={(v) => updateFullBackupsRetention({ gfsYears: v ?? 0 })}
                      size="small"
                      className="w-[80px] max-w-[80px]"
                    />
                  </div>
                </div>
              )}
            </div>
          )}
        </div>
      </div>

      {backupConfig.isBackupsEnabled && (
        <div className="mt-4 mb-1 flex w-full flex-col items-start sm:flex-row sm:items-start">
          <div className="mt-0 mb-1 min-w-[150px] sm:mt-1 sm:mb-0 sm:pr-2">
            {t('backups.config.notifications')}
          </div>
          <div className="flex flex-col space-y-2">
            <Checkbox
              checked={settings.notificationTriggers.includes(
                PhysicalNotificationTrigger.BACKUP_SUCCESS,
              )}
              onChange={(e) =>
                toggleNotificationTrigger(
                  PhysicalNotificationTrigger.BACKUP_SUCCESS,
                  e.target.checked,
                )
              }
            >
              {t(
                PHYSICAL_NOTIFICATION_TRIGGER_LABEL_KEYS[
                  PhysicalNotificationTrigger.BACKUP_SUCCESS
                ],
              )}
            </Checkbox>

            <Checkbox
              checked={settings.notificationTriggers.includes(
                PhysicalNotificationTrigger.BACKUP_FAILED,
              )}
              onChange={(e) =>
                toggleNotificationTrigger(
                  PhysicalNotificationTrigger.BACKUP_FAILED,
                  e.target.checked,
                )
              }
            >
              {t(
                PHYSICAL_NOTIFICATION_TRIGGER_LABEL_KEYS[PhysicalNotificationTrigger.BACKUP_FAILED],
              )}
            </Checkbox>

            {isIncrementalAllowed && (
              <Checkbox
                checked={settings.notificationTriggers.includes(
                  PhysicalNotificationTrigger.CHAIN_BROKEN,
                )}
                onChange={(e) =>
                  toggleNotificationTrigger(
                    PhysicalNotificationTrigger.CHAIN_BROKEN,
                    e.target.checked,
                  )
                }
              >
                {t(
                  PHYSICAL_NOTIFICATION_TRIGGER_LABEL_KEYS[
                    PhysicalNotificationTrigger.CHAIN_BROKEN
                  ],
                )}
              </Checkbox>
            )}

            {isWalStream && (
              <Checkbox
                checked={settings.notificationTriggers.includes(
                  PhysicalNotificationTrigger.WAL_GAP,
                )}
                onChange={(e) =>
                  toggleNotificationTrigger(PhysicalNotificationTrigger.WAL_GAP, e.target.checked)
                }
              >
                {t(PHYSICAL_NOTIFICATION_TRIGGER_LABEL_KEYS[PhysicalNotificationTrigger.WAL_GAP])}
              </Checkbox>
            )}
          </div>
        </div>
      )}

      <div className="mt-5 flex">
        {isShowBackButton && (
          <Button className="mr-1" type="primary" ghost onClick={onBack}>
            {t('common.actions.back')}
          </Button>
        )}

        {isShowCancelButton && (
          <Button danger ghost className="mr-1" disabled={isSaving} onClick={onCancel}>
            {t('common.actions.cancel')}
          </Button>
        )}

        <Button
          type="primary"
          className={`${isShowCancelButton ? 'ml-1' : 'ml-auto'} mr-5`}
          onClick={confirmSaveBackupConfig}
          loading={isSaving}
          disabled={!isAllFieldsFilled || (isSaveToApi && !isUnsaved)}
        >
          {saveButtonText || t('common.actions.save')}
        </Button>
      </div>

      {isSaveToApi && database.id && (
        <StorageSwitchHistoryComponent
          workspaceId={database.workspaceId}
          databaseId={database.id}
          initialOperation={acceptedSwitch}
        />
      )}
      {isShowWarn && (
        <ConfirmationComponent
          onConfirm={() => {
            setIsShowWarn(false);
            void saveBackupConfig();
          }}
          onDecline={() => setIsShowWarn(false)}
          description={t('backups.storageSwitch.warning')}
          isDestructive
          actionText={t('backups.storageSwitch.confirm')}
          cancelText={t('common.actions.cancel')}
        />
      )}
    </div>
  );
};
