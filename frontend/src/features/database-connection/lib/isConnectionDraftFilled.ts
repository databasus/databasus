import {
  type DatabaseConnectionDraft,
  DatabaseType,
  hasStoredSshTunnelSecretsForAuthType,
  isSshTunnelReadyToTest,
} from '@/entities/databases';

const hasStoredSshSecrets = (draft: DatabaseConnectionDraft): boolean =>
  draft.scenario === 'edit' &&
  hasStoredSshTunnelSecretsForAuthType(
    draft.savedSettings.sshTunnel,
    draft.settings.sshTunnel?.authType,
  );

export const isConnectionDraftFilled = (draft: DatabaseConnectionDraft): boolean => {
  const { settings } = draft;

  if (!settings.host || !settings.username) return false;
  if (draft.scenario !== 'edit' && !settings.password?.trim()) return false;
  if (!isSshTunnelReadyToTest(settings.sshTunnel, hasStoredSshSecrets(draft))) return false;

  switch (settings.type) {
    case DatabaseType.POSTGRES_PHYSICAL:
      return !!settings.port;
    case DatabaseType.MONGODB:
      return (!!settings.isSrv || !!settings.port) && !!settings.database;
    case DatabaseType.POSTGRES_LOGICAL:
    case DatabaseType.MYSQL:
    case DatabaseType.MARIADB:
      return !!settings.port && !!settings.database;
  }
};
