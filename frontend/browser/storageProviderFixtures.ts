import type { components } from '../src/shared/api/schema.gen';

export const storageProviders: {
  variant?: string;
  settings: components['schemas']['StorageSettings'];
}[] = [
  { settings: { type: 'LOCAL' } },
  {
    settings: {
      type: 'S3',
      s3Bucket: 'archive-bucket',
      s3Region: 'eu-west-1',
      s3Endpoint: 'https://s3.example.com',
      s3Prefix: 'archives/',
      isVirtualHostedStyle: false,
      isSkipTlsVerify: false,
      s3StorageClass: 'STANDARD_IA',
    },
  },
  {
    settings: {
      type: 'AZURE_BLOB',
      authMethod: 'ACCOUNT_KEY',
      accountName: 'archives',
      containerName: 'backups',
      endpoint: 'https://archives.example.com',
      prefix: 'nightly/',
    },
  },
  {
    settings: {
      type: 'FTP',
      host: 'ftp.example.com',
      port: 21,
      username: 'backup',
      path: '/archives',
      isSsl: true,
      isSkipTlsVerify: false,
    },
  },
  {
    variant: 'AZURE_BLOB connection string',
    settings: {
      type: 'AZURE_BLOB',
      authMethod: 'CONNECTION_STRING',
      containerName: 'backups',
      endpoint: 'https://archives.example.com',
      prefix: 'nightly/',
    },
  },
  {
    settings: {
      type: 'SFTP',
      host: 'sftp.example.com',
      port: 22,
      username: 'backup',
      path: '/archives',
      authMethods: ['PASSWORD', 'PRIVATE_KEY'],
      isSkipHostKeyVerify: false,
    },
  },
  {
    settings: {
      type: 'NAS',
      host: 'nas.example.com',
      port: 445,
      share: 'backups',
      username: 'backup',
      isSsl: true,
      domain: 'example',
      path: '/nightly',
    },
  },
];
