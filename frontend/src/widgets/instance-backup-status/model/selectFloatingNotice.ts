import { InstanceBackupHealth, type InstanceBackupStatus } from '@/entities/instance-backups';

export type FloatingNotice =
  | { kind: 'incident' }
  | { kind: 'resuming' }
  | { kind: 'master-key' }
  | { kind: 'setup' };

interface Selection {
  status: InstanceBackupStatus;
  isMasterKeyOnboardingDeferred: boolean;
  now: Date;
}

// One notice at a time, the most urgent first. An open incident outranks a deferred setup
// reminder, so stopped protection shows even after the owner postponed setup.
export const selectFloatingNotice = ({
  status,
  isMasterKeyOnboardingDeferred,
  now,
}: Selection): FloatingNotice | undefined => {
  if (status.incident) return { kind: 'incident' };
  if (status.health === InstanceBackupHealth.RESUMING) return { kind: 'resuming' };
  if (!status.isMasterKeyAcknowledged) {
    return isMasterKeyOnboardingDeferred ? { kind: 'master-key' } : undefined;
  }

  const isSetupDeferred =
    !!status.reminder.deferredUntil && new Date(status.reminder.deferredUntil) > now;
  if (
    status.reminder.isEligible &&
    status.destinations.length === 0 &&
    status.health === InstanceBackupHealth.NOT_CONFIGURED &&
    !isSetupDeferred
  ) {
    return { kind: 'setup' };
  }

  return undefined;
};
