import { DatabaseType } from './DatabaseType';

export const getDatabaseLogoFromType = (type: DatabaseType) => {
  switch (type) {
    case DatabaseType.POSTGRES_LOGICAL:
    case DatabaseType.POSTGRES_PHYSICAL:
      return '/icons/databases/postgresql.svg';
    case DatabaseType.MYSQL:
      return '/icons/databases/mysql.svg';
    case DatabaseType.MARIADB:
      return '/icons/databases/mariadb.svg';
    case DatabaseType.MONGODB:
      return '/icons/databases/mongodb.svg';
    default:
      return '';
  }
};
