import { describe, expect, it } from 'vitest';

import { createEmptySshTunnelSettings } from './createEmptySshTunnelSettings';
import { setSshTunnelAuthTypeAndClearUnusedSecrets } from './setSshTunnelAuthTypeAndClearUnusedSecrets';
import { SshTunnelAuthType } from './SshTunnelAuthType';
import type { SshTunnelSettings } from './SshTunnelSettings';

function tunnelWithBothSecrets(): SshTunnelSettings {
  return {
    ...createEmptySshTunnelSettings(),
    isEnabled: true,
    host: 'bastion.example.com',
    username: 'tunneluser',
    password: 'tunnelpassword',
    privateKey: 'tunnelprivatekey',
    privateKeyPassphrase: 'tunnelpassphrase',
  };
}

describe('setSshTunnelAuthTypeAndClearUnusedSecrets', () => {
  it('clears the private key and its passphrase when switching to a password', () => {
    const passwordTunnel = setSshTunnelAuthTypeAndClearUnusedSecrets(
      tunnelWithBothSecrets(),
      SshTunnelAuthType.PASSWORD,
    );

    expect(passwordTunnel.authType).toBe(SshTunnelAuthType.PASSWORD);
    expect(passwordTunnel.privateKey).toBe('');
    expect(passwordTunnel.privateKeyPassphrase).toBe('');
    expect(passwordTunnel.password).toBe('tunnelpassword');
  });

  it('clears the password when switching to a private key', () => {
    const privateKeyTunnel = setSshTunnelAuthTypeAndClearUnusedSecrets(
      tunnelWithBothSecrets(),
      SshTunnelAuthType.PRIVATE_KEY,
    );

    expect(privateKeyTunnel.authType).toBe(SshTunnelAuthType.PRIVATE_KEY);
    expect(privateKeyTunnel.password).toBe('');
    expect(privateKeyTunnel.privateKey).toBe('tunnelprivatekey');
    expect(privateKeyTunnel.privateKeyPassphrase).toBe('tunnelpassphrase');
  });

  it('keeps the connection fields untouched', () => {
    const originalTunnel = tunnelWithBothSecrets();

    const privateKeyTunnel = setSshTunnelAuthTypeAndClearUnusedSecrets(
      originalTunnel,
      SshTunnelAuthType.PRIVATE_KEY,
    );

    expect(privateKeyTunnel.host).toBe(originalTunnel.host);
    expect(privateKeyTunnel.port).toBe(originalTunnel.port);
    expect(privateKeyTunnel.username).toBe(originalTunnel.username);
    expect(privateKeyTunnel.isEnabled).toBe(true);
  });
});
