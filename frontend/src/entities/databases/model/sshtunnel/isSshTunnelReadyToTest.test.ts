import { describe, expect, it } from 'vitest';

import { createEmptySshTunnelSettings } from '@/entities/databases/model/sshtunnel/createEmptySshTunnelSettings';
import { isSshTunnelReadyToTest } from '@/entities/databases/model/sshtunnel/isSshTunnelReadyToTest';
import { SshTunnelAuthType } from '@/entities/databases/model/sshtunnel/SshTunnelAuthType';
import type { SshTunnelSettings } from '@/entities/databases/model/sshtunnel/SshTunnelSettings';

function enabledTunnel(): SshTunnelSettings {
  return {
    ...createEmptySshTunnelSettings(),
    isEnabled: true,
    host: 'bastion.example.com',
    username: 'tunneluser',
    password: 'tunnelpassword',
  };
}

describe('isSshTunnelReadyToTest', () => {
  it('accepts a missing config, since the tunnel is optional', () => {
    expect(isSshTunnelReadyToTest(undefined, false)).toBe(true);
  });

  it('ignores an incomplete config while the tunnel is off', () => {
    expect(isSshTunnelReadyToTest(createEmptySshTunnelSettings(), false)).toBe(true);
  });

  it('accepts a complete config with a password', () => {
    expect(isSshTunnelReadyToTest(enabledTunnel(), false)).toBe(true);
  });

  it('accepts a private key when that is the chosen auth type', () => {
    expect(
      isSshTunnelReadyToTest(
        {
          ...enabledTunnel(),
          authType: SshTunnelAuthType.PRIVATE_KEY,
          password: '',
          privateKey: 'key',
        },
        false,
      ),
    ).toBe(true);
  });

  it('rejects a private key while the chosen auth type is a password', () => {
    expect(
      isSshTunnelReadyToTest({ ...enabledTunnel(), password: '', privateKey: 'key' }, false),
    ).toBe(false);
  });

  it('rejects a password while the chosen auth type is a private key', () => {
    expect(
      isSshTunnelReadyToTest(
        { ...enabledTunnel(), authType: SshTunnelAuthType.PRIVATE_KEY },
        false,
      ),
    ).toBe(false);
  });

  it.each(['host', 'username'] as const)('rejects a missing %s', (field) => {
    expect(isSshTunnelReadyToTest({ ...enabledTunnel(), [field]: '' }, false)).toBe(false);
  });

  it('rejects a missing port', () => {
    expect(isSshTunnelReadyToTest({ ...enabledTunnel(), port: 0 }, false)).toBe(false);
  });

  it('rejects a missing secret when creating', () => {
    expect(isSshTunnelReadyToTest({ ...enabledTunnel(), password: '' }, false)).toBe(false);
  });

  it('accepts a missing secret when one is already stored', () => {
    expect(isSshTunnelReadyToTest({ ...enabledTunnel(), password: '' }, true)).toBe(true);
  });
});
