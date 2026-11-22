import type { TranslationKey } from '@/shared/i18n';

export type ManagedPostgresqlProvider = 'neon' | 'supabase';

interface ManagedPostgresqlProviderDetails {
  name: string;
  logo: string;
  connectionExamples: {
    host: string;
    username: string;
    database: string;
  };
  // A provider whose console hands out a whole connection string gets that string as the main way to
  // fill the connection, with the individual fields as the alternative.
  connectionStringPaste?: {
    example: string;
    hintKey: TranslationKey;
  };
}

// A managed provider is PostgreSQL under its own name. The examples follow the connection string the
// provider's console shows: Neon's direct host and Supabase's session pooler, which the website FAQ
// recommends for backups.
/* eslint-disable i18next/no-literal-string -- provider names and connection values */
export const MANAGED_POSTGRESQL_PROVIDERS: Record<
  ManagedPostgresqlProvider,
  ManagedPostgresqlProviderDetails
> = {
  neon: {
    name: 'Neon',
    logo: '/icons/databases/neon.svg',
    connectionExamples: {
      host: 'ep-young-dew-123456.us-east-2.aws.neon.tech',
      username: 'neondb_owner',
      database: 'neondb',
    },
    connectionStringPaste: {
      example:
        'postgresql://neondb_owner:****@ep-young-dew-123456-pooler.us-east-2.aws.neon.tech/neondb?sslmode=require&channel_binding=require',
      hintKey: 'databases.edit.connectionStringPaste.neonHint',
    },
  },
  supabase: {
    name: 'Supabase',
    logo: '/icons/databases/supabase.svg',
    connectionExamples: {
      host: 'aws-0-eu-central-1.pooler.supabase.com',
      username: 'postgres.abcdefghijklmnop',
      database: 'postgres',
    },
  },
};
/* eslint-enable i18next/no-literal-string */
