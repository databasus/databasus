import type { TranslationKey } from '../../../shared/i18n';
import { AzureBlobAuthMethod } from './AzureBlobAuthMethod';

export const AZURE_BLOB_AUTH_METHOD_LABEL_KEYS: Record<AzureBlobAuthMethod, TranslationKey> = {
  [AzureBlobAuthMethod.ACCOUNT_KEY]: 'storages.azureBlobAuthMethods.accountKey',
  [AzureBlobAuthMethod.CONNECTION_STRING]: 'storages.azureBlobAuthMethods.connectionString',
};
