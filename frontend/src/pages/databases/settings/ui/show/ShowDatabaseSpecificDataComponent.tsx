import { type Database, DatabaseType } from '@/entities/databases';
import { ShowMariaDbSpecificDataComponent } from '@/pages/databases/settings/ui/show/ShowMariaDbSpecificDataComponent';
import { ShowMongoDbSpecificDataComponent } from '@/pages/databases/settings/ui/show/ShowMongoDbSpecificDataComponent';
import { ShowMySqlSpecificDataComponent } from '@/pages/databases/settings/ui/show/ShowMySqlSpecificDataComponent';
import { ShowPostgreSqlLogicalSpecificDataComponent } from '@/pages/databases/settings/ui/show/ShowPostgreSqlLogicalSpecificDataComponent';
import { ShowPostgreSqlPhysicalSpecificDataComponent } from '@/pages/databases/settings/ui/show/ShowPostgreSqlPhysicalSpecificDataComponent';

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
