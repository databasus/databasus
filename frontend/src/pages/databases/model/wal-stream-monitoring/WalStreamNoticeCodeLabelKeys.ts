import type { TranslationKey } from '../../../../shared/i18n';
import { WalStreamNoticeCode } from './WalStreamNoticeCode';

export const WAL_STREAM_NOTICE_CODE_LABEL_KEYS: Record<WalStreamNoticeCode, TranslationKey> = {
  [WalStreamNoticeCode.NO_SLOT_WAL_LIMIT]:
    'backups.physical.monitoring.notices.noSlotWalLimit.title',
  [WalStreamNoticeCode.SLOT_KEEPS_EXTRA_WAL_STEADILY]:
    'backups.physical.monitoring.notices.slotKeepsExtraWalSteadily.title',
  [WalStreamNoticeCode.NO_FULL_BACKUP_YET]:
    'backups.physical.monitoring.notices.noFullBackupYet.title',
};
