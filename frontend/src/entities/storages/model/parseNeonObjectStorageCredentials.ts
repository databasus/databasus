import { getS3CompatibleServiceFromEndpoint } from '@/entities/storages/model/getS3CompatibleServiceFromEndpoint';
import type { LocalizedText } from '@/shared/i18n';

/* eslint-disable i18next/no-literal-string -- variable names of the Neon Console block */
const ENDPOINT_NAME = 'AWS_ENDPOINT_URL_S3';
const REGION_NAME = 'AWS_REGION';
const ACCESS_KEY_NAME = 'AWS_ACCESS_KEY_ID';
const SECRET_KEY_NAME = 'AWS_SECRET_ACCESS_KEY';
const EXPORT_PREFIX = 'export ';
/* eslint-enable i18next/no-literal-string */

// Whitespace, angle brackets, ellipses, bullets and asterisks are how a key looks before a
// credential is generated or while the console hides it, never how a real value looks.
const PLACEHOLDER_PATTERN = /[\s<>•*…]|\.\.\./;

export type NeonObjectStorageKeyName = typeof ACCESS_KEY_NAME | typeof SECRET_KEY_NAME;

export interface NeonObjectStorageCredentials {
  endpoint: string;
  region?: string;
  accessKey?: string;
  secretKey?: string;
  // Keys the block lacks or holds only a placeholder for; their fields stay as they were.
  missingKeys: NeonObjectStorageKeyName[];
}

const unquote = (rawValue: string): string => {
  const quote = rawValue[0];

  if (quote === '"' || quote === "'") {
    const closingQuoteIndex = rawValue.indexOf(quote, 1);
    return closingQuoteIndex === -1 ? rawValue.slice(1) : rawValue.slice(1, closingQuoteIndex);
  }

  const commentIndex = rawValue.search(/\s#/);
  return (commentIndex === -1 ? rawValue : rawValue.slice(0, commentIndex)).trim();
};

const readDotenvValues = (text: string): Map<string, string> => {
  const values = new Map<string, string>();

  for (const rawLine of text.split(/\r?\n/)) {
    const line = rawLine.trim();
    if (!line || line.startsWith('#')) continue;

    const assignment = line.startsWith(EXPORT_PREFIX) ? line.slice(EXPORT_PREFIX.length) : line;
    const separatorIndex = assignment.indexOf('=');
    if (separatorIndex <= 0) continue;

    const name = assignment.slice(0, separatorIndex).trim();
    values.set(name, unquote(assignment.slice(separatorIndex + 1).trim()));
  }

  return values;
};

const toRealValue = (value: string | undefined): string | undefined =>
  value && !PLACEHOLDER_PATTERN.test(value) ? value : undefined;

// Reads the S3 credentials block Neon Console gives for object storage. No message it returns holds
// a value from the block, because the block carries the secret key.
export const parseNeonObjectStorageCredentials = (
  text: string,
): NeonObjectStorageCredentials | { error: LocalizedText } => {
  const values = readDotenvValues(text);
  const endpoint = toRealValue(values.get(ENDPOINT_NAME));

  if (!endpoint || getS3CompatibleServiceFromEndpoint(endpoint) !== 'neon') {
    return { error: { key: 'storages.edit.s3.neonPaste.notNeonCredentials' } };
  }

  const accessKey = toRealValue(values.get(ACCESS_KEY_NAME));
  const secretKey = toRealValue(values.get(SECRET_KEY_NAME));
  const missingKeys: NeonObjectStorageKeyName[] = [];

  if (!accessKey) missingKeys.push(ACCESS_KEY_NAME);
  if (!secretKey) missingKeys.push(SECRET_KEY_NAME);

  return {
    endpoint,
    region: toRealValue(values.get(REGION_NAME)),
    accessKey,
    secretKey,
    missingKeys,
  };
};
