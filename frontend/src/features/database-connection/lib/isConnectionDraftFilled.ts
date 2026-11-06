import {
  type DatabaseConnectionDraft,
  DatabaseType,
  hasStoredSshTunnelSecretsForAuthType,
  isSshTunnelReadyToTest,
} from '@/entities/databases';
import { isConnectionAddressChanged } from '@/features/database-connection/lib/isConnectionAddressChanged';
import { isSshTunnelAddressChanged } from '@/features/database-connection/lib/isSshTunnelAddressChanged';

const hasStoredSshSecrets = (draft: DatabaseConnectionDraft): boolean =>
  draft.scenario === 'edit' &&
  !isSshTunnelAddressChanged(draft.settings.sshTunnel, draft.savedSettings.sshTunnel) &&
  hasStoredSshTunnelSecretsForAuthType(
    draft.savedSettings.sshTunnel,
    draft.settings.sshTunnel?.authType,
  );

const isClientKeyRequiredAgain = (
  settings: { sslClientCert?: string; sslClientKey?: string },
  isAddressChanged: boolean,
): boolean => isAddressChanged && !!settings.sslClientCert && !settings.sslClientKey;

export const isConnectionDraftFilled = (draft: DatabaseConnectionDraft): boolean => {
  const { settings } = draft;
  const isAddressChanged = isConnectionAddressChanged(draft);

  if (!settings.host || !settings.username) return false;
  if ((draft.scenario !== 'edit' || isAddressChanged) && !settings.password?.trim()) return false;
  if (!isSshTunnelReadyToTest(settings.sshTunnel, hasStoredSshSecrets(draft))) return false;

  switch (settings.type) {
    case DatabaseType.POSTGRES_PHYSICAL:
      return !!settings.port && !isClientKeyRequiredAgain(settings, isAddressChanged);
    case DatabaseType.MONGODB:
      return (!!settings.isSrv || !!settings.port) && !!settings.database;
    case DatabaseType.POSTGRES_LOGICAL:
      return (
        !!settings.port &&
        !!settings.database &&
        !isClientKeyRequiredAgain(settings, isAddressChanged)
      );
    case DatabaseType.MYSQL:
    case DatabaseType.MARIADB:
      return !!settings.port && !!settings.database;
  }
};
