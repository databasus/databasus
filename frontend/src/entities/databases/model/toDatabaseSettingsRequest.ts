import { toOptionalValue } from '../../../shared/lib';
import type { DatabaseSettings } from './DatabaseSettings';
import { DatabaseType } from './DatabaseType';
import { toSshTunnelRequest } from './sshtunnel/toSshTunnelRequest';

// The settings as a request sends them: an empty secret is left out, so a saved database keeps
// the one it stored while the resource address stays the stored one, and the values only the server sets are dropped.
export const toDatabaseSettingsRequest = (settings: DatabaseSettings): DatabaseSettings => {
  const sshTunnel = toSshTunnelRequest(settings.sshTunnel);

  switch (settings.type) {
    case DatabaseType.POSTGRES_LOGICAL:
      return {
        ...settings,
        password: toOptionalValue(settings.password?.trim()),
        database: toOptionalValue(settings.database),
        sslClientCert: toOptionalValue(settings.sslClientCert),
        sslClientKey: toOptionalValue(settings.sslClientKey),
        sslRootCert: toOptionalValue(settings.sslRootCert),
        sshTunnel,
      };
    case DatabaseType.POSTGRES_PHYSICAL: {
      const { systemIdentifier: _, walSegmentSizeBytes: __, ...editableSettings } = settings;

      return {
        ...editableSettings,
        password: toOptionalValue(settings.password?.trim()),
        sslClientCert: toOptionalValue(settings.sslClientCert),
        sslClientKey: toOptionalValue(settings.sslClientKey),
        sslRootCert: toOptionalValue(settings.sslRootCert),
        sshTunnel,
      };
    }
    case DatabaseType.MYSQL:
    case DatabaseType.MARIADB: {
      const { privileges: _, ...editableSettings } = settings;

      return {
        ...editableSettings,
        password: toOptionalValue(settings.password?.trim()),
        database: toOptionalValue(settings.database),
        sshTunnel,
      };
    }
    case DatabaseType.MONGODB:
      return {
        ...settings,
        password: toOptionalValue(settings.password?.trim()),
        sshTunnel,
      };
  }
};
