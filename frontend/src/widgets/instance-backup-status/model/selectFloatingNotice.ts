import { InstanceBackupHealth, type InstanceBackupStatus } from '@/entities/instance-backups';

export type FloatingNotice = { kind: 'incident' } | { kind: 'resuming' } | { kind: 'setup' };

// One notice at a time, the most urgent first. The setup reminder cannot be put off, so it stays
// until a destination is selected; it waits only while the master key is unsaved.
export const selectFloatingNotice = (status: InstanceBackupStatus): FloatingNotice | undefined => {
  if (status.incident) return { kind: 'incident' };
  if (status.health === InstanceBackupHealth.RESUMING) return { kind: 'resuming' };
  if (!status.isMasterKeyAcknowledged) return undefined;

  if (
    status.reminder.isEligible &&
    status.destinations.length === 0 &&
    status.health === InstanceBackupHealth.NOT_CONFIGURED
  ) {
    return { kind: 'setup' };
  }

  return undefined;
};
