import type { ApiComponents } from '../../../../../shared/api';

// Database-level backup strategy for a physical PostgreSQL source. Distinct from
// the list-item PhysicalBackupType (FULL/INCREMENTAL/WAL) - this selects which
// cadences and retention modes the backup config may use:
//   FULL                        - standalone full backups only
//   FULL_INCREMENTAL            - full + incremental chains (requires summarize_wal=on)
//   FULL_INCREMENTAL_WAL_STREAM - the above + continuous WAL streaming (self-hosted only)
export type PhysicalDatabaseBackupType = ApiComponents['schemas']['PhysicalBackupType'];

export const PhysicalDatabaseBackupType = {
  FULL: 'FULL',
  FULL_INCREMENTAL: 'FULL_INCREMENTAL',
  FULL_INCREMENTAL_WAL_STREAM: 'FULL_INCREMENTAL_WAL_STREAM',
} as const satisfies Record<string, PhysicalDatabaseBackupType>;
