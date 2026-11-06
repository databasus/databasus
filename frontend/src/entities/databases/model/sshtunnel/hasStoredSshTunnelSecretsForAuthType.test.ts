import { describe, expect, it } from 'vitest';

import { hasStoredSshTunnelSecretsForAuthType } from '@/entities/databases/model/sshtunnel/hasStoredSshTunnelSecretsForAuthType';
import { SshTunnelAuthType } from '@/entities/databases/model/sshtunnel/SshTunnelAuthType';
import type { SshTunnelSettings } from '@/entities/databases/model/sshtunnel/SshTunnelSettings';

const enabledTunnel = (): SshTunnelSettings => ({
  isEnabled: true,
  authType: SshTunnelAuthType.PASSWORD,
  host: 'bastion.example.com',
  port: 22,
  username: 'tunneluser',
  password: '',
  privateKey: '',
  privateKeyPassphrase: '',
});

describe('hasStoredSshTunnelSecretsForAuthType', () => {
  it('is true for a saved database whose tunnel is enabled', () => {
    expect(hasStoredSshTunnelSecretsForAuthType(enabledTunnel(), SshTunnelAuthType.PASSWORD)).toBe(
      true,
    );
  });

  it('is false once the user picks the other auth type', () => {
    expect(
      hasStoredSshTunnelSecretsForAuthType(enabledTunnel(), SshTunnelAuthType.PRIVATE_KEY),
    ).toBe(false);
  });

  it('is false for a saved database that never had a tunnel', () => {
    expect(hasStoredSshTunnelSecretsForAuthType(undefined, SshTunnelAuthType.PASSWORD)).toBe(false);
    expect(
      hasStoredSshTunnelSecretsForAuthType(
        { ...enabledTunnel(), isEnabled: false },
        SshTunnelAuthType.PASSWORD,
      ),
    ).toBe(false);
  });
});
