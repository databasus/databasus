import type { ApiComponents } from '@/shared/api';
import type { TranslationKey } from '@/shared/i18n';

export type PhysicalBackupStatus = ApiComponents['schemas']['PhysicalBackupStatus'];

export const PhysicalBackupStatus = {
  IN_PROGRESS: 'IN_PROGRESS',
  COMPLETED: 'COMPLETED',
  ERROR: 'ERROR',
  CHAIN_BROKEN: 'CHAIN_BROKEN',
  CANCELED: 'CANCELED',
} as const satisfies Record<string, PhysicalBackupStatus>;

export const PHYSICAL_BACKUP_STATUS_LABEL_KEYS: Record<PhysicalBackupStatus, TranslationKey> = {
  [PhysicalBackupStatus.COMPLETED]: 'status.physicalBackup.completed',
  [PhysicalBackupStatus.IN_PROGRESS]: 'status.physicalBackup.inProgress',
  [PhysicalBackupStatus.ERROR]: 'status.physicalBackup.error',
  [PhysicalBackupStatus.CHAIN_BROKEN]: 'status.physicalBackup.chainBroken',
  [PhysicalBackupStatus.CANCELED]: 'status.physicalBackup.canceled',
};
