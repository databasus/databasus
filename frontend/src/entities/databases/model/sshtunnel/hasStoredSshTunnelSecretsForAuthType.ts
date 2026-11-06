import type { SshTunnelAuthType } from '@/entities/databases/model/sshtunnel/SshTunnelAuthType';
import type { SshTunnelSettings } from '@/entities/databases/model/sshtunnel/SshTunnelSettings';

/**
 * Answers from the saved database, never from the edited copy: a database that never had a tunnel
 * would otherwise show the masked placeholder the moment the checkbox is ticked. The stored secret
 * only counts for the auth type it was saved under - switching to the other one needs a new secret.
 */
export function hasStoredSshTunnelSecretsForAuthType(
  savedSshTunnel: SshTunnelSettings | undefined,
  editedAuthType: SshTunnelAuthType | undefined,
): boolean {
  return !!savedSshTunnel?.isEnabled && savedSshTunnel.authType === editedAuthType;
}
