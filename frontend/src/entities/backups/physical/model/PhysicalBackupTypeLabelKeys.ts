import type { TranslationKey } from '../../../../shared/i18n';
import { PhysicalBackupType } from './PhysicalBackupType';

export const PHYSICAL_BACKUP_TYPE_LABEL_KEYS: Record<PhysicalBackupType, TranslationKey> = {
  [PhysicalBackupType.FULL]: 'backups.physical.types.full',
  [PhysicalBackupType.INCREMENTAL]: 'backups.physical.types.incremental',
  [PhysicalBackupType.WAL]: 'backups.physical.types.wal',
};
