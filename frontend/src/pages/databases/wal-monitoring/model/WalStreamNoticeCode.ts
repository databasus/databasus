import type { ApiComponents } from '@/shared/api';
import type { TranslationKey } from '@/shared/i18n';

export type WalStreamNoticeCode = ApiComponents['schemas']['WalStreamNoticeCode'];

export const WalStreamNoticeCode = {
  NO_SLOT_WAL_LIMIT: 'NO_SLOT_WAL_LIMIT',
  SLOT_KEEPS_EXTRA_WAL_STEADILY: 'SLOT_KEEPS_EXTRA_WAL_STEADILY',
  NO_FULL_BACKUP_YET: 'NO_FULL_BACKUP_YET',
} as const satisfies Record<string, WalStreamNoticeCode>;

export const WAL_STREAM_NOTICE_CODE_LABEL_KEYS: Record<WalStreamNoticeCode, TranslationKey> = {
  [WalStreamNoticeCode.NO_SLOT_WAL_LIMIT]:
    'backups.physical.monitoring.notices.noSlotWalLimit.title',
  [WalStreamNoticeCode.SLOT_KEEPS_EXTRA_WAL_STEADILY]:
    'backups.physical.monitoring.notices.slotKeepsExtraWalSteadily.title',
  [WalStreamNoticeCode.NO_FULL_BACKUP_YET]:
    'backups.physical.monitoring.notices.noFullBackupYet.title',
};
