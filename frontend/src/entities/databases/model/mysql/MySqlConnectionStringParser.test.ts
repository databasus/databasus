import { describe, expect, it } from 'vitest';

import {
  MySqlConnectionStringParser,
  type ParseError,
  type ParseResult,
} from './MySqlConnectionStringParser';

describe('MySqlConnectionStringParser', () => {
  // Helper to assert successful parse
  const expectSuccess = (result: ParseResult | ParseError): ParseResult => {
    expect('error' in result).toBe(false);
    return result as ParseResult;
  };

  // Helper to assert parse error
  const expectError = (result: ParseResult | ParseError): ParseError => {
    expect('error' in result).toBe(true);
    return result as ParseError;
  };

  describe('Standard MySQL URI (mysql://)', () => {
    it('should parse basic mysql:// connection string', () => {
      const result = expectSuccess(
        MySqlConnectionStringParser.parse('mysql://myuser:mypassword@localhost:3306/mydb'),
      );

      expect(result.host).toBe('localhost');
      expect(result.port).toBe(3306);
      expect(result.username).toBe('myuser');
      expect(result.password).toBe('mypassword');
      expect(result.database).toBe('mydb');
      expect(result.isHttps).toBe(false);
    });

    it('should default port to 3306 when not specified', () => {
      const result = expectSuccess(MySqlConnectionStringParser.parse('mysql://user:pass@host/db'));

      expect(result.port).toBe(3306);
    });

    it('should handle URL-encoded passwords', () => {
      const result = expectSuccess(
        MySqlConnectionStringParser.parse('mysql://user:p%40ss%23word@host:3306/db'),
      );

      expect(result.password).toBe('p@ss#word');
    });

    it('should handle URL-encoded usernames', () => {
      const result = expectSuccess(
        MySqlConnectionStringParser.parse('mysql://user%40domain:password@host:3306/db'),
      );

      expect(result.username).toBe('user@domain');
    });
  });

  describe('AWS RDS Connection String', () => {
    it('should parse AWS RDS connection string', () => {
      const result = expectSuccess(
        MySqlConnectionStringParser.parse(
          'mysql://rdsuser:rdspass@mydb.abc123xyz.us-east-1.rds.amazonaws.com:3306/mydb',
        ),
      );

      expect(result.host).toBe('mydb.abc123xyz.us-east-1.rds.amazonaws.com');
      expect(result.port).toBe(3306);
      expect(result.username).toBe('rdsuser');
    });
  });

  describe('PlanetScale Connection String', () => {
    it('should parse PlanetScale connection string with sslaccept', () => {
      const result = expectSuccess(
        MySqlConnectionStringParser.parse(
          'mysql://psuser:pspass@xxx.connect.psdb.cloud/mydb?sslaccept=strict',
        ),
      );

      expect(result.host).toBe('xxx.connect.psdb.cloud');
      expect(result.username).toBe('psuser');
      expect(result.database).toBe('mydb');
      expect(result.isHttps).toBe(true);
    });
  });

  describe('DigitalOcean Connection String', () => {
    it('should parse DigitalOcean connection string with ssl-mode', () => {
      const result = expectSuccess(
        MySqlConnectionStringParser.parse(
          'mysql://doadmin:dopassword@db-mysql-nyc1-12345-do-user-123456-0.b.db.ondigitalocean.com:25060/defaultdb?ssl-mode=REQUIRED',
        ),
      );

      expect(result.host).toBe('db-mysql-nyc1-12345-do-user-123456-0.b.db.ondigitalocean.com');
      expect(result.port).toBe(25060);
      expect(result.username).toBe('doadmin');
      expect(result.database).toBe('defaultdb');
      expect(result.isHttps).toBe(true);
    });
  });

  describe('Azure Database for MySQL Connection String', () => {
    it('should parse Azure connection string with user@server format', () => {
      const result = expectSuccess(
        MySqlConnectionStringParser.parse(
          'mysql://myuser@myserver:mypassword@myserver.mysql.database.azure.com:3306/mydb?ssl-mode=REQUIRED',
        ),
      );

      expect(result.host).toBe('myserver.mysql.database.azure.com');
      expect(result.port).toBe(3306);
      expect(result.username).toBe('myuser');
      expect(result.password).toBe('mypassword');
      expect(result.database).toBe('mydb');
      expect(result.isHttps).toBe(true);
    });
  });

  describe('Railway Connection String', () => {
    it('should parse Railway connection string', () => {
      const result = expectSuccess(
        MySqlConnectionStringParser.parse(
          'mysql://root:railwaypass@containers-us-west-123.railway.app:3306/railway',
        ),
      );

      expect(result.host).toBe('containers-us-west-123.railway.app');
      expect(result.username).toBe('root');
      expect(result.database).toBe('railway');
    });
  });

  describe('JDBC Connection String', () => {
    it('should parse JDBC connection string with user and password params', () => {
      const result = expectSuccess(
        MySqlConnectionStringParser.parse(
          'jdbc:mysql://localhost:3306/mydb?user=admin&password=secret',
        ),
      );

      expect(result.host).toBe('localhost');
      expect(result.port).toBe(3306);
      expect(result.username).toBe('admin');
      expect(result.password).toBe('secret');
      expect(result.database).toBe('mydb');
    });

    it('should parse JDBC connection string without port', () => {
      const result = expectSuccess(
        MySqlConnectionStringParser.parse(
          'jdbc:mysql://db.example.com/mydb?user=admin&password=secret',
        ),
      );

      expect(result.host).toBe('db.example.com');
      expect(result.port).toBe(3306);
    });

    it('should parse JDBC with useSSL parameter', () => {
      const result = expectSuccess(
        MySqlConnectionStringParser.parse(
          'jdbc:mysql://host:3306/db?user=u&password=p&useSSL=true',
        ),
      );

      expect(result.isHttps).toBe(true);
    });

    it('should parse JDBC with sslMode parameter', () => {
      const result = expectSuccess(
        MySqlConnectionStringParser.parse(
          'jdbc:mysql://host:3306/db?user=u&password=p&sslMode=REQUIRED',
        ),
      );

      expect(result.isHttps).toBe(true);
    });

    it('should return error for JDBC without user parameter', () => {
      const result = expectError(
        MySqlConnectionStringParser.parse('jdbc:mysql://host:3306/db?password=secret'),
      );

      expect(result.error.key).toBe('databases.connectionString.errors.jdbc.usernameMissing');
      expect(result.format).toBe('JDBC');
    });

    it('should return error for JDBC without password parameter', () => {
      const result = expectError(
        MySqlConnectionStringParser.parse('jdbc:mysql://host:3306/db?user=admin'),
      );

      expect(result.error.key).toBe('databases.connectionString.errors.jdbc.passwordMissing');
      expect(result.format).toBe('JDBC');
    });
  });

  describe('SSL Mode Handling', () => {
    it('should set isHttps=true for ssl=true', () => {
      const result = expectSuccess(
        MySqlConnectionStringParser.parse('mysql://u:p@host:3306/db?ssl=true'),
      );

      expect(result.isHttps).toBe(true);
    });

    it('should set isHttps=true for sslMode=REQUIRED', () => {
      const result = expectSuccess(
        MySqlConnectionStringParser.parse('mysql://u:p@host:3306/db?sslMode=REQUIRED'),
      );

      expect(result.isHttps).toBe(true);
    });

    it('should set isHttps=true for ssl-mode=REQUIRED', () => {
      const result = expectSuccess(
        MySqlConnectionStringParser.parse('mysql://u:p@host:3306/db?ssl-mode=REQUIRED'),
      );

      expect(result.isHttps).toBe(true);
    });

    it('should set isHttps=true for useSSL=true', () => {
      const result = expectSuccess(
        MySqlConnectionStringParser.parse('mysql://u:p@host:3306/db?useSSL=true'),
      );

      expect(result.isHttps).toBe(true);
    });

    it('should set isHttps=true for sslMode=verify_ca', () => {
      const result = expectSuccess(
        MySqlConnectionStringParser.parse('mysql://u:p@host:3306/db?sslMode=verify_ca'),
      );

      expect(result.isHttps).toBe(true);
    });

    it('should set isHttps=true for sslMode=verify_identity', () => {
      const result = expectSuccess(
        MySqlConnectionStringParser.parse('mysql://u:p@host:3306/db?sslMode=verify_identity'),
      );

      expect(result.isHttps).toBe(true);
    });

    it('should set isHttps=false for ssl=false', () => {
      const result = expectSuccess(
        MySqlConnectionStringParser.parse('mysql://u:p@host:3306/db?ssl=false'),
      );

      expect(result.isHttps).toBe(false);
    });

    it('should set isHttps=false when no ssl specified', () => {
      const result = expectSuccess(MySqlConnectionStringParser.parse('mysql://u:p@host:3306/db'));

      expect(result.isHttps).toBe(false);
    });
  });

  describe('Key-Value Format', () => {
    it('should parse key-value format connection string', () => {
      const result = expectSuccess(
        MySqlConnectionStringParser.parse(
          'host=localhost port=3306 database=mydb user=admin password=secret',
        ),
      );

      expect(result.host).toBe('localhost');
      expect(result.port).toBe(3306);
      expect(result.username).toBe('admin');
      expect(result.password).toBe('secret');
      expect(result.database).toBe('mydb');
    });

    it('should parse key-value format with quoted password containing spaces', () => {
      const result = expectSuccess(
        MySqlConnectionStringParser.parse(
          "host=localhost port=3306 database=mydb user=admin password='my secret pass'",
        ),
      );

      expect(result.password).toBe('my secret pass');
    });

    it('should default port to 3306 when not specified in key-value format', () => {
      const result = expectSuccess(
        MySqlConnectionStringParser.parse(
          'host=localhost database=mydb user=admin password=secret',
        ),
      );

      expect(result.port).toBe(3306);
    });

    it('should handle hostaddr as alternative to host', () => {
      const result = expectSuccess(
        MySqlConnectionStringParser.parse(
          'hostaddr=192.168.1.1 port=3306 database=mydb user=admin password=secret',
        ),
      );

      expect(result.host).toBe('192.168.1.1');
    });

    it('should handle dbname as alternative to database', () => {
      const result = expectSuccess(
        MySqlConnectionStringParser.parse(
          'host=localhost port=3306 dbname=mydb user=admin password=secret',
        ),
      );

      expect(result.database).toBe('mydb');
    });

    it('should handle username as alternative to user', () => {
      const result = expectSuccess(
        MySqlConnectionStringParser.parse(
          'host=localhost port=3306 database=mydb username=admin password=secret',
        ),
      );

      expect(result.username).toBe('admin');
    });

    it('should parse ssl in key-value format', () => {
      const result = expectSuccess(
        MySqlConnectionStringParser.parse(
          'host=localhost database=mydb user=admin password=secret ssl=true',
        ),
      );

      expect(result.isHttps).toBe(true);
    });

    it('should parse sslMode in key-value format', () => {
      const result = expectSuccess(
        MySqlConnectionStringParser.parse(
          'host=localhost database=mydb user=admin password=secret sslMode=REQUIRED',
        ),
      );

      expect(result.isHttps).toBe(true);
    });

    it('should return error for key-value format missing host', () => {
      const result = expectError(
        MySqlConnectionStringParser.parse('port=3306 database=mydb user=admin password=secret'),
      );

      expect(result.error.key).toBe('databases.connectionString.errors.keyValue.hostMissing');
      expect(result.format).toBe('key-value');
    });

    it('should return error for key-value format missing user', () => {
      const result = expectError(
        MySqlConnectionStringParser.parse('host=localhost database=mydb password=secret'),
      );

      expect(result.error.key).toBe('databases.connectionString.errors.keyValue.usernameMissing');
      expect(result.format).toBe('key-value');
    });

    it('should return error for key-value format missing password', () => {
      const result = expectError(
        MySqlConnectionStringParser.parse('host=localhost database=mydb user=admin'),
      );

      expect(result.error.key).toBe('databases.connectionString.errors.keyValue.passwordMissing');
      expect(result.format).toBe('key-value');
    });

    it('should return error for key-value format missing database', () => {
      const result = expectError(
        MySqlConnectionStringParser.parse('host=localhost user=admin password=secret'),
      );

      expect(result.error.key).toBe('databases.connectionString.errors.keyValue.databaseMissing');
      expect(result.format).toBe('key-value');
    });
  });

  describe('Error Cases', () => {
    it('should return error for empty string', () => {
      const result = expectError(MySqlConnectionStringParser.parse(''));

      expect(result.error.key).toBe('databases.connectionString.errors.empty');
    });

    it('should return error for whitespace-only string', () => {
      const result = expectError(MySqlConnectionStringParser.parse('   '));

      expect(result.error.key).toBe('databases.connectionString.errors.empty');
    });

    it('should return error for unrecognized format', () => {
      const result = expectError(MySqlConnectionStringParser.parse('some random text'));

      expect(result.error.key).toBe('databases.connectionString.errors.unrecognizedFormat');
    });

    it('should return error for missing username in URI', () => {
      const result = expectError(
        MySqlConnectionStringParser.parse('mysql://:password@host:3306/db'),
      );

      expect(result.error.key).toBe('databases.connectionString.errors.usernameMissing');
    });

    it('should return error for missing password in URI', () => {
      const result = expectError(MySqlConnectionStringParser.parse('mysql://user@host:3306/db'));

      expect(result.error.key).toBe('databases.connectionString.errors.passwordMissing');
    });

    it('should return error for missing database in URI', () => {
      const result = expectError(MySqlConnectionStringParser.parse('mysql://user:pass@host:3306/'));

      expect(result.error.key).toBe('databases.connectionString.errors.databaseMissing');
    });

    it('should return error for invalid JDBC format', () => {
      const result = expectError(MySqlConnectionStringParser.parse('jdbc:mysql://invalid'));

      expect(result.error.key).toBe('databases.connectionString.errors.jdbc.invalidFormat');
      expect(result.format).toBe('JDBC');
    });

    it('should return error for postgresql:// format (wrong database type)', () => {
      const result = expectError(
        MySqlConnectionStringParser.parse('postgresql://user:pass@host:5432/db'),
      );

      expect(result.error.key).toBe('databases.connectionString.errors.unrecognizedFormat');
    });
  });

  describe('Edge Cases', () => {
    it('should handle special characters in password', () => {
      const result = expectSuccess(
        MySqlConnectionStringParser.parse('mysql://user:p%40ss%3Aw%2Ford@host:3306/db'),
      );

      expect(result.password).toBe('p@ss:w/ord');
    });

    it('should handle numeric database names', () => {
      const result = expectSuccess(
        MySqlConnectionStringParser.parse('mysql://user:pass@host:3306/12345'),
      );

      expect(result.database).toBe('12345');
    });

    it('should handle hyphenated host names', () => {
      const result = expectSuccess(
        MySqlConnectionStringParser.parse('mysql://user:pass@my-database-host.example.com:3306/db'),
      );

      expect(result.host).toBe('my-database-host.example.com');
    });

    it('should handle connection string with extra query parameters', () => {
      const result = expectSuccess(
        MySqlConnectionStringParser.parse(
          'mysql://user:pass@host:3306/db?ssl=true&connectTimeout=10&charset=utf8mb4',
        ),
      );

      expect(result.isHttps).toBe(true);
      expect(result.database).toBe('db');
    });

    it('should trim whitespace from connection string', () => {
      const result = expectSuccess(
        MySqlConnectionStringParser.parse('  mysql://user:pass@host:3306/db  '),
      );

      expect(result.host).toBe('host');
    });
  });
});
