import {
  DatabaseType,
  type RestoreTargetDraft,
  createDefaultDatabaseSettings,
} from '@/entities/databases';

// The target is usually another server, so it starts as a new database of the backup's engine. A
// physical backup is restored by other means and has no target.
export const createRestoreTargetDraft = (type: DatabaseType): RestoreTargetDraft | undefined =>
  type === DatabaseType.POSTGRES_PHYSICAL
    ? undefined
    : {
        scenario: 'restore',
        settings: createDefaultDatabaseSettings(type),
        hasUserChosenSslMode: false,
      };
