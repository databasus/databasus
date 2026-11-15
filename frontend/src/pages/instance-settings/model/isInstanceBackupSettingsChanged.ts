import type { InstanceBackupSettings } from '@/entities/instance-backups';
import { isScheduleChanged } from '@/entities/schedules';
import { haveSameDestinations } from '@/pages/instance-settings/model/haveSameDestinations';

interface SettingsComparison {
  acknowledged: InstanceBackupSettings;
  shown: InstanceBackupSettings;
}

export const isInstanceBackupSettingsChanged = ({
  acknowledged,
  shown,
}: SettingsComparison): boolean =>
  isScheduleChanged({ acknowledged: acknowledged.schedule, shown: shown.schedule }) ||
  acknowledged.retentionDays !== shown.retentionDays ||
  !haveSameDestinations(acknowledged.storageIds, shown.storageIds);
