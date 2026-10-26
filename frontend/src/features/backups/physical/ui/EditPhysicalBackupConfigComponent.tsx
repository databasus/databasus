import { InfoCircleOutlined } from '@ant-design/icons';
import { Button, Checkbox, InputNumber, Modal, Select, Spin, Switch, Tooltip } from 'antd';
import { type JSX, useEffect, useState } from 'react';
import { useTranslation } from 'react-i18next';

import {
  type FullBackupsRetention,
  PHYSICAL_BACKUP_NOTIFICATION_TYPE_LABEL_KEYS,
  PHYSICAL_FULL_BACKUPS_POLICY_LABEL_KEYS,
  type PhysicalBackupConfig,
  PhysicalBackupNotificationType,
  PhysicalFullBackupsPolicy,
  PhysicalRetention,
  physicalBackupConfigApi,
} from '../../../../entity/backups/physical';
import {
  BACKUP_ENCRYPTION_OPTION_LABEL_KEYS,
  BackupEncryption,
} from '../../../../entity/backups/shared';
import {
  type Database,
  DatabaseType,
  PhysicalDatabaseBackupType,
} from '../../../../entity/databases';
import { type Interval, IntervalType } from '../../../../entity/intervals';
import { type Storage, getStorageLogoFromType, storageApi } from '../../../../entity/storages';
import { translateApiError } from '../../../../shared/i18n';
import { ConfirmationComponent } from '../../../../shared/ui';
import { EditStorageComponent } from '../../../storages/ui/edit/EditStorageComponent';
import { toBackupConfigStorage } from '../../lib/toBackupConfigStorage';
import { PhysicalIntervalEditor } from './PhysicalIntervalEditor';

interface Props {
  database: Database;

  // Seeds the editor in wizard mode (isSaveToApi=false) so going Back restores the
  // in-progress config instead of refetching a default by database id.
  initialConfig?: PhysicalBackupConfig;

  isShowBackButton: boolean;
  onBack: () => void;

  isShowCancelButton?: boolean;
  onCancel: () => void;

  saveButtonText?: string;
  isSaveToApi: boolean;
  onSaved: (backupConfig: PhysicalBackupConfig) => void;
}

const BYTES_IN_MB = 1024 * 1024;

