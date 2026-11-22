import type { TranslationKey, WEBSITE_PAGES } from '@/shared/i18n';

export type S3CompatibleService = 'neon';

interface S3CompatibleServiceDetails {
  name: string;
  logo: string;
  examples: {
    endpoint: string;
    region: string;
  };
  guidePage: keyof typeof WEBSITE_PAGES;
  guideLinkKey: TranslationKey;
  // The service's console hands out a credentials block, which becomes the main way to fill the
  // S3 fields, with the individual fields as the alternative.
  credentialsPaste: {
    example: string;
    hintKey: TranslationKey;
  };
}

// A managed object storage service is S3 under its own name: choosing it saves an ordinary S3
// storage. The examples follow the block Neon Console shows under Connect -> Storage.
/* eslint-disable i18next/no-literal-string -- service names and endpoint values */
export const S3_COMPATIBLE_SERVICES: Record<S3CompatibleService, S3CompatibleServiceDetails> = {
  neon: {
    name: 'Neon Object Storage',
    logo: '/icons/storages/neon.svg',
    examples: {
      endpoint: 'https://br-<branch>.storage.<cell>.<region>.aws.neon.tech',
      region: 'us-east-2',
    },
    guidePage: 'storagesNeonObjectStorage',
    guideLinkKey: 'storages.edit.s3.neonPaste.guide',
    credentialsPaste: {
      example: [
        'AWS_ENDPOINT_URL_S3="https://br-<branch>.storage.<cell>.<region>.aws.neon.tech"',
        'AWS_ACCESS_KEY_ID="nak_..."',
        'AWS_SECRET_ACCESS_KEY="nsk_..."',
        'AWS_REGION="us-east-2"',
      ].join('\n'),
      hintKey: 'storages.edit.s3.neonPaste.hint',
    },
  },
};
/* eslint-enable i18next/no-literal-string */
