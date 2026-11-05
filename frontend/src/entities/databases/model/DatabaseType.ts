import type { DatabaseSettings } from './DatabaseSettings';

export type DatabaseType = DatabaseSettings['type'];

// The values the screens compare against. satisfies keeps them within the contract's enum.
export const DatabaseType = {
  POSTGRES_LOGICAL: 'POSTGRES_LOGICAL',
  POSTGRES_PHYSICAL: 'POSTGRES_PHYSICAL',
  MYSQL: 'MYSQL',
  MARIADB: 'MARIADB',
  MONGODB: 'MONGODB',
} as const satisfies Record<string, DatabaseType>;
