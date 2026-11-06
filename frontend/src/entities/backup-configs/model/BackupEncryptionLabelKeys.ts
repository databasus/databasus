import { BackupEncryption } from '@/entities/backup-configs/model/BackupEncryption';
import type { TranslationKey } from '@/shared/i18n';

export const BACKUP_ENCRYPTION_LABEL_KEYS: Record<BackupEncryption, TranslationKey> = {
  [BackupEncryption.NONE]: 'backups.encryption.none',
  [BackupEncryption.ENCRYPTED]: 'backups.encryption.encrypted',
};
