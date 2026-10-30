import { type Database, DatabaseType } from '../../../../entities/databases';
import { ShowMariaDbSpecificDataComponent } from './ShowMariaDbSpecificDataComponent';
import { ShowMongoDbSpecificDataComponent } from './ShowMongoDbSpecificDataComponent';
import { ShowMySqlSpecificDataComponent } from './ShowMySqlSpecificDataComponent';
import { ShowPostgreSqlLogicalSpecificDataComponent } from './ShowPostgreSqlLogicalSpecificDataComponent';
import { ShowPostgreSqlPhysicalSpecificDataComponent } from './ShowPostgreSqlPhysicalSpecificDataComponent';

interface Props {
  database: Database;
}

export const ShowDatabaseSpecificDataComponent = ({ database }: Props) => {
  const { settings } = database;

  switch (settings.type) {
    case DatabaseType.POSTGRES_LOGICAL:
      return <ShowPostgreSqlLogicalSpecificDataComponent settings={settings} />;
    case DatabaseType.POSTGRES_PHYSICAL:
      return <ShowPostgreSqlPhysicalSpecificDataComponent settings={settings} />;
    case DatabaseType.MYSQL:
      return <ShowMySqlSpecificDataComponent settings={settings} />;
    case DatabaseType.MARIADB:
      return <ShowMariaDbSpecificDataComponent settings={settings} />;
    case DatabaseType.MONGODB:
      return <ShowMongoDbSpecificDataComponent settings={settings} />;
  }
};
