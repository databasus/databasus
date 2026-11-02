import {
  PostgresSslMode,
  type PostgresqlLogicalSettings,
  type PostgresqlPhysicalSettings,
} from '@/entities/databases';

import { stripHostScheme } from './stripHostScheme';

const IPV4_PATTERN = /^\d{1,3}(\.\d{1,3}){3}$/;

export const deriveSslModeFromHost = (rawHost: string): PostgresSslMode | null => {
  const trimmedHost = rawHost.trim().toLowerCase();

  if (trimmedHost.startsWith('https://')) return PostgresSslMode.Require;
  if (trimmedHost.startsWith('http://')) return PostgresSslMode.Disable;

  const bareHost = trimmedHost.split(':')[0];
  if (bareHost === 'localhost' || IPV4_PATTERN.test(bareHost)) return PostgresSslMode.Disable;

  return null;
};

const clearPostgresqlCertificates = <
  Settings extends PostgresqlLogicalSettings | PostgresqlPhysicalSettings,
>(
  settings: Settings,
): Settings => ({ ...settings, sslClientCert: '', sslClientKey: '', sslRootCert: '' });

export const applyPostgresqlSslMode = <
  Settings extends PostgresqlLogicalSettings | PostgresqlPhysicalSettings,
>(
  settings: Settings,
  sslMode: PostgresSslMode,
): Settings =>
  sslMode === PostgresSslMode.Disable
    ? { ...clearPostgresqlCertificates(settings), sslMode }
    : { ...settings, sslMode };

// A new database follows the host until the user chooses a mode; for a chosen mode only an
// explicit https:// host still turns SSL on.
export const applyPostgresqlHost = <
  Settings extends PostgresqlLogicalSettings | PostgresqlPhysicalSettings,
>(
  settings: Settings,
  rawHost: string,
  hasUserChosenSslMode: boolean,
): Settings => {
  const settingsWithHost = { ...settings, host: stripHostScheme(rawHost) };

  if (!hasUserChosenSslMode) {
    const derivedSslMode = deriveSslModeFromHost(rawHost);

    return derivedSslMode === null
      ? settingsWithHost
      : applyPostgresqlSslMode(settingsWithHost, derivedSslMode);
  }

  const isHttpsHost = rawHost.trim().toLowerCase().startsWith('https://');
  const isSslOff = (settings.sslMode ?? PostgresSslMode.Disable) === PostgresSslMode.Disable;

  return isHttpsHost && isSslOff
    ? applyPostgresqlSslMode(settingsWithHost, PostgresSslMode.Require)
    : settingsWithHost;
};
