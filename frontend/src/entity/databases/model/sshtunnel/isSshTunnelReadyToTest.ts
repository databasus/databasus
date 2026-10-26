import { SshTunnelAuthType } from './SshTunnelAuthType';
import type { SshTunnelSettings } from './SshTunnelSettings';

/**
 * Secrets are only required when creating: an existing database keeps the ones already stored, and
 * the edit form never receives them back from the API.
 */
export function isSshTunnelReadyToTest(
  sshTunnel: SshTunnelSettings | undefined,
  hasStoredSecrets: boolean,
): boolean {
  if (!sshTunnel?.isEnabled) return true;

  if (!sshTunnel.host) return false;
  if (!sshTunnel.port) return false;
  if (!sshTunnel.username) return false;

  if (hasStoredSecrets) return true;

  return sshTunnel.authType === SshTunnelAuthType.PASSWORD
    ? !!sshTunnel.password
    : !!sshTunnel.privateKey;
}
