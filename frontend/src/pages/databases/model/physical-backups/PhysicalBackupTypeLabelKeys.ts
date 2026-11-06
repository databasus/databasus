import { PhysicalBackupType } from '@/pages/databases/model/physical-backups/PhysicalBackupType';
import type { TranslationKey } from '@/shared/i18n';

export const PHYSICAL_BACKUP_TYPE_LABEL_KEYS: Record<PhysicalBackupType, TranslationKey> = {
  [PhysicalBackupType.FULL]: 'backups.physical.types.full',
  [PhysicalBackupType.INCREMENTAL]: 'backups.physical.types.incremental',
  [PhysicalBackupType.WAL]: 'backups.physical.types.wal',
};
