import type { S3CompatibleService } from '@/entities/storages/model/S3CompatibleService';

/* eslint-disable i18next/no-literal-string -- host name parts */
const NEON_DOMAIN = 'neon.tech';
const URL_SCHEME_PATTERN = /^[a-z][a-z0-9+.-]*:\/\//i;
/* eslint-enable i18next/no-literal-string */

const getEndpointHost = (endpoint: string): string | undefined => {
  const trimmedEndpoint = endpoint.trim();
  if (!trimmedEndpoint) return undefined;

  const endpointUrl = URL_SCHEME_PATTERN.test(trimmedEndpoint)
    ? trimmedEndpoint
    : `https://${trimmedEndpoint}`;

  try {
    return new URL(endpointUrl).hostname.toLowerCase();
  } catch {
    return undefined;
  }
};

const isNeonHost = (host: string): boolean =>
  host === NEON_DOMAIN || host.endsWith(`.${NEON_DOMAIN}`);

export const getS3CompatibleServiceFromEndpoint = (
  endpoint: string | undefined,
): S3CompatibleService | undefined => {
  const host = endpoint === undefined ? undefined : getEndpointHost(endpoint);

  if (host && isNeonHost(host)) return 'neon';

  return undefined;
};
