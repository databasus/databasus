import { type Database, DatabaseType, PhysicalDatabaseBackupType } from '@/entities/databases';

export const isMonitoringTabShown = (database: Database): boolean =>
  database.settings.type === DatabaseType.POSTGRES_PHYSICAL &&
  database.settings.backupType === PhysicalDatabaseBackupType.FULL_INCREMENTAL_WAL_STREAM;
