import { SshTunnelAuthType } from '@/entities/databases/model/sshtunnel/SshTunnelAuthType';
import type { TranslationKey } from '@/shared/i18n';

export const SSH_TUNNEL_AUTH_TYPE_LABEL_KEYS: Record<SshTunnelAuthType, TranslationKey> = {
  [SshTunnelAuthType.PASSWORD]: 'databases.sshTunnel.authTypes.password',
  [SshTunnelAuthType.PRIVATE_KEY]: 'databases.sshTunnel.authTypes.privateKey',
};
