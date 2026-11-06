import { describe, expect, it } from 'vitest';

import { disableSrvWhenTunneled } from '@/entities/databases/model/mongodb/disableSrvWhenTunneled';
import type { MongodbSettings } from '@/entities/databases/model/ProviderDatabaseSettings';
import { SshTunnelAuthType } from '@/entities/databases/model/sshtunnel/SshTunnelAuthType';
import type { SshTunnelSettings } from '@/entities/databases/model/sshtunnel/SshTunnelSettings';

const enabledTunnel = (): SshTunnelSettings => ({
  isEnabled: true,
  authType: SshTunnelAuthType.PASSWORD,
  host: 'bastion.example.com',
  port: 22,
  username: 'tunneluser',
  password: 'tunnelpassword',
  privateKey: '',
  privateKeyPassphrase: '',
});

const srvDatabase = (
  sshTunnel: SshTunnelSettings = { ...enabledTunnel(), isEnabled: false },
): MongodbSettings => ({
  type: 'MONGODB',
  version: '7',
  host: 'cluster0.example.mongodb.net',
  port: 27017,
  username: 'testuser',
  password: 'testpassword',
  database: 'testdb',
  authDatabase: 'admin',
  isHttps: false,
  isSrv: true,
  isDirectConnection: false,
  cpuCount: 1,
  sshTunnel,
  excludeCollections: [],
});

describe('disableSrvWhenTunneled', () => {
  it('turns SRV off when the tunnel is enabled', () => {
    expect(disableSrvWhenTunneled(srvDatabase(enabledTunnel())).isSrv).toBe(false);
  });

  it('leaves SRV alone when there is no tunnel', () => {
    expect(disableSrvWhenTunneled(srvDatabase()).isSrv).toBe(true);
    expect(
      disableSrvWhenTunneled(srvDatabase({ ...enabledTunnel(), isEnabled: false })).isSrv,
    ).toBe(true);
  });

  it('does not touch direct connection, which the backend forces per operation', () => {
    expect(disableSrvWhenTunneled(srvDatabase(enabledTunnel())).isDirectConnection).toBe(false);
  });

  it('returns the same object when nothing changes', () => {
    const database = srvDatabase();

    expect(disableSrvWhenTunneled(database)).toBe(database);
  });
});
