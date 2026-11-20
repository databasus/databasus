import type { InstanceBackupSettings } from '@/entities/instance-backups';
import { isScheduleChanged } from '@/entities/schedules';
import { haveSameDestinations } from '@/pages/instance-settings/model/haveSameDestinations';

interface Acknowledgment {
  current: InstanceBackupSettings;
  submitted: InstanceBackupSettings;
  acknowledged: InstanceBackupSettings;
}

// A field edited again while the save was pending keeps the newer input; every other field takes
// the acknowledged value.
export const acknowledgeInstanceBackupSettings = ({
  current,
  submitted,
  acknowledged,
}: Acknowledgment): InstanceBackupSettings => ({
  schedule: isScheduleChanged({ acknowledged: submitted.schedule, shown: current.schedule })
    ? current.schedule
    : acknowledged.schedule,
  retentionDays:
    current.retentionDays === submitted.retentionDays
      ? acknowledged.retentionDays
      : current.retentionDays,
  storageIds: haveSameDestinations(submitted.storageIds, current.storageIds)
    ? acknowledged.storageIds
    : current.storageIds,
});
