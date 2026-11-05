import type { TranslationKey } from '../../../../shared/i18n';
import { PhysicalBackupStatus } from './PhysicalBackupStatus';

export const PHYSICAL_BACKUP_STATUS_LABEL_KEYS: Record<PhysicalBackupStatus, TranslationKey> = {
  [PhysicalBackupStatus.COMPLETED]: 'status.physicalBackup.completed',
  [PhysicalBackupStatus.IN_PROGRESS]: 'status.physicalBackup.inProgress',
  [PhysicalBackupStatus.ERROR]: 'status.physicalBackup.error',
  [PhysicalBackupStatus.CHAIN_BROKEN]: 'status.physicalBackup.chainBroken',
  [PhysicalBackupStatus.CANCELED]: 'status.physicalBackup.canceled',
};
