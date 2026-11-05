import { SshTunnelAuthType } from './SshTunnelAuthType';
import type { SshTunnelSettings } from './SshTunnelSettings';

export const DEFAULT_SSH_PORT = 22;

export function createEmptySshTunnelSettings(): SshTunnelSettings {
  return {
    isEnabled: false,
    host: '',
    port: DEFAULT_SSH_PORT,
    username: '',
    authType: SshTunnelAuthType.PASSWORD,
    password: '',
    privateKey: '',
    privateKeyPassphrase: '',
  };
}
