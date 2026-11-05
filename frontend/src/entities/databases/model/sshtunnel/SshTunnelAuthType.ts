import type { ApiComponents } from '../../../../shared/api';

export type SshTunnelAuthType = ApiComponents['schemas']['SshAuthType'];

export const SshTunnelAuthType = {
  PASSWORD: 'PASSWORD',
  PRIVATE_KEY: 'PRIVATE_KEY',
} as const satisfies Record<string, SshTunnelAuthType>;
