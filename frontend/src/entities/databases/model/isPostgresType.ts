import { DatabaseType } from './DatabaseType';

export const isPostgresType = (type: DatabaseType): boolean =>
  type === DatabaseType.POSTGRES_LOGICAL || type === DatabaseType.POSTGRES_PHYSICAL;
