import { toOptionalValue } from '../../../../shared/lib';
import type { SshTunnelSettings } from './SshTunnelSettings';

// A disabled tunnel sends nothing but the switch, and an empty secret is left out so a saved
// database keeps the one it stored while the tunnel's address stays the stored one.
export const toSshTunnelRequest = (sshTunnel: SshTunnelSettings): SshTunnelSettings => {
  if (!sshTunnel.isEnabled) {
    return { isEnabled: false };
  }

  return {
    isEnabled: true,
    host: toOptionalValue(sshTunnel.host),
    port: sshTunnel.port,
    username: toOptionalValue(sshTunnel.username),
    authType: sshTunnel.authType,
    password: toOptionalValue(sshTunnel.password),
    privateKey: toOptionalValue(sshTunnel.privateKey),
    privateKeyPassphrase: toOptionalValue(sshTunnel.privateKeyPassphrase),
  };
};
