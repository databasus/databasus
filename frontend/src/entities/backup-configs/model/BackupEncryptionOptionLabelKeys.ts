import { BackupEncryption } from '@/entities/backup-configs/model/BackupEncryption';
import type { TranslationKey } from '@/shared/i18n';

// The choice as offered in a config form ("Encrypt backup files"). BACKUP_ENCRYPTION_LABEL_KEYS holds
// the resulting state shown on a read-only config ("Enabled"); several languages word the two apart.
export const BACKUP_ENCRYPTION_OPTION_LABEL_KEYS: Record<BackupEncryption, TranslationKey> = {
  [BackupEncryption.NONE]: 'backups.encryptionOptions.none',
  [BackupEncryption.ENCRYPTED]: 'backups.encryptionOptions.encrypted',
};
