import { RestoreStatus } from '@/entities/restores/model/RestoreStatus';
import type { TranslationKey } from '@/shared/i18n';

export const RESTORE_STATUS_LABEL_KEYS: Record<RestoreStatus, TranslationKey> = {
  [RestoreStatus.IN_PROGRESS]: 'status.restore.inProgress',
  [RestoreStatus.COMPLETED]: 'status.restore.completed',
  [RestoreStatus.FAILED]: 'status.restore.failed',
  [RestoreStatus.CANCELED]: 'status.restore.canceled',
};
