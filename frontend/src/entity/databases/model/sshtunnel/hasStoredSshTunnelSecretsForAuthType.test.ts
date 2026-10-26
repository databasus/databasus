import { describe, expect, it } from 'vitest';

import { hasStoredSshTunnelSecretsForAuthType } from './hasStoredSshTunnelSecretsForAuthType';
import { SshTunnelAuthType } from './SshTunnelAuthType';
import type { SshTunnelSettings } from './SshTunnelSettings';

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
    expect(
      hasStoredSshTunnelSecretsForAuthType(enabledTunnel(), SshTunnelAuthType.PASSWORD, 'db-1'),
    ).toBe(true);
  });

  it('is false once the user picks the other auth type', () => {
    expect(
      hasStoredSshTunnelSecretsForAuthType(enabledTunnel(), SshTunnelAuthType.PRIVATE_KEY, 'db-1'),
    ).toBe(false);
  });

  it('is false while the database is still being created', () => {
    expect(
      hasStoredSshTunnelSecretsForAuthType(enabledTunnel(), SshTunnelAuthType.PASSWORD, undefined),
    ).toBe(false);
  });

  it('is false for a saved database that never had a tunnel', () => {
    expect(
      hasStoredSshTunnelSecretsForAuthType(undefined, SshTunnelAuthType.PASSWORD, 'db-1'),
    ).toBe(false);
    expect(
      hasStoredSshTunnelSecretsForAuthType(
        { ...enabledTunnel(), isEnabled: false },
        SshTunnelAuthType.PASSWORD,
        'db-1',
      ),
    ).toBe(false);
  });
});
