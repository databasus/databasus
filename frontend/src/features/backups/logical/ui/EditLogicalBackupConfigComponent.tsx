import { DownOutlined, InfoCircleOutlined, UpOutlined } from '@ant-design/icons';
import {
  Button,
  Checkbox,
  Input,
  InputNumber,
  Modal,
  Select,
  Spin,
  Switch,
  TimePicker,
  Tooltip,
} from 'antd';
import { CronExpressionParser } from 'cron-parser';
import dayjs, { Dayjs } from 'dayjs';
import { useEffect, useMemo, useState } from 'react';
import { useTranslation } from 'react-i18next';

import {
  BackupSettingsType,
  type LogicalBackupConfigInput,
  type LogicalBackupSettings,
  SCHEDULE_TYPE_LABEL_KEYS,
  type Schedule,
  ScheduleType,
  backupConfigApi,
  isScheduleFilled,
  toLogicalBackupConfigInput,
} from '../../../../entity/backups/config';
import {
  LOGICAL_NOTIFICATION_TRIGGER_LABEL_KEYS,
  LOGICAL_RETENTION_POLICY_LABEL_KEYS,
  LogicalNotificationTrigger,
  LogicalRetentionPolicy,
  RETENTION_PERIOD_LABEL_KEYS,
  RetentionPeriod,
} from '../../../../entity/backups/logical';
import {
  BACKUP_ENCRYPTION_OPTION_LABEL_KEYS,
  BackupEncryption,
} from '../../../../entity/backups/shared';
import type { Database } from '../../../../entity/databases';
import { WEEKDAYS, WEEKDAY_LABEL_KEYS } from '../../../../entity/intervals';
import { type Storage, getStorageLogoFromType, storageApi } from '../../../../entity/storages';
import { translateApiError, useLocale } from '../../../../shared/i18n';
import { getUserTimeFormat } from '../../../../shared/time';
import {
  getUserTimeFormat as getIs12Hour,
  getLocalDayOfMonth,
  getLocalWeekday,
  getUtcDayOfMonth,
  getUtcWeekday,
} from '../../../../shared/time/utils';
import { ConfirmationComponent } from '../../../../shared/ui';
import { EditStorageComponent } from '../../../storages/ui/edit/EditStorageComponent';

interface Props {
  database: Database;

  isShowBackButton: boolean;
  onBack: () => void;

  isShowCancelButton?: boolean;
  onCancel: () => void;

  saveButtonText?: string;
  isSaveToApi: boolean;
  onSaved: (backupConfig: LogicalBackupConfigInput) => void;
}

type LogicalRetention = LogicalBackupSettings['retention'];

const RETENTION_POLICY_ORDER = [
  LogicalRetentionPolicy.GFS,
  LogicalRetentionPolicy.TIME_PERIOD,
  LogicalRetentionPolicy.COUNT,
];

const createDefaultLogicalBackupConfig = (): LogicalBackupConfigInput => ({
  isBackupsEnabled: true,
  storageId: null,
  encryption: BackupEncryption.ENCRYPTED,
  settings: {
    type: BackupSettingsType.LOGICAL,
    schedule: { type: ScheduleType.DAILY, timeOfDay: '00:00' },
    retention: {
      policy: LogicalRetentionPolicy.TIME_PERIOD,
      period: RetentionPeriod.THREE_MONTHS,
      count: 100,
      gfsHours: 24,
      gfsDays: 7,
      gfsWeeks: 4,
      gfsMonths: 12,
      gfsYears: 3,
    },
    isRetryIfFailed: true,
    maxFailedTriesCount: 3,
    notificationTriggers: [LogicalNotificationTrigger.BACKUP_FAILED],
  },
});

