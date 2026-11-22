import { describe, expect, it } from 'vitest';

import { getManagedPostgresqlProviderFromHost } from '@/entities/databases/model/postgresql/getManagedPostgresqlProviderFromHost';

describe('getManagedPostgresqlProviderFromHost', () => {
  it.each([
    { host: 'ep-young-dew-123456.us-east-2.aws.neon.tech', provider: 'neon' },
    { host: 'EP-Young-Dew-123456-pooler.c-9.us-east-2.aws.neon.tech', provider: 'neon' },
    { host: 'db.abcdefghijklmnop.supabase.co', provider: 'supabase' },
    { host: 'aws-0-eu-central-1.pooler.supabase.com', provider: 'supabase' },
    { host: 'db.example.com', provider: undefined },
    { host: 'neon.tech.example.com', provider: undefined },
  ])('returns $provider for $host', ({ host, provider }) => {
    expect(getManagedPostgresqlProviderFromHost(host)).toBe(provider);
  });
});
