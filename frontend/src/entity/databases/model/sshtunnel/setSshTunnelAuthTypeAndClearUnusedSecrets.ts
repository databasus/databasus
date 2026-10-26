import { SshTunnelAuthType } from './SshTunnelAuthType';
import type { SshTunnelSettings } from './SshTunnelSettings';

/**
 * Mirrors what the backend does on save, so the form never submits a secret belonging to the way
 * of logging in the user just abandoned.
 */
export function setSshTunnelAuthTypeAndClearUnusedSecrets(
  sshTunnel: SshTunnelSettings,
  authType: SshTunnelAuthType,
): SshTunnelSettings {
  if (authType === SshTunnelAuthType.PASSWORD) {
    return { ...sshTunnel, authType, privateKey: '', privateKeyPassphrase: '' };
  }

  return { ...sshTunnel, authType, password: '' };
}
