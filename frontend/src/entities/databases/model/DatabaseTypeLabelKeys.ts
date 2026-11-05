import type { TranslationKey } from '../../../shared/i18n';
import { DatabaseType } from './DatabaseType';

export const DATABASE_TYPE_LABEL_KEYS: Record<DatabaseType, TranslationKey> = {
  [DatabaseType.POSTGRES_LOGICAL]: 'databases.types.postgresql',
  [DatabaseType.POSTGRES_PHYSICAL]: 'databases.types.postgresql',
  [DatabaseType.MYSQL]: 'databases.types.mysql',
  [DatabaseType.MARIADB]: 'databases.types.mariadb',
  [DatabaseType.MONGODB]: 'databases.types.mongodb',
};