export const EditLogicalBackupConfigComponent = ({
  database,

  isShowBackButton,
  onBack,

  isShowCancelButton,
  onCancel,
  saveButtonText,
  isSaveToApi,
  onSaved,
}: Props) => {
  const { t } = useTranslation();
  const { formatRelativeTime } = useLocale();
  const [backupConfig, setBackupConfig] = useState<LogicalBackupConfigInput>();
  const [isUnsaved, setIsUnsaved] = useState(false);
  const [isSaving, setIsSaving] = useState(false);

  const [storages, setStorages] = useState<Storage[]>([]);
  const [isShowCreateStorage, setShowCreateStorage] = useState(false);
  const [storageSelectKey, setStorageSelectKey] = useState(0);

  const [isShowWarn, setIsShowWarn] = useState(false);

  const [isLoading, setIsLoading] = useState(true);

  const hasAdvancedValues = !!backupConfig?.settings.isRetryIfFailed;
  const [isShowAdvanced, setShowAdvanced] = useState(hasAdvancedValues);
  const [isShowGfsHint, setShowGfsHint] = useState(false);

  const timeFormat = useMemo(() => {
    const is12 = getIs12Hour();
    return { use12Hours: is12, format: is12 ? 'h:mm A' : 'HH:mm' };
  }, []);

  const dateTimeFormat = useMemo(() => getUserTimeFormat(), []);

  const updateBackupConfig = (patch: Partial<LogicalBackupConfigInput>) => {
    setBackupConfig((prev) => (prev ? { ...prev, ...patch } : prev));
    setIsUnsaved(true);
  };

  const updateSettings = (patch: Partial<LogicalBackupSettings>) => {
    setBackupConfig((prev) =>
      prev ? { ...prev, settings: { ...prev.settings, ...patch } } : prev,
    );
    setIsUnsaved(true);
  };

  const updateRetention = (patch: Partial<LogicalRetention>) => {
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

  const updateSchedule = (patch: Partial<Schedule>) => {
    setBackupConfig((prev) =>
      prev
        ? {
            ...prev,
            settings: { ...prev.settings, schedule: { ...prev.settings.schedule, ...patch } },
          }
        : prev,
    );
    setIsUnsaved(true);
  };

  const toggleNotificationTrigger = (trigger: LogicalNotificationTrigger, isChecked: boolean) => {
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

  const saveBackupConfig = async () => {
    if (!backupConfig) return;

    if (isSaveToApi) {
      setIsSaving(true);
      try {
        await backupConfigApi.updateBackupConfig(database.id, backupConfig);
        setIsUnsaved(false);
      } catch (e) {
        alert(translateApiError(e, t));
      }
      setIsSaving(false);
    }

    onSaved(backupConfig);
  };

  const loadStorages = async () => {
    try {
      const storages = await storageApi.listAllStorages(database.workspaceId);
      setStorages(storages);
    } catch (e) {
      alert(translateApiError(e, t));
    }
  };

  useEffect(() => {
    const run = async () => {
      setIsLoading(true);

      try {
        if (database.id) {
          const loadedBackupConfig = await backupConfigApi.getBackupConfig(database.id);
          setBackupConfig(toLogicalBackupConfigInput(loadedBackupConfig));
          setIsUnsaved(false);
          setIsSaving(false);
        } else {
          setBackupConfig(createDefaultLogicalBackupConfig());
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

  const { settings } = backupConfig;
  const { schedule, retention } = settings;

  // UTC → local conversions for display
  const localTime: Dayjs | undefined = schedule.timeOfDay
    ? dayjs.utc(schedule.timeOfDay, 'HH:mm').local()
    : undefined;

  const displayedWeekday: number | undefined =
    schedule.type === ScheduleType.WEEKLY && schedule.weekday && schedule.timeOfDay
      ? getLocalWeekday(schedule.weekday, schedule.timeOfDay)
      : schedule.weekday;

  const displayedDayOfMonth: number | undefined =
    schedule.type === ScheduleType.MONTHLY && schedule.dayOfMonth && schedule.timeOfDay
      ? getLocalDayOfMonth(schedule.dayOfMonth, schedule.timeOfDay)
      : schedule.dayOfMonth;

  const retentionPolicy = retention.policy;

  const selectedStorage = storages.find((storage) => storage.id === backupConfig.storageId);

  const isShowGfsHours =
    schedule.type === ScheduleType.HOURLY || schedule.type === ScheduleType.CRON;

  const isRetentionValid = (() => {
    switch (retentionPolicy) {
      case LogicalRetentionPolicy.TIME_PERIOD:
        return Boolean(retention.period);
      case LogicalRetentionPolicy.COUNT:
        return retention.count > 0;
      case LogicalRetentionPolicy.GFS:
        return (
          retention.gfsDays > 0 ||
          retention.gfsWeeks > 0 ||
          retention.gfsMonths > 0 ||
          retention.gfsYears > 0
        );
    }
  })();

  const weekdayOptions = WEEKDAYS.map((weekday) => ({
    value: weekday,
    label: t(WEEKDAY_LABEL_KEYS[weekday]),
  }));

  const scheduleTypeOptions = Object.values(ScheduleType).map((scheduleType) => ({
    value: scheduleType,
    label: t(SCHEDULE_TYPE_LABEL_KEYS[scheduleType]),
  }));

  const retentionPolicyOptions = RETENTION_POLICY_ORDER.map((policy) => ({
    value: policy,
    label: t(LOGICAL_RETENTION_POLICY_LABEL_KEYS[policy]),
  }));

  const periodOptions = Object.values(RetentionPeriod).map((period) => ({
    value: period,
    label: t(RETENTION_PERIOD_LABEL_KEYS[period]),
  }));

  const encryptionOptions = Object.values(BackupEncryption).map((encryption) => ({
    value: encryption,
    label: t(BACKUP_ENCRYPTION_OPTION_LABEL_KEYS[encryption]),
  }));

  const isAllFieldsFilled =
    !backupConfig.isBackupsEnabled ||
    (isRetentionValid &&
      Boolean(backupConfig.storageId) &&
      Boolean(backupConfig.encryption) &&
      isScheduleFilled(schedule));

  return (
    <div>
      {database.id && (
        <div className="mb-1 flex w-full flex-col items-start sm:flex-row sm:items-center">
          <div className="mb-1 min-w-[150px] sm:mb-0 sm:pr-2">
            {t('backups.config.backupsEnabled')}
          </div>
          <Switch
            checked={backupConfig.isBackupsEnabled}
            onChange={(checked) => {
              updateBackupConfig({ isBackupsEnabled: checked });
            }}
            size="small"
          />
        </div>
      )}

      {backupConfig.isBackupsEnabled && (
        <>
          <div className="mt-4 mb-1 flex w-full flex-col items-start sm:flex-row sm:items-center">
            <div className="mb-1 min-w-[150px] sm:mb-0 sm:pr-2">
              {t('backups.logical.config.interval')}
            </div>
            <Select
              value={schedule.type}
              onChange={(v) => {
                updateSchedule({ type: v });

                const isDailyOrMore =
                  v === ScheduleType.DAILY ||
                  v === ScheduleType.WEEKLY ||
                  v === ScheduleType.MONTHLY;

                if (isDailyOrMore && retentionPolicy === LogicalRetentionPolicy.GFS) {
                  updateRetention({ gfsHours: 24 });
                }
              }}
              size="small"
              className="w-full max-w-[200px] grow"
              options={scheduleTypeOptions}
            />
          </div>

          {schedule.type === ScheduleType.WEEKLY && (
            <div className="mb-1 flex w-full flex-col items-start sm:flex-row sm:items-center">
              <div className="mb-1 min-w-[150px] sm:mb-0 sm:pr-2">
                {t('backups.logical.config.weekday')}
              </div>
              <Select
                value={displayedWeekday}
                onChange={(localWeekday) => {
                  if (!localWeekday) return;
                  const ref = localTime ?? dayjs();
                  updateSchedule({ weekday: getUtcWeekday(localWeekday, ref) });
                }}
                size="small"
                className="w-full max-w-[200px] grow"
                options={weekdayOptions}
              />
            </div>
          )}

          {schedule.type === ScheduleType.MONTHLY && (
            <div className="mb-1 flex w-full flex-col items-start sm:flex-row sm:items-center">
              <div className="mb-1 min-w-[150px] sm:mb-0 sm:pr-2">
                {t('backups.logical.config.dayOfMonth')}
              </div>
              <InputNumber
                min={1}
                max={31}
                value={displayedDayOfMonth}
                onChange={(localDom) => {
                  if (!localDom) return;
                  const ref = localTime ?? dayjs();
                  updateSchedule({ dayOfMonth: getUtcDayOfMonth(localDom, ref) });
                }}
                size="small"
                className="w-full max-w-[200px] grow"
              />
            </div>
          )}

          {schedule.type === ScheduleType.CRON && (
            <>
              <div className="mb-1 flex w-full flex-col items-start sm:flex-row sm:items-center">
                <div className="mb-1 min-w-[150px] sm:mb-0 sm:pr-2">
                  {t('backups.config.schedule.cronExpression')}
                </div>
                <div className="flex items-center">
                  <Input
                    value={schedule.cronExpression || ''}
                    onChange={(e) => updateSchedule({ cronExpression: e.target.value })}
                    placeholder="0 2 * * *"
                    size="small"
                    className="w-full max-w-[200px] grow"
                  />
                  <Tooltip
                    className="cursor-pointer"
                    title={
                      <div>
                        <div className="font-bold">
                          {t('backups.config.schedule.cronHelp.format')}
                        </div>
                        <div className="mt-1">{t('backups.config.schedule.cronHelp.examples')}</div>
                        <div>
                          {'•'} 0 2 * * * - {t('backups.config.schedule.cronHelp.dailyAt2am')}
                        </div>
                        <div>
                          {'•'} 0 */6 * * * - {t('backups.config.schedule.cronHelp.every6Hours')}
                        </div>
                        <div>
                          {'•'} 0 3 * * 1 - {t('backups.config.schedule.cronHelp.mondaysAt3am')}
                        </div>
                        <div>
                          {'•'} 30 4 1,15 * * - {t('backups.config.schedule.cronHelp.twiceAMonth')}
                        </div>
                      </div>
                    }
                  >
                    <InfoCircleOutlined className="ml-2" style={{ color: 'gray' }} />
                  </Tooltip>
                </div>
              </div>
              {schedule.cronExpression &&
                (() => {
                  try {
                    const interval = CronExpressionParser.parse(schedule.cronExpression, {
                      tz: 'UTC',
                    });
                    const nextRun = interval.next().toDate();
                    return (
                      <div className="mb-1 flex w-full flex-col items-start text-xs text-gray-600 sm:flex-row sm:items-center dark:text-gray-400">
                        <div className="mb-1 min-w-[150px] sm:mb-0 sm:pr-2" />
                        <div className="text-gray-600 dark:text-gray-400">
                          {t('backups.config.schedule.nextRun', {
                            time: dayjs(nextRun).local().format(dateTimeFormat.format),
                          })}
                          <br />({formatRelativeTime(nextRun)})
                        </div>
                      </div>
                    );
                  } catch {
                    return (
                      <div className="mb-1 flex w-full flex-col items-start text-red-500 sm:flex-row sm:items-center">
                        <div className="mb-1 min-w-[150px] sm:mb-0 sm:pr-2" />
                        <div className="text-red-500">
                          {t('backups.config.schedule.invalidCron')}
                        </div>
                      </div>
                    );
                  }
                })()}
            </>
          )}

          {schedule.type !== ScheduleType.HOURLY && schedule.type !== ScheduleType.CRON && (
            <div className="mb-1 flex w-full flex-col items-start sm:flex-row sm:items-center">
              <div className="mb-1 min-w-[150px] sm:mb-0 sm:pr-2">
                {t('backups.logical.config.timeOfDay')}
              </div>
              <TimePicker
                value={localTime}
                format={timeFormat.format}
                use12Hours={timeFormat.use12Hours}
                allowClear={false}
                size="small"
                className="w-full max-w-[200px] grow"
                onChange={(pickedTime) => {
                  if (!pickedTime) return;
                  const patch: Partial<Schedule> = {
                    timeOfDay: pickedTime.utc().format('HH:mm'),
                  };

                  if (schedule.type === ScheduleType.WEEKLY && displayedWeekday) {
                    patch.weekday = getUtcWeekday(displayedWeekday, pickedTime);
                  }
                  if (schedule.type === ScheduleType.MONTHLY && displayedDayOfMonth) {
                    patch.dayOfMonth = getUtcDayOfMonth(displayedDayOfMonth, pickedTime);
                  }

                  updateSchedule(patch);
                }}
              />
            </div>
          )}

          <div className="mb-3" />
        </>
      )}

      <div className="mt-5 mb-1 flex w-full flex-col items-start sm:flex-row sm:items-center">
        <div className="mb-1 min-w-[150px] sm:mb-0 sm:pr-2">{t('backups.config.storage')}</div>
        <div className="flex w-full items-center">
          <Select
            key={storageSelectKey}
            value={backupConfig.storageId ?? undefined}
            onChange={(storageId) => {
              if (storageId.includes('create-new-storage')) {
                setShowCreateStorage(true);
                return;
              }

              updateBackupConfig({ storageId });

              if (backupConfig.storageId) {
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

          {selectedStorage && (
            <img
              src={getStorageLogoFromType(selectedStorage.settings.type)}
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

          <Tooltip className="cursor-pointer" title={t('backups.logical.config.encryptionTooltip')}>
            <InfoCircleOutlined className="ml-2" style={{ color: 'gray' }} />
          </Tooltip>
        </div>
      </div>

      <div className="mt-5 mb-1 flex w-full flex-col items-start sm:flex-row sm:items-start">
        <div className="mt-1 mb-1 min-w-[150px] sm:mb-0 sm:pr-2">
          {t('backups.logical.config.retentionPolicy')}
        </div>
        <div className="flex min-w-0 grow flex-col gap-1">
          <Select
            value={retentionPolicy}
            options={retentionPolicyOptions}
            size="small"
            className="w-[200px]"
            popupMatchSelectWidth={false}
            onChange={(v) => {
              const updates: Partial<LogicalRetention> = { policy: v };

              if (v === LogicalRetentionPolicy.GFS) {
                updates.gfsHours = 24;
                updates.gfsDays = 7;
                updates.gfsWeeks = 4;
                updates.gfsMonths = 12;
                updates.gfsYears = 3;
              } else if (v === LogicalRetentionPolicy.COUNT) {
                updates.count = 100;
              }

              updateRetention(updates);
            }}
          />

          {retentionPolicy === LogicalRetentionPolicy.TIME_PERIOD && (
            <div className="flex w-full items-center">
              <Select
                value={retention.period}
                onChange={(v) => updateRetention({ period: v })}
                size="small"
                className="min-w-0 grow"
                options={periodOptions}
              />

              <Tooltip
                className="cursor-pointer"
                title={t('backups.logical.config.retention.timePeriodTooltip')}
              >
                <InfoCircleOutlined className="ml-2" style={{ color: 'gray' }} />
              </Tooltip>
            </div>
          )}

          {retentionPolicy === LogicalRetentionPolicy.COUNT && (
            <div className="flex items-center">
              <span className="mr-2 shrink-0 text-sm text-gray-600 dark:text-gray-400">
                {t('backups.logical.config.retention.count')}
              </span>
              <InputNumber
                min={1}
                value={retention.count}
                onChange={(v) => updateRetention({ count: v ?? 1 })}
                size="small"
                className="w-[80px]"
              />

              <Tooltip
                className="cursor-pointer"
                title={t('backups.logical.config.retention.countTooltip')}
              >
                <InfoCircleOutlined className="ml-2" style={{ color: 'gray' }} />
              </Tooltip>
            </div>
          )}

          {retentionPolicy === LogicalRetentionPolicy.GFS && (
            <>
              <div>
                <span
                  className="cursor-pointer text-xs text-blue-600 hover:text-blue-800"
                  onClick={() => setShowGfsHint(!isShowGfsHint)}
                >
                  {isShowGfsHint
                    ? t('backups.logical.config.retention.hideGfsHint')
                    : t('backups.logical.config.retention.showGfsHint')}
                </span>

                {isShowGfsHint && (
                  <div className="mt-1 max-w-[280px] text-xs text-gray-600 dark:text-gray-400">
                    {t('backups.logical.config.retention.gfsHint')}
                  </div>
                )}
              </div>

              <div className="flex flex-col gap-1">
                {isShowGfsHours && (
                  <div className="flex items-center gap-2">
                    <span className="w-[110px] text-sm text-gray-600 dark:text-gray-400">
                      {t('backups.logical.config.retention.gfsFields.hourly')}
                    </span>
                    <InputNumber
                      min={0}
                      value={retention.gfsHours}
                      onChange={(v) => updateRetention({ gfsHours: v ?? 0 })}
                      size="small"
                      className="w-[80px]"
                    />
                  </div>
                )}

                <div className="flex items-center gap-2">
                  <span className="w-[110px] text-sm text-gray-600 dark:text-gray-400">
                    {t('backups.logical.config.retention.gfsFields.daily')}
                  </span>
                  <InputNumber
                    min={0}
                    value={retention.gfsDays}
                    onChange={(v) => updateRetention({ gfsDays: v ?? 0 })}
                    size="small"
                    className="w-[80px]"
                  />
                </div>

                <div className="flex items-center gap-2">
                  <span className="w-[110px] text-sm text-gray-600 dark:text-gray-400">
                    {t('backups.logical.config.retention.gfsFields.weekly')}
                  </span>
                  <InputNumber
                    min={0}
                    value={retention.gfsWeeks}
                    onChange={(v) => updateRetention({ gfsWeeks: v ?? 0 })}
                    size="small"
                    className="w-[80px]"
                  />
                </div>

                <div className="flex items-center gap-2">
                  <span className="w-[110px] text-sm text-gray-600 dark:text-gray-400">
                    {t('backups.logical.config.retention.gfsFields.monthly')}
                  </span>
                  <InputNumber
                    min={0}
                    value={retention.gfsMonths}
                    onChange={(v) => updateRetention({ gfsMonths: v ?? 0 })}
                    size="small"
                    className="w-[80px]"
                  />
                </div>
                <div className="flex items-center gap-2">
                  <span className="w-[110px] text-sm text-gray-600 dark:text-gray-400">
                    {t('backups.logical.config.retention.gfsFields.yearly')}
                  </span>
                  <InputNumber
                    min={0}
                    value={retention.gfsYears}
                    onChange={(v) => updateRetention({ gfsYears: v ?? 0 })}
                    size="small"
                    className="w-[80px]"
                  />
                </div>
              </div>
            </>
          )}
        </div>
      </div>

      {backupConfig.isBackupsEnabled && (
        <>
          <div className="mt-4 mb-1 flex w-full flex-col items-start sm:flex-row sm:items-start">
            <div className="mt-0 mb-1 min-w-[150px] sm:mt-1 sm:mb-0 sm:pr-2">
              {t('backups.config.notifications')}
            </div>
            <div className="flex flex-col space-y-2">
              <Checkbox
                checked={settings.notificationTriggers.includes(
                  LogicalNotificationTrigger.BACKUP_SUCCESS,
                )}
                onChange={(e) =>
                  toggleNotificationTrigger(
                    LogicalNotificationTrigger.BACKUP_SUCCESS,
                    e.target.checked,
                  )
                }
              >
                {t(
                  LOGICAL_NOTIFICATION_TRIGGER_LABEL_KEYS[
                    LogicalNotificationTrigger.BACKUP_SUCCESS
                  ],
                )}
              </Checkbox>

              <Checkbox
                checked={settings.notificationTriggers.includes(
                  LogicalNotificationTrigger.BACKUP_FAILED,
                )}
                onChange={(e) =>
                  toggleNotificationTrigger(
                    LogicalNotificationTrigger.BACKUP_FAILED,
                    e.target.checked,
                  )
                }
              >
                {t(
                  LOGICAL_NOTIFICATION_TRIGGER_LABEL_KEYS[LogicalNotificationTrigger.BACKUP_FAILED],
                )}
              </Checkbox>
            </div>
          </div>
        </>
      )}

      <div className="mt-4 mb-1 flex items-center">
        <div
          className="flex cursor-pointer items-center text-sm text-blue-600 hover:text-blue-800"
          onClick={() => setShowAdvanced(!isShowAdvanced)}
        >
          <span className="mr-2">{t('backups.logical.config.advancedSettings')}</span>

          {isShowAdvanced ? (
            <UpOutlined style={{ fontSize: '12px' }} />
          ) : (
            <DownOutlined style={{ fontSize: '12px' }} />
          )}
        </div>
      </div>

      {isShowAdvanced && backupConfig.isBackupsEnabled && (
        <>
          <div className="mb-1 flex w-full flex-col items-start sm:flex-row sm:items-center">
            <div className="mb-1 min-w-[150px] sm:mb-0 sm:pr-2">
              {t('backups.logical.config.retryIfFailed')}
            </div>
            <div className="flex items-center">
              <Switch
                size="small"
                checked={settings.isRetryIfFailed}
                onChange={(checked) => updateSettings({ isRetryIfFailed: checked })}
              />

              <Tooltip
                className="cursor-pointer"
                title={t('backups.logical.config.retryIfFailedTooltip')}
              >
                <InfoCircleOutlined className="ml-2" style={{ color: 'gray' }} />
              </Tooltip>
            </div>
          </div>

          {settings.isRetryIfFailed && (
            <div className="mb-1 flex w-full flex-col items-start sm:flex-row sm:items-center">
              <div className="mb-1 min-w-[150px] sm:mb-0 sm:pr-2">
                {t('backups.logical.config.maxFailedTries')}
              </div>
              <div className="flex items-center">
                <InputNumber
                  min={1}
                  max={10}
                  value={settings.maxFailedTriesCount}
                  onChange={(value) => updateSettings({ maxFailedTriesCount: value || 1 })}
                  size="small"
                  className="w-full max-w-[75px] grow"
                />

                <Tooltip
                  className="cursor-pointer"
                  title={t('backups.logical.config.maxFailedTriesTooltip')}
                >
                  <InfoCircleOutlined className="ml-2" style={{ color: 'gray' }} />
                </Tooltip>
              </div>
            </div>
          )}
        </>
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
          disabled={!isUnsaved || !isAllFieldsFilled}
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
              const hadExistingStorage = !!backupConfig?.storageId;
              await loadStorages();
              updateBackupConfig({ storageId: createdStorage.id });
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
          onConfirm={() => {
            setIsShowWarn(false);
          }}
          onDecline={() => {
            setIsShowWarn(false);
          }}
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
