import { PhysicalDatabaseBackupType } from '@/entities/databases/model/postgresql/physical/PhysicalDatabaseBackupType';
import type { TranslationKey } from '@/shared/i18n';

export const PHYSICAL_DATABASE_BACKUP_TYPE_LABEL_KEYS: Record<
  PhysicalDatabaseBackupType,
  TranslationKey
> = {
  [PhysicalDatabaseBackupType.FULL]: 'databases.physicalBackupTypes.full',
  [PhysicalDatabaseBackupType.FULL_INCREMENTAL]: 'databases.physicalBackupTypes.fullIncremental',
  [PhysicalDatabaseBackupType.FULL_INCREMENTAL_WAL_STREAM]:
    'databases.physicalBackupTypes.fullIncrementalWalStream',
};
