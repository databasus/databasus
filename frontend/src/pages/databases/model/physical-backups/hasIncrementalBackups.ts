import { PhysicalDatabaseBackupType } from '../../../../entities/databases';

export const hasIncrementalBackups = (backupType: PhysicalDatabaseBackupType): boolean =>
  backupType !== PhysicalDatabaseBackupType.FULL;
