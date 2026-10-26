import type { ApiComponents } from '../../../../../shared/api';

// The class of a failed connection test; all user-facing copy lives in the frontend.
export type ConnectionErrorCode = ApiComponents['schemas']['DatabaseConnectionFailure'];

export const ConnectionErrorCode = {
  ConnectionFailed: 'CONNECTION_FAILED',
  PgHbaNoEntry: 'PG_HBA_NO_ENTRY',
  BadCredentials: 'BAD_CREDENTIALS',
  NoReplicationPrivilege: 'NO_REPLICATION_PRIVILEGE',
  WalLevelInvalid: 'WAL_LEVEL_INVALID',
  NoWalSenders: 'NO_WAL_SENDERS',
  NoReplicationSlots: 'NO_REPLICATION_SLOTS',
  WalSummaryDisabled: 'WAL_SUMMARY_DISABLED',
  NoWalSwitchPrivilege: 'NO_WAL_SWITCH_PRIVILEGE',
  CustomTablespaces: 'CUSTOM_TABLESPACES',
  SystemIdentifierMismatch: 'SYSTEM_IDENTIFIER_MISMATCH',
} as const satisfies Record<string, ConnectionErrorCode>;
