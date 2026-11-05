import { SshTunnelAuthType } from './SshTunnelAuthType';
import type { SshTunnelSettings } from './SshTunnelSettings';

/**
 * A saved database at an unchanged tunnel address keeps the secrets already stored, which the edit
 * form never receives back from the API. Every other tunnel needs them entered.
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
