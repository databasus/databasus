import type { DatabaseSettings } from '@/entities/databases';

// Behind a bastion a loopback address names the database as the bastion sees it, which is the
// documented setup rather than the mistake the localhost hint warns about.
export const isLocalhostConnection = (settings: DatabaseSettings): boolean =>
  !settings.sshTunnel?.isEnabled &&
  (settings.host.includes('localhost') || settings.host.includes('127.0.0.1'));