const isFullBackupsRetentionValid = (retention: FullBackupsRetention): boolean => {
  if (retention.policy === PhysicalFullBackupsPolicy.LAST_N) {
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

// The backend forbids the retention sub-object the selected mode doesn't use (e.g. CHAINS
// must not carry full_backups_retention). The editor keeps both populated so values
// survive toggling the dropdown, so zero the unused one only at save time.
const ZERO_FULL_BACKUPS_RETENTION: FullBackupsRetention = {
  policy: '' as PhysicalFullBackupsPolicy, // empty policy makes the backend treat it as unset
  count: 0,
  gfsHours: 0,
  gfsDays: 0,
  gfsWeeks: 0,
  gfsMonths: 0,
  gfsYears: 0,
};

const normalizeRetentionForSave = (config: PhysicalBackupConfig): PhysicalBackupConfig => {
  if (config.retention === PhysicalRetention.CHAINS) {
    return { ...config, fullBackupsRetention: ZERO_FULL_BACKUPS_RETENTION };
  }

  if (config.retention === PhysicalRetention.FULL_BACKUPS) {
    return { ...config, chainsRetention: { count: 0 } };
  }

  return config;
};

const isIntervalValid = (interval?: Interval): boolean => {
  if (!interval?.type) return false;

  if (interval.type === IntervalType.WEEKLY) return Boolean(interval.weekday);
  if (interval.type === IntervalType.MONTHLY) return Boolean(interval.dayOfMonth);
  if (interval.type === IntervalType.CRON) return Boolean(interval.cronExpression);

  return true;
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
  const [backupConfig, setBackupConfig] = useState<PhysicalBackupConfig>();
  const [isUnsaved, setIsUnsaved] = useState(false);
  const [isSaving, setIsSaving] = useState(false);
  const [isLoading, setIsLoading] = useState(true);

  const [storages, setStorages] = useState<Storage[]>([]);
  const [isShowCreateStorage, setShowCreateStorage] = useState(false);
  const [storageSelectKey, setStorageSelectKey] = useState(0);

  const [isShowWarn, setIsShowWarn] = useState(false);

  // DB-level strategy lives on the database, not the config. It decides which
  // cadences and retention modes the config may use.
  const backupType =
    database.settings.type === DatabaseType.POSTGRES_PHYSICAL
      ? database.settings.backupType
      : PhysicalDatabaseBackupType.FULL;
  const isIncrementalAllowed = backupType !== PhysicalDatabaseBackupType.FULL;
  const isWalStream = backupType === PhysicalDatabaseBackupType.FULL_INCREMENTAL_WAL_STREAM;

  const updateBackupConfig = (patch: Partial<PhysicalBackupConfig>) => {
    setBackupConfig((prev) => (prev ? { ...prev, ...patch } : prev));
    setIsUnsaved(true);
  };

  const updateFullInterval = (patch: Partial<Interval>) => {
    setBackupConfig((prev) => {
      if (!prev) return prev;
      const merged = { ...(prev.fullBackupInterval ?? {}), ...patch } as Interval;
      return { ...prev, fullBackupInterval: merged };
    });
    setIsUnsaved(true);
  };

  const updateIncrementalInterval = (patch: Partial<Interval>) => {
    setBackupConfig((prev) => {
      if (!prev) return prev;
      const merged = { ...(prev.incrementalBackupInterval ?? {}), ...patch } as Interval;
      return { ...prev, incrementalBackupInterval: merged };
    });
    setIsUnsaved(true);
  };

  const updateFullBackupsRetention = (patch: Partial<FullBackupsRetention>) => {
    setBackupConfig((prev) =>
      prev ? { ...prev, fullBackupsRetention: { ...prev.fullBackupsRetention, ...patch } } : prev,
    );
    setIsUnsaved(true);
  };

  const toggleNotification = (type: PhysicalBackupNotificationType, isChecked: boolean) => {
    setBackupConfig((prev) => {
      if (!prev) return prev;
      const notifications = prev.sendNotificationsOn.filter((n) => n !== type);
      if (isChecked) notifications.push(type);
      return { ...prev, sendNotificationsOn: notifications };
    });
    setIsUnsaved(true);
  };

  const saveBackupConfig = async () => {
    if (!backupConfig) return;

    const configToSave = normalizeRetentionForSave(backupConfig);

    if (isSaveToApi) {
      setIsSaving(true);
      try {
        await physicalBackupConfigApi.savePhysicalBackupConfig(configToSave);
        setIsUnsaved(false);
      } catch (e) {
        alert(translateApiError(e, t));
      }
      setIsSaving(false);
    }

    onSaved(configToSave);
  };

  const loadStorages = async () => {
    try {
      const loadedStorages = await storageApi.listAllStorages(database.workspaceId);
      setStorages(loadedStorages);
    } catch (e) {
      alert(translateApiError(e, t));
    }
  };

  const buildDefaultConfig = (): PhysicalBackupConfig => ({
    databaseId: database.id,
    isBackupsEnabled: true,
    fullBackupInterval: { type: IntervalType.WEEKLY, timeOfDay: '00:00', weekday: 1 },
    incrementalBackupInterval: isIncrementalAllowed
      ? { type: IntervalType.DAILY, timeOfDay: '00:00' }
      : undefined,
    retention:
      backupType === PhysicalDatabaseBackupType.FULL
        ? PhysicalRetention.FULL_BACKUPS
        : PhysicalRetention.CHAINS_AND_FULL_BACKUPS,
    chainsRetention: { count: 7 },
    fullBackupsRetention: {
      policy: PhysicalFullBackupsPolicy.GFS,
      count: 0,
      gfsHours: 0,
      gfsDays: 7,
      gfsWeeks: 4,
      gfsMonths: 12,
      gfsYears: 3,
    },
    walLagThresholdBytes: isWalStream ? 256 * BYTES_IN_MB : 0,
    storage: undefined,
    encryption: BackupEncryption.NONE,
    sendNotificationsOn: [PhysicalBackupNotificationType.BACKUP_FAILED],
  });

  useEffect(() => {
    const run = async () => {
      setIsLoading(true);

      try {
        // Wizard mode (isSaveToApi=false) is purely in-memory: never refetch by database
        // id, even after the DB row exists - seed from the in-progress config or defaults.
        if (isSaveToApi && database.id) {
          const config = await physicalBackupConfigApi.getPhysicalBackupConfigByDbId(database.id);
          setBackupConfig(config);
          setIsUnsaved(false);
          setIsSaving(false);
        } else {
          setBackupConfig(initialConfig ?? buildDefaultConfig());
        }

        await loadStorages();
      } catch (e) {
        alert(translateApiError(e, t));
      } finally {
        setIsLoading(false);
      }
    };

    run();
  }, [database]);

  if (isLoading) {
    return (
      <div className="mb-5 flex items-center">
        <Spin />
      </div>
    );
  }

  if (!backupConfig) return <div />;

  const fullBackupsRetention = backupConfig.fullBackupsRetention;

  // FULL databases are forced to FULL_BACKUPS retention; the others may pick
  // between CHAINS and CHAINS_AND_FULL_BACKUPS.
  const isShowChainsCount =
    backupType !== PhysicalDatabaseBackupType.FULL &&
    (backupConfig.retention === PhysicalRetention.CHAINS ||
      backupConfig.retention === PhysicalRetention.CHAINS_AND_FULL_BACKUPS);

  const isShowFullBackupsEditor =
    backupType === PhysicalDatabaseBackupType.FULL ||
    backupConfig.retention === PhysicalRetention.CHAINS_AND_FULL_BACKUPS;

  const isChainsCountValid = !isShowChainsCount || (backupConfig.chainsRetention?.count ?? 0) > 0;

  const isFullRetentionValid =
    !isShowFullBackupsEditor || isFullBackupsRetentionValid(fullBackupsRetention);

  const encryptionOptions = Object.values(BackupEncryption).map((encryption) => ({
    value: encryption,
    label: t(BACKUP_ENCRYPTION_OPTION_LABEL_KEYS[encryption]),
  }));

  const fullBackupsPolicyOptions = Object.values(PhysicalFullBackupsPolicy).map((policy) => ({
    value: policy,
    label: t(PHYSICAL_FULL_BACKUPS_POLICY_LABEL_KEYS[policy]),
  }));

  const isAllFieldsFilled =
    !backupConfig.isBackupsEnabled ||
    (Boolean(backupConfig.storage?.id) &&
      Boolean(backupConfig.encryption) &&
      isIntervalValid(backupConfig.fullBackupInterval) &&
      (!isIncrementalAllowed || isIntervalValid(backupConfig.incrementalBackupInterval)) &&
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
          <PhysicalIntervalEditor
            label={t('backups.physical.config.fullBackupCadence')}
            interval={backupConfig.fullBackupInterval}
            onChange={updateFullInterval}
          />

          {isIncrementalAllowed && (
            <>
              <PhysicalIntervalEditor
                label={t('backups.physical.config.incrementalBackupCadence')}
                interval={backupConfig.incrementalBackupInterval}
                onChange={updateIncrementalInterval}
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
          <Select
            key={storageSelectKey}
            value={backupConfig.storage?.id}
            onChange={(storageId) => {
              if (storageId.includes('create-new-storage')) {
                setShowCreateStorage(true);
                return;
              }

              const selectedStorage = storages.find((s) => s.id === storageId);
              updateBackupConfig({
                storage: selectedStorage && toBackupConfigStorage(selectedStorage),
              });

              if (backupConfig.storage?.id) {
                setIsShowWarn(true);
              }
            }}
            size="small"
            className="mr-2 max-w-[200px] grow"
            options={[
              ...storages.map((s) => ({ label: s.name, value: s.id })),
              { label: t('backups.config.createNewStorage'), value: 'create-new-storage' },
            ]}
            placeholder={t('backups.config.selectStorage')}
          />

          {backupConfig.storage?.type && (
            <img
              src={getStorageLogoFromType(backupConfig.storage.type)}
              alt=""
              className="ml-1 h-4 w-4"
            />
          )}
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
                value={backupConfig.retention}
                onChange={(v) => updateBackupConfig({ retention: v })}
                size="small"
                className="min-w-0 grow"
                popupMatchSelectWidth={false}
                options={[
                  {
                    label: t('backups.physical.config.retentionOptions.chains'),
                    value: PhysicalRetention.CHAINS,
                  },
                  {
                    label: t('backups.physical.config.retentionOptions.chainsAndFullBackups'),
                    value: PhysicalRetention.CHAINS_AND_FULL_BACKUPS,
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
                value={backupConfig.chainsRetention?.count}
                onChange={(v) => updateBackupConfig({ chainsRetention: { count: v ?? 1 } })}
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

              {fullBackupsRetention.policy === PhysicalFullBackupsPolicy.LAST_N && (
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

              {fullBackupsRetention.policy === PhysicalFullBackupsPolicy.GFS && (
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
              checked={backupConfig.sendNotificationsOn.includes(
                PhysicalBackupNotificationType.BACKUP_SUCCESS,
              )}
              onChange={(e) =>
                toggleNotification(PhysicalBackupNotificationType.BACKUP_SUCCESS, e.target.checked)
              }
            >
              {t(
                PHYSICAL_BACKUP_NOTIFICATION_TYPE_LABEL_KEYS[
                  PhysicalBackupNotificationType.BACKUP_SUCCESS
                ],
              )}
            </Checkbox>

            <Checkbox
              checked={backupConfig.sendNotificationsOn.includes(
                PhysicalBackupNotificationType.BACKUP_FAILED,
              )}
              onChange={(e) =>
                toggleNotification(PhysicalBackupNotificationType.BACKUP_FAILED, e.target.checked)
              }
            >
              {t(
                PHYSICAL_BACKUP_NOTIFICATION_TYPE_LABEL_KEYS[
                  PhysicalBackupNotificationType.BACKUP_FAILED
                ],
              )}
            </Checkbox>

            {isIncrementalAllowed && (
              <Checkbox
                checked={backupConfig.sendNotificationsOn.includes(
                  PhysicalBackupNotificationType.CHAIN_BROKEN,
                )}
                onChange={(e) =>
                  toggleNotification(PhysicalBackupNotificationType.CHAIN_BROKEN, e.target.checked)
                }
              >
                {t(
                  PHYSICAL_BACKUP_NOTIFICATION_TYPE_LABEL_KEYS[
                    PhysicalBackupNotificationType.CHAIN_BROKEN
                  ],
                )}
              </Checkbox>
            )}

            {isWalStream && (
              <Checkbox
                checked={backupConfig.sendNotificationsOn.includes(
                  PhysicalBackupNotificationType.WAL_GAP,
                )}
                onChange={(e) =>
                  toggleNotification(PhysicalBackupNotificationType.WAL_GAP, e.target.checked)
                }
              >
                {t(
                  PHYSICAL_BACKUP_NOTIFICATION_TYPE_LABEL_KEYS[
                    PhysicalBackupNotificationType.WAL_GAP
                  ],
                )}
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
          <Button danger ghost className="mr-1" onClick={onCancel}>
            {t('common.actions.cancel')}
          </Button>
        )}

        <Button
          type="primary"
          className={`${isShowCancelButton ? 'ml-1' : 'ml-auto'} mr-5`}
          onClick={saveBackupConfig}
          loading={isSaving}
          disabled={!isAllFieldsFilled || (isSaveToApi && !isUnsaved)}
        >
          {saveButtonText || t('common.actions.save')}
        </Button>
      </div>

      {isShowCreateStorage && (
        <Modal
          title={t('backups.config.addStorageTitle')}
          footer={<div />}
          open={isShowCreateStorage}
          onCancel={() => {
            setShowCreateStorage(false);
            setStorageSelectKey((prev) => prev + 1);
          }}
          maskClosable={false}
        >
          <div className="my-3 max-w-[275px] text-gray-500 dark:text-gray-400">
            {t('backups.config.storageDescription')}
          </div>

          <EditStorageComponent
            workspaceId={database.workspaceId}
            isShowName
            isShowClose={false}
            onClose={() => setShowCreateStorage(false)}
            onChanged={async (createdStorage) => {
              const hadExistingStorage = !!backupConfig?.storage?.id;
              await loadStorages();
              updateBackupConfig({ storage: toBackupConfigStorage(createdStorage) });
              setShowCreateStorage(false);
              if (hadExistingStorage) {
                setIsShowWarn(true);
              }
            }}
          />
        </Modal>
      )}

      {isShowWarn && (
        <ConfirmationComponent
          onConfirm={() => setIsShowWarn(false)}
          onDecline={() => setIsShowWarn(false)}
          description={t('backups.config.storageChangeWarning')}
          actionButtonColor="red"
          actionText={t('backups.config.storageChangeAcknowledge')}
          cancelText={t('common.actions.cancel')}
          hideCancelButton
        />
      )}
    </div>
  );
};
