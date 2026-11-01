import { describe, expect, it } from 'vitest';

import {
  DatabaseType,
  type PostgresqlLogicalSettings,
  PostgresSslMode,
  createDefaultDatabaseSettings,
} from '@/entities/databases';

import {
  applyPostgresqlHost,
  applyPostgresqlSslMode,
  deriveSslModeFromHost,
} from './postgresqlHostSsl';

const settingsWithCertificates: PostgresqlLogicalSettings = {
  ...createDefaultDatabaseSettings(DatabaseType.POSTGRES_LOGICAL),
  sslMode: PostgresSslMode.Require,
  sslClientCert: 'client-cert',
  sslClientKey: 'client-key',
  sslRootCert: 'root-cert',
};

describe('deriveSslModeFromHost', () => {
  it.each([
    ['https://db.example.com', PostgresSslMode.Require],
    ['http://db.example.com', PostgresSslMode.Disable],
    ['localhost', PostgresSslMode.Disable],
    ['localhost:5432', PostgresSslMode.Disable],
    ['10.0.0.5', PostgresSslMode.Disable],
    ['db.example.com', null],
  ])('derives the mode of %s as %s', (host, sslMode) => {
    expect(deriveSslModeFromHost(host)).toBe(sslMode);
  });
});

describe('applyPostgresqlSslMode', () => {
  it('clears every certificate when SSL turns off', () => {
    expect(applyPostgresqlSslMode(settingsWithCertificates, PostgresSslMode.Disable)).toMatchObject(
      {
        sslMode: PostgresSslMode.Disable,
        sslClientCert: '',
        sslClientKey: '',
        sslRootCert: '',
      },
    );
  });

  it('keeps the certificates for another SSL mode', () => {
    expect(
      applyPostgresqlSslMode(settingsWithCertificates, PostgresSslMode.VerifyFull).sslClientCert,
    ).toBe('client-cert');
  });
});

describe('applyPostgresqlHost', () => {
  it('derives the mode and clears certificates while the user has not chosen a mode', () => {
    expect(applyPostgresqlHost(settingsWithCertificates, 'localhost', false)).toMatchObject({
      host: 'localhost',
      sslMode: PostgresSslMode.Disable,
      sslClientCert: '',
    });
  });

  it('leaves the mode alone for a remote hostname', () => {
    expect(applyPostgresqlHost(settingsWithCertificates, 'db.example.com', false).sslMode).toBe(
      PostgresSslMode.Require,
    );
  });

  it('keeps a chosen mode and the certificates for localhost', () => {
    expect(applyPostgresqlHost(settingsWithCertificates, 'localhost', true)).toMatchObject({
      sslMode: PostgresSslMode.Require,
      sslClientCert: 'client-cert',
    });
  });

  it('turns SSL on for an https host when a chosen mode has it off and strips the scheme', () => {
    const settingsWithSslOff = { ...settingsWithCertificates, sslMode: PostgresSslMode.Disable };

    expect(applyPostgresqlHost(settingsWithSslOff, 'https://db.example.com', true)).toMatchObject({
      host: 'db.example.com',
      sslMode: PostgresSslMode.Require,
    });
  });
});
