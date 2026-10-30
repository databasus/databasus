import type { ApiComponents } from '../../../../shared/api';

export type WalStreamNoticeCode = ApiComponents['schemas']['WalStreamNoticeCode'];

export const WalStreamNoticeCode = {
  NO_SLOT_WAL_LIMIT: 'NO_SLOT_WAL_LIMIT',
  SLOT_KEEPS_EXTRA_WAL_STEADILY: 'SLOT_KEEPS_EXTRA_WAL_STEADILY',
  NO_FULL_BACKUP_YET: 'NO_FULL_BACKUP_YET',
} as const satisfies Record<string, WalStreamNoticeCode>;
