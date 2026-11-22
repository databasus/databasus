import { describe, expect, it } from 'vitest';

import { parseNeonObjectStorageCredentials } from '@/entities/storages/model/parseNeonObjectStorageCredentials';

const neonEndpoint = 'https://br-dark-river-123456.storage.c-9.us-east-2.aws.neon.tech';
const fakeAccessKey = `nak_live_${'0a1b2c3d'.repeat(4)}`;
const fakeSecretKey = `nsk_live_${'9f8e7d6c'.repeat(8)}`;

const completeCredentials = {
  endpoint: neonEndpoint,
  region: 'us-east-2',
  accessKey: fakeAccessKey,
  secretKey: fakeSecretKey,
  missingKeys: [],
};

const notNeonCredentials = { error: { key: 'storages.edit.s3.neonPaste.notNeonCredentials' } };

describe('parseNeonObjectStorageCredentials', () => {
  it('reads every value when the block comes from Neon Console', () => {
    const consoleBlock = [
      '# S3-compatible storage. Generate a credential to fill in the key ID + secret.',
      `AWS_ENDPOINT_URL_S3="${neonEndpoint}"`,
      `AWS_ACCESS_KEY_ID="${fakeAccessKey}"`,
      `AWS_SECRET_ACCESS_KEY="${fakeSecretKey}"`,
      'AWS_REGION="us-east-2"',
    ].join('\n');

    expect(parseNeonObjectStorageCredentials(consoleBlock)).toEqual(completeCredentials);
  });

  it('reads every value when lines are reordered, exported, unquoted or single-quoted', () => {
    const editedBlock = [
      "export AWS_REGION='us-east-2'",
      '',
      `AWS_SECRET_ACCESS_KEY=${fakeSecretKey} # generated today`,
      'UNRELATED_SETTING="kept out"',
      `export AWS_ACCESS_KEY_ID=${fakeAccessKey}`,
      `AWS_ENDPOINT_URL_S3=${neonEndpoint}`,
    ].join('\r\n');

    expect(parseNeonObjectStorageCredentials(editedBlock)).toEqual(completeCredentials);
  });

  it.each([
    { scenario: 'are bullets of a hidden secret', accessKey: '••••••••', secretKey: '••••••••' },
    { scenario: 'are asterisks', accessKey: '********', secretKey: '****' },
    { scenario: 'are angle-bracket placeholders', accessKey: '<key id>', secretKey: '<secret>' },
    { scenario: 'end in an ellipsis', accessKey: 'nak_...', secretKey: 'nsk_…' },
    { scenario: 'are empty', accessKey: '', secretKey: '' },
    { scenario: 'hold whitespace', accessKey: 'generate a key', secretKey: ' ' },
  ])('lists both keys as missing when they $scenario', ({ accessKey, secretKey }) => {
    const placeholderBlock = [
      `AWS_ENDPOINT_URL_S3="${neonEndpoint}"`,
      `AWS_ACCESS_KEY_ID="${accessKey}"`,
      `AWS_SECRET_ACCESS_KEY="${secretKey}"`,
      'AWS_REGION="us-east-2"',
    ].join('\n');

    expect(parseNeonObjectStorageCredentials(placeholderBlock)).toEqual({
      endpoint: neonEndpoint,
      region: 'us-east-2',
      accessKey: undefined,
      secretKey: undefined,
      missingKeys: ['AWS_ACCESS_KEY_ID', 'AWS_SECRET_ACCESS_KEY'],
    });
  });

  it('lists only the absent key when one key line is missing', () => {
    const partialBlock = [
      `AWS_ENDPOINT_URL_S3="${neonEndpoint}"`,
      `AWS_ACCESS_KEY_ID="${fakeAccessKey}"`,
    ].join('\n');

    expect(parseNeonObjectStorageCredentials(partialBlock)).toEqual({
      endpoint: neonEndpoint,
      region: undefined,
      accessKey: fakeAccessKey,
      secretKey: undefined,
      missingKeys: ['AWS_SECRET_ACCESS_KEY'],
    });
  });

  it.each([
    { scenario: 'unrelated text', text: 'not a credentials block' },
    {
      scenario: 'an endpoint outside Neon',
      text: `AWS_ENDPOINT_URL_S3="https://0123456789abcdef.r2.cloudflarestorage.com"\nAWS_ACCESS_KEY_ID="${fakeAccessKey}"`,
    },
    {
      scenario: 'a placeholder endpoint',
      text: 'AWS_ENDPOINT_URL_S3="https://br-<branch>.storage.<cell>.<region>.aws.neon.tech"',
    },
    {
      scenario: 'keys without an endpoint',
      text: `AWS_ACCESS_KEY_ID="${fakeAccessKey}"\nAWS_SECRET_ACCESS_KEY="${fakeSecretKey}"`,
    },
  ])('returns an error without any value for $scenario', ({ text }) => {
    const parseResult = parseNeonObjectStorageCredentials(text);

    expect(parseResult).toEqual(notNeonCredentials);
    expect(JSON.stringify(parseResult)).not.toContain(fakeAccessKey);
    expect(JSON.stringify(parseResult)).not.toContain(fakeSecretKey);
  });
});
