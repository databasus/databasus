import type { InstanceBackupDestinationHealth } from '@/entities/instance-backups';

export const getNewestSuccessfulSnapshotAt = (
  destinations: InstanceBackupDestinationHealth[],
): string | undefined => {
  const snapshotTimes = destinations
    .map((destination) => destination.lastSuccessfulSnapshotAt)
    .filter((snapshotAt): snapshotAt is string => !!snapshotAt);

  if (snapshotTimes.length === 0) return undefined;

  return snapshotTimes.reduce((newest, snapshotAt) =>
    Date.parse(snapshotAt) > Date.parse(newest) ? snapshotAt : newest,
  );
};
