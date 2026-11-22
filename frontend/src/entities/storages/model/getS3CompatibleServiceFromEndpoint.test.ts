import { describe, expect, it } from 'vitest';

import { getS3CompatibleServiceFromEndpoint } from '@/entities/storages/model/getS3CompatibleServiceFromEndpoint';

describe('getS3CompatibleServiceFromEndpoint', () => {
  it.each([
    {
      endpoint: 'https://br-dark-river-123456.storage.c-9.us-east-2.aws.neon.tech',
      service: 'neon',
    },
    {
      endpoint: 'HTTPS://BR-Dark-River-123456.Storage.C-9.us-east-2.aws.neon.tech/',
      service: 'neon',
    },
    { endpoint: 'br-dark-river-123456.storage.c-9.us-east-2.aws.neon.tech', service: 'neon' },
    { endpoint: 'https://neon.tech', service: 'neon' },
    { endpoint: 'https://0123456789abcdef.r2.cloudflarestorage.com', service: undefined },
    { endpoint: 'https://s3.us-east-2.amazonaws.com', service: undefined },
    { endpoint: 'https://neon.tech.example.com', service: undefined },
    { endpoint: 'https://notneon.tech', service: undefined },
    { endpoint: '', service: undefined },
    { endpoint: undefined, service: undefined },
  ])('returns $service for $endpoint', ({ endpoint, service }) => {
    expect(getS3CompatibleServiceFromEndpoint(endpoint)).toBe(service);
  });
});
