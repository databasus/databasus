import type { TranslationKey } from '../../../../shared/i18n';
import { PostgresSslMode } from './PostgresSslMode';

export const POSTGRES_SSL_MODE_LABEL_KEYS: Record<PostgresSslMode, TranslationKey> = {
  [PostgresSslMode.Disable]: 'databases.sslModes.disable',
  [PostgresSslMode.Require]: 'databases.sslModes.require',
  [PostgresSslMode.VerifyCa]: 'databases.sslModes.verifyCa',
  [PostgresSslMode.VerifyFull]: 'databases.sslModes.verifyFull',
};
