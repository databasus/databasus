import type { TranslationKey } from '../../../../shared/i18n';
import type { StorageSwitch } from './StorageSwitch';

export const STORAGE_SWITCH_STATUS_LABEL_KEYS: Record<StorageSwitch['status'], TranslationKey> = {
  CLEANUP_PENDING: 'backups.storageSwitch.pending',
  COMPLETED: 'backups.storageSwitch.completed',
  COMPLETED_WITH_REMAINDERS: 'backups.storageSwitch.remainders',
};
