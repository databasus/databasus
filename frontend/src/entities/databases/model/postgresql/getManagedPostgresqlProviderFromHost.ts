import type { ManagedPostgresqlProvider } from '@/entities/databases/model/postgresql/ManagedPostgresqlProvider';

/* eslint-disable i18next/no-literal-string -- host name parts */
const NEON_DOMAIN_SUFFIX = '.neon.tech';
const SUPABASE_DOMAIN_SUFFIXES = ['.supabase.co', '.supabase.com'];
/* eslint-enable i18next/no-literal-string */

export const getManagedPostgresqlProviderFromHost = (
  host: string,
): ManagedPostgresqlProvider | undefined => {
  const normalizedHost = host.trim().toLowerCase();

  if (normalizedHost.endsWith(NEON_DOMAIN_SUFFIX)) return 'neon';
  if (SUPABASE_DOMAIN_SUFFIXES.some((suffix) => normalizedHost.endsWith(suffix))) return 'supabase';

  return undefined;
};
