import type { LocalizedText, TranslationKey } from '../../../../../shared/i18n';
import { quotePostgresqlIdentifier } from '../quotePostgresqlIdentifier';
import { ConnectionErrorCode } from './ConnectionErrorCode';

// A note is one dictionary sentence rendered through Trans, with <bold> marking the emphasized
// words. Commands are technical strings and stay literal in every language.
export type PhysicalConnectionErrorStep =
  | { type: 'command'; command: string }
  | { type: 'note'; key: TranslationKey };

export interface PhysicalConnectionErrorContent {
  titleKey: TranslationKey;
  summaryKey: TranslationKey;
  buildSteps?: (ctx: { username: string }) => PhysicalConnectionErrorStep[];
  // Note about managed PostgreSQL (RDS / Azure / GCP). Rendered below the steps, since on managed
  // providers the commands do not apply and the operator uses the provider console instead.
  buildManagedNote?: (ctx: { username: string }) => LocalizedText;
}

// pg_hba.conf accepts the literal "all" in the address field to match every host. We use it because
// the exact IP Databasus connects from lives only in the server-side rejection, which the backend no
// longer forwards - the operator can narrow it to a CIDR range afterwards if they want.
// eslint-disable-next-line i18next/no-literal-string -- pg_hba.conf address keyword
const REPLICATION_HBA_ADDRESS = 'all';

export const physicalConnectionErrorContent: Record<
  ConnectionErrorCode,
  PhysicalConnectionErrorContent
