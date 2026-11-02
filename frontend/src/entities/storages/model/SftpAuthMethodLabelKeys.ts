import type { TranslationKey } from '../../../shared/i18n';
import { SftpAuthMethod } from './SftpAuthMethod';

export const SFTP_AUTH_METHOD_LABEL_KEYS: Record<SftpAuthMethod, TranslationKey> = {
  [SftpAuthMethod.PASSWORD]: 'storages.sftpAuthMethods.password',
  [SftpAuthMethod.PRIVATE_KEY]: 'storages.sftpAuthMethods.privateKey',
};
