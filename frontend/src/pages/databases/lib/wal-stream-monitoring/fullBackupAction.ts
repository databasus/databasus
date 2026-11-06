import { FullBackupAction } from '@/pages/databases/model/wal-stream-monitoring/FullBackupAction';
import { WalStreamHealthLevel } from '@/pages/databases/model/wal-stream-monitoring/WalStreamHealthLevel';
import { type WalStreamReason } from '@/pages/databases/model/wal-stream-monitoring/WalStreamReason';
import { WalStreamReasonCode } from '@/pages/databases/model/wal-stream-monitoring/WalStreamReasonCode';

interface FullBackupActionInput {
  level: WalStreamHealthLevel;
  isStateConfirmed: boolean;
  reasons: WalStreamReason[];
  latestRestorableAt: string | null;
  isFullBackupPending: boolean;
  isCanManageDBs: boolean;
  isRequestingFullBackup: boolean;
}

// Not streaming takes no backups at all, and collecting data lasts two minutes at most, so neither
// offers a full backup.
const LEVELS_OFFERING_FULL_BACKUP = new Set<WalStreamHealthLevel>([
  WalStreamHealthLevel.HEALTHY,
  WalStreamHealthLevel.NEEDS_ATTENTION,
  WalStreamHealthLevel.BROKEN,
]);

const REASONS_REPAIRED_BY_FULL_BACKUP = new Set<WalStreamReasonCode>([
  WalStreamReasonCode.WAL_CHAIN_BROKEN,
  WalStreamReasonCode.SLOT_MISSING,
]);

// Pending needs no state of its own: the backend stores the request before it answers, so the
// refresh after a successful request already reports the full backup as pending.
export const getFullBackupAction = (input: FullBackupActionInput): FullBackupAction => {
  if (!input.isCanManageDBs || !input.isStateConfirmed || input.isFullBackupPending) {
    return FullBackupAction.HIDDEN;
  }

  if (!LEVELS_OFFERING_FULL_BACKUP.has(input.level) || !isFullBackupTheFix(input)) {
    return FullBackupAction.HIDDEN;
  }

  return input.isRequestingFullBackup ? FullBackupAction.REQUESTING : FullBackupAction.AVAILABLE;
};

const isFullBackupTheFix = (input: FullBackupActionInput): boolean =>
  input.latestRestorableAt === null ||
  input.reasons.some((reason) => REASONS_REPAIRED_BY_FULL_BACKUP.has(reason.code));
