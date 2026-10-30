import type { TranslationKey } from '../../../../shared/i18n';
import { BackupEncryption } from './BackupEncryption';

export const BACKUP_ENCRYPTION_LABEL_KEYS: Record<BackupEncryption, TranslationKey> = {
  [BackupEncryption.NONE]: 'backups.encryption.none',
  [BackupEncryption.ENCRYPTED]: 'backups.encryption.encrypted',
};