> = {
  [ConnectionErrorCode.PgHbaNoEntry]: {
    titleKey: 'databases.physicalConnectionErrors.pgHbaNoEntry.title',
    summaryKey: 'databases.physicalConnectionErrors.pgHbaNoEntry.summary',
    buildSteps: () => [
      { type: 'note', key: 'databases.physicalConnectionErrors.pgHbaNoEntry.addLineNote' },
      {
        type: 'command',
        // eslint-disable-next-line i18next/no-literal-string -- a command, shown verbatim in every language
        command: `host    replication    all    ${REPLICATION_HBA_ADDRESS}    scram-sha-256`,
      },
      { type: 'note', key: 'databases.physicalConnectionErrors.pgHbaNoEntry.reloadNote' },
    ],
    buildManagedNote: () => ({
      key: 'databases.physicalConnectionErrors.pgHbaNoEntry.managedNote',
    }),
  },
  [ConnectionErrorCode.BadCredentials]: {
    titleKey: 'databases.physicalConnectionErrors.badCredentials.title',
    summaryKey: 'databases.physicalConnectionErrors.badCredentials.summary',
  },
  [ConnectionErrorCode.NoReplicationPrivilege]: {
    titleKey: 'databases.physicalConnectionErrors.noReplicationPrivilege.title',
    summaryKey: 'databases.physicalConnectionErrors.noReplicationPrivilege.summary',
    buildSteps: ({ username }) => [
      { type: 'note', key: 'databases.physicalConnectionErrors.grantAsSuperuserNote' },
      {
        type: 'command',
        // eslint-disable-next-line i18next/no-literal-string -- a command, shown verbatim in every language
        command: `ALTER ROLE ${quotePostgresqlIdentifier(username)} REPLICATION;`,
      },
    ],
    buildManagedNote: ({ username }) => ({
      key: 'databases.physicalConnectionErrors.noReplicationPrivilege.managedNote',
      params: { username: quotePostgresqlIdentifier(username) },
    }),
  },
  [ConnectionErrorCode.NoWalSwitchPrivilege]: {
    titleKey: 'databases.physicalConnectionErrors.noWalSwitchPrivilege.title',
    summaryKey: 'databases.physicalConnectionErrors.noWalSwitchPrivilege.summary',
    buildSteps: ({ username }) => [
      { type: 'note', key: 'databases.physicalConnectionErrors.grantAsSuperuserNote' },
      {
        type: 'command',
        // eslint-disable-next-line i18next/no-literal-string -- a command, shown verbatim in every language
        command: `GRANT EXECUTE ON FUNCTION pg_switch_wal() TO ${quotePostgresqlIdentifier(username)};`,
      },
      {
        type: 'note',
        key: 'databases.physicalConnectionErrors.noWalSwitchPrivilege.switchBackupTypeNote',
      },
    ],
    buildManagedNote: () => ({
      key: 'databases.physicalConnectionErrors.noWalSwitchPrivilege.managedNote',
    }),
  },
  [ConnectionErrorCode.WalLevelInvalid]: {
    titleKey: 'databases.physicalConnectionErrors.walLevelInvalid.title',
    summaryKey: 'databases.physicalConnectionErrors.walLevelInvalid.summary',
    buildSteps: () => [
      { type: 'note', key: 'databases.physicalConnectionErrors.walLevelInvalid.applyNote' },
      // eslint-disable-next-line i18next/no-literal-string -- a command, shown verbatim in every language
      { type: 'command', command: 'ALTER SYSTEM SET wal_level = replica;' },
    ],
    buildManagedNote: () => ({
      key: 'databases.physicalConnectionErrors.walLevelInvalid.managedNote',
    }),
  },
  [ConnectionErrorCode.NoWalSenders]: {
    titleKey: 'databases.physicalConnectionErrors.noWalSenders.title',
    summaryKey: 'databases.physicalConnectionErrors.noWalSenders.summary',
    buildSteps: () => [
      { type: 'note', key: 'databases.physicalConnectionErrors.applyAndRestartNote' },
      // eslint-disable-next-line i18next/no-literal-string -- a command, shown verbatim in every language
      { type: 'command', command: 'ALTER SYSTEM SET max_wal_senders = 10;' },
    ],
    buildManagedNote: () => ({
      key: 'databases.physicalConnectionErrors.noWalSenders.managedNote',
    }),
  },
  [ConnectionErrorCode.NoReplicationSlots]: {
    titleKey: 'databases.physicalConnectionErrors.noReplicationSlots.title',
    summaryKey: 'databases.physicalConnectionErrors.noReplicationSlots.summary',
    buildSteps: () => [
      { type: 'note', key: 'databases.physicalConnectionErrors.applyAndRestartNote' },
      // eslint-disable-next-line i18next/no-literal-string -- a command, shown verbatim in every language
      { type: 'command', command: 'ALTER SYSTEM SET max_replication_slots = 10;' },
    ],
    buildManagedNote: () => ({
      key: 'databases.physicalConnectionErrors.noReplicationSlots.managedNote',
    }),
  },
  [ConnectionErrorCode.WalSummaryDisabled]: {
    titleKey: 'databases.physicalConnectionErrors.walSummaryDisabled.title',
    summaryKey: 'databases.physicalConnectionErrors.walSummaryDisabled.summary',
    buildSteps: () => [
      { type: 'note', key: 'databases.physicalConnectionErrors.walSummaryDisabled.applyNote' },
      // eslint-disable-next-line i18next/no-literal-string -- a command, shown verbatim in every language
      { type: 'command', command: 'ALTER SYSTEM SET summarize_wal = on;' },
      { type: 'note', key: 'databases.physicalConnectionErrors.walSummaryDisabled.thenNote' },
      // eslint-disable-next-line i18next/no-literal-string -- a command, shown verbatim in every language
      { type: 'command', command: 'SELECT pg_reload_conf();' },
    ],
    buildManagedNote: () => ({
      key: 'databases.physicalConnectionErrors.walSummaryDisabled.managedNote',
    }),
  },
  [ConnectionErrorCode.CustomTablespaces]: {
    titleKey: 'databases.physicalConnectionErrors.customTablespaces.title',
    summaryKey: 'databases.physicalConnectionErrors.customTablespaces.summary',
  },
  [ConnectionErrorCode.SystemIdentifierMismatch]: {
    titleKey: 'databases.physicalConnectionErrors.systemIdentifierMismatch.title',
    summaryKey: 'databases.physicalConnectionErrors.systemIdentifierMismatch.summary',
  },
  [ConnectionErrorCode.ConnectionFailed]: {
    titleKey: 'databases.physicalConnectionErrors.connectionFailed.title',
    summaryKey: 'databases.physicalConnectionErrors.connectionFailed.summary',
  },
};
