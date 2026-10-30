import type { TranslationKey } from '../../../../shared/i18n';
import { SshTunnelAuthType } from './SshTunnelAuthType';

export const SSH_TUNNEL_AUTH_TYPE_LABEL_KEYS: Record<SshTunnelAuthType, TranslationKey> = {
  [SshTunnelAuthType.PASSWORD]: 'databases.sshTunnel.authTypes.password',
  [SshTunnelAuthType.PRIVATE_KEY]: 'databases.sshTunnel.authTypes.privateKey',
};
