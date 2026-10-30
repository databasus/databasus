import { describe, expect, it } from 'vitest';

import {
  ConnectionStringParser,
  type ParseError,
  type ParseResult,
} from './ConnectionStringParser';
import { PostgresSslMode } from './PostgresSslMode';

describe('ConnectionStringParser', () => {
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

  describe('Standard PostgreSQL URI (postgresql://)', () => {
    it('should parse basic postgresql:// connection string', () => {
      const result = expectSuccess(
        ConnectionStringParser.parse('postgresql://myuser:mypassword@localhost:5432/mydb'),
      );

      expect(result.host).toBe('localhost');
      expect(result.port).toBe(5432);
      expect(result.username).toBe('myuser');
      expect(result.password).toBe('mypassword');
      expect(result.database).toBe('mydb');
      expect(result.sslMode).toBe(PostgresSslMode.Disable);
    });

    it('should default port to 5432 when not specified', () => {
      const result = expectSuccess(ConnectionStringParser.parse('postgresql://user:pass@host/db'));

      expect(result.port).toBe(5432);
    });

    it('should handle URL-encoded passwords', () => {
      const result = expectSuccess(
        ConnectionStringParser.parse('postgresql://user:p%40ss%23word@host:5432/db'),
      );

      expect(result.password).toBe('p@ss#word');
    });

    it('should handle URL-encoded usernames', () => {
      const result = expectSuccess(
        ConnectionStringParser.parse('postgresql://user%40domain:password@host:5432/db'),
      );

      expect(result.username).toBe('user@domain');
    });
  });

  describe('Postgres URI (postgres://)', () => {
    it('should parse basic postgres:// connection string', () => {
      const result = expectSuccess(
        ConnectionStringParser.parse('postgres://admin:secret@db.example.com:5432/production'),
      );

      expect(result.host).toBe('db.example.com');
      expect(result.port).toBe(5432);
      expect(result.username).toBe('admin');
      expect(result.password).toBe('secret');
      expect(result.database).toBe('production');
    });
  });

  describe('Supabase Direct Connection', () => {
    it('should parse Supabase direct connection string', () => {
      const result = expectSuccess(
        ConnectionStringParser.parse(
          'postgresql://postgres:mySecretPassword@db.abcdefghijklmnop.supabase.co:5432/postgres',
        ),
      );

      expect(result.host).toBe('db.abcdefghijklmnop.supabase.co');
      expect(result.port).toBe(5432);
      expect(result.username).toBe('postgres');
      expect(result.password).toBe('mySecretPassword');
      expect(result.database).toBe('postgres');
    });
  });

  describe('Supabase Pooler Connection', () => {
    it('should parse Supabase pooler session mode connection string (port 5432)', () => {
      const result = expectSuccess(
        ConnectionStringParser.parse(
          'postgres://postgres.abcdefghijklmnop:myPassword@aws-0-us-east-1.pooler.supabase.com:5432/postgres',
        ),
      );

      expect(result.host).toBe('aws-0-us-east-1.pooler.supabase.com');
      expect(result.port).toBe(5432);
      expect(result.username).toBe('postgres.abcdefghijklmnop');
      expect(result.password).toBe('myPassword');
      expect(result.database).toBe('postgres');
    });

    it('should parse Supabase pooler transaction mode connection string (port 6543)', () => {
      const result = expectSuccess(
        ConnectionStringParser.parse(
          'postgres://postgres.projectref:myPassword@aws-0-eu-west-1.pooler.supabase.com:6543/postgres',
        ),
      );

      expect(result.host).toBe('aws-0-eu-west-1.pooler.supabase.com');
      expect(result.port).toBe(6543);
      expect(result.username).toBe('postgres.projectref');
    });
  });

  describe('JDBC Connection String', () => {
    it('should parse JDBC connection string with user and password params', () => {
      const result = expectSuccess(
        ConnectionStringParser.parse(
          'jdbc:postgresql://localhost:5432/mydb?user=admin&password=secret',
        ),
      );

      expect(result.host).toBe('localhost');
      expect(result.port).toBe(5432);
      expect(result.username).toBe('admin');
      expect(result.password).toBe('secret');
      expect(result.database).toBe('mydb');
    });

    it('should parse JDBC connection string without port', () => {
      const result = expectSuccess(
        ConnectionStringParser.parse(
          'jdbc:postgresql://db.example.com/mydb?user=admin&password=secret',
        ),
      );

      expect(result.host).toBe('db.example.com');
      expect(result.port).toBe(5432);
    });

    it('should parse JDBC with sslmode parameter', () => {
      const result = expectSuccess(
        ConnectionStringParser.parse(
          'jdbc:postgresql://host:5432/db?user=u&password=p&sslmode=require',
        ),
      );

      expect(result.sslMode).toBe(PostgresSslMode.Require);
    });

    it('should return error for JDBC without user parameter', () => {
      const result = expectError(
        ConnectionStringParser.parse('jdbc:postgresql://host:5432/db?password=secret'),
      );

      expect(result.error.key).toBe('databases.connectionString.errors.jdbc.usernameMissing');
      expect(result.format).toBe('JDBC');
    });

    it('should return error for JDBC without password parameter', () => {
      const result = expectError(
        ConnectionStringParser.parse('jdbc:postgresql://host:5432/db?user=admin'),
      );

      expect(result.error.key).toBe('databases.connectionString.errors.jdbc.passwordMissing');
      expect(result.format).toBe('JDBC');
    });
  });

  describe('Neon Connection String', () => {
    it('should parse Neon connection string', () => {
      const result = expectSuccess(
        ConnectionStringParser.parse(
          'postgresql://neonuser:password123@ep-cool-name-123456.us-east-2.aws.neon.tech/neondb',
        ),
      );

      expect(result.host).toBe('ep-cool-name-123456.us-east-2.aws.neon.tech');
      expect(result.username).toBe('neonuser');
      expect(result.database).toBe('neondb');
    });
  });

  describe('Railway Connection String', () => {
    it('should parse Railway connection string', () => {
      const result = expectSuccess(
        ConnectionStringParser.parse(
          'postgresql://postgres:railwaypass@containers-us-west-123.railway.app:5432/railway',
        ),
      );

      expect(result.host).toBe('containers-us-west-123.railway.app');
      expect(result.username).toBe('postgres');
      expect(result.database).toBe('railway');
    });
  });

  describe('Render Connection String', () => {
    it('should parse Render connection string', () => {
      const result = expectSuccess(
        ConnectionStringParser.parse(
          'postgresql://renderuser:renderpass@dpg-abc123.oregon-postgres.render.com/mydb',
        ),
      );

      expect(result.host).toBe('dpg-abc123.oregon-postgres.render.com');
      expect(result.username).toBe('renderuser');
      expect(result.database).toBe('mydb');
    });
  });

  describe('DigitalOcean Connection String', () => {
    it('should parse DigitalOcean connection string with sslmode', () => {
      const result = expectSuccess(
        ConnectionStringParser.parse(
          'postgresql://doadmin:dopassword@db-postgresql-nyc1-12345-do-user-123456-0.b.db.ondigitalocean.com:25060/defaultdb?sslmode=require',
        ),
      );

      expect(result.host).toBe('db-postgresql-nyc1-12345-do-user-123456-0.b.db.ondigitalocean.com');
      expect(result.port).toBe(25060);
      expect(result.username).toBe('doadmin');
      expect(result.database).toBe('defaultdb');
      expect(result.sslMode).toBe(PostgresSslMode.Require);
    });
  });

  describe('AWS RDS Connection String', () => {
    it('should parse AWS RDS connection string', () => {
      const result = expectSuccess(
        ConnectionStringParser.parse(
          'postgresql://rdsuser:rdspass@mydb.abc123xyz.us-east-1.rds.amazonaws.com:5432/mydb',
        ),
      );

      expect(result.host).toBe('mydb.abc123xyz.us-east-1.rds.amazonaws.com');
      expect(result.username).toBe('rdsuser');
    });
  });

  describe('Azure Database for PostgreSQL Connection String', () => {
    it('should parse Azure connection string with user@server format', () => {
      const result = expectSuccess(
        ConnectionStringParser.parse(
          'postgresql://myuser@myserver:mypassword@myserver.postgres.database.azure.com:5432/mydb?sslmode=require',
        ),
      );

      expect(result.host).toBe('myserver.postgres.database.azure.com');
      expect(result.port).toBe(5432);
      expect(result.username).toBe('myuser');
      expect(result.password).toBe('mypassword');
      expect(result.database).toBe('mydb');
      expect(result.sslMode).toBe(PostgresSslMode.Require);
    });
  });

  describe('Heroku Connection String', () => {
    it('should parse Heroku connection string', () => {
      const result = expectSuccess(
        ConnectionStringParser.parse(
          'postgres://herokuuser:herokupass@ec2-12-34-56-789.compute-1.amazonaws.com:5432/herokudb',
        ),
      );

      expect(result.host).toBe('ec2-12-34-56-789.compute-1.amazonaws.com');
      expect(result.username).toBe('herokuuser');
      expect(result.database).toBe('herokudb');
    });
  });

  describe('CockroachDB Connection String', () => {
    it('should parse CockroachDB connection string with sslmode=verify-full', () => {
      const result = expectSuccess(
        ConnectionStringParser.parse(
          'postgresql://crdbuser:crdbpass@free-tier.gcp-us-central1.cockroachlabs.cloud:26257/defaultdb?sslmode=verify-full',
        ),
      );

      expect(result.host).toBe('free-tier.gcp-us-central1.cockroachlabs.cloud');
      expect(result.port).toBe(26257);
      expect(result.sslMode).toBe(PostgresSslMode.VerifyFull);
    });
  });

  describe('SSL Mode Handling', () => {
    it('should set sslMode=require for sslmode=require', () => {
      const result = expectSuccess(
        ConnectionStringParser.parse('postgresql://u:p@host:5432/db?sslmode=require'),
      );

      expect(result.sslMode).toBe(PostgresSslMode.Require);
    });

    it('should set sslMode=verify-ca for sslmode=verify-ca', () => {
      const result = expectSuccess(
        ConnectionStringParser.parse('postgresql://u:p@host:5432/db?sslmode=verify-ca'),
      );

      expect(result.sslMode).toBe(PostgresSslMode.VerifyCa);
    });

    it('should set sslMode=verify-full for sslmode=verify-full', () => {
      const result = expectSuccess(
        ConnectionStringParser.parse('postgresql://u:p@host:5432/db?sslmode=verify-full'),
      );

      expect(result.sslMode).toBe(PostgresSslMode.VerifyFull);
    });

    it('should set sslMode=disable for sslmode=disable', () => {
      const result = expectSuccess(
        ConnectionStringParser.parse('postgresql://u:p@host:5432/db?sslmode=disable'),
      );

      expect(result.sslMode).toBe(PostgresSslMode.Disable);
    });

    it('should derive sslMode=require from a remote host when no sslmode specified', () => {
      const result = expectSuccess(ConnectionStringParser.parse('postgresql://u:p@host:5432/db'));

      expect(result.sslMode).toBe(PostgresSslMode.Require);
    });

    it('should derive sslMode=disable from localhost when no sslmode specified', () => {
      const result = expectSuccess(
        ConnectionStringParser.parse('postgresql://u:p@localhost:5432/db'),
      );

      expect(result.sslMode).toBe(PostgresSslMode.Disable);
    });

    it('should derive sslMode=disable from an IPv4 host when no sslmode specified', () => {
      const result = expectSuccess(
        ConnectionStringParser.parse('postgresql://u:p@192.168.1.10:5432/db'),
      );

      expect(result.sslMode).toBe(PostgresSslMode.Disable);
    });

    it('should treat sslmode=prefer as disable', () => {
      const result = expectSuccess(
        ConnectionStringParser.parse('postgresql://u:p@host:5432/db?sslmode=prefer'),
      );

      expect(result.sslMode).toBe(PostgresSslMode.Disable);
    });
  });

  describe('libpq Key-Value Format', () => {
    it('should parse libpq format connection string', () => {
      const result = expectSuccess(
        ConnectionStringParser.parse(
          'host=localhost port=5432 dbname=mydb user=admin password=secret',
        ),
      );

      expect(result.host).toBe('localhost');
      expect(result.port).toBe(5432);
      expect(result.username).toBe('admin');
      expect(result.password).toBe('secret');
      expect(result.database).toBe('mydb');
    });

    it('should parse libpq format with quoted password containing spaces', () => {
      const result = expectSuccess(
        ConnectionStringParser.parse(
          "host=localhost port=5432 dbname=mydb user=admin password='my secret pass'",
        ),
      );

      expect(result.password).toBe('my secret pass');
    });

    it('should default port to 5432 when not specified in libpq format', () => {
      const result = expectSuccess(
        ConnectionStringParser.parse('host=localhost dbname=mydb user=admin password=secret'),
      );

      expect(result.port).toBe(5432);
    });

    it('should handle hostaddr as alternative to host', () => {
      const result = expectSuccess(
        ConnectionStringParser.parse(
          'hostaddr=192.168.1.1 port=5432 dbname=mydb user=admin password=secret',
        ),
      );

      expect(result.host).toBe('192.168.1.1');
    });

    it('should handle database as alternative to dbname', () => {
      const result = expectSuccess(
        ConnectionStringParser.parse(
          'host=localhost port=5432 database=mydb user=admin password=secret',
        ),
      );

      expect(result.database).toBe('mydb');
    });

    it('should handle username as alternative to user', () => {
      const result = expectSuccess(
        ConnectionStringParser.parse(
          'host=localhost port=5432 dbname=mydb username=admin password=secret',
        ),
      );

      expect(result.username).toBe('admin');
    });

    it('should parse sslmode in libpq format', () => {
      const result = expectSuccess(
        ConnectionStringParser.parse(
          'host=localhost dbname=mydb user=admin password=secret sslmode=require',
        ),
      );

      expect(result.sslMode).toBe(PostgresSslMode.Require);
    });

    it('should return error for libpq format missing host', () => {
      const result = expectError(
        ConnectionStringParser.parse('port=5432 dbname=mydb user=admin password=secret'),
      );

      expect(result.error.key).toBe('databases.connectionString.errors.keyValue.hostMissing');
      expect(result.format).toBe('libpq');
    });

    it('should return error for libpq format missing user', () => {
      const result = expectError(
        ConnectionStringParser.parse('host=localhost dbname=mydb password=secret'),
      );

      expect(result.error.key).toBe('databases.connectionString.errors.keyValue.usernameMissing');
      expect(result.format).toBe('libpq');
    });

    it('should return error for libpq format missing password', () => {
      const result = expectError(
        ConnectionStringParser.parse('host=localhost dbname=mydb user=admin'),
      );

      expect(result.error.key).toBe('databases.connectionString.errors.keyValue.passwordMissing');
      expect(result.format).toBe('libpq');
    });

    it('should return error for libpq format missing dbname', () => {
      const result = expectError(
        ConnectionStringParser.parse('host=localhost user=admin password=secret'),
      );

      expect(result.error.key).toBe('databases.connectionString.errors.libpq.databaseMissing');
      expect(result.format).toBe('libpq');
    });
  });

  describe('Error Cases', () => {
    it('should return error for empty string', () => {
      const result = expectError(ConnectionStringParser.parse(''));

      expect(result.error.key).toBe('databases.connectionString.errors.empty');
    });

    it('should return error for whitespace-only string', () => {
      const result = expectError(ConnectionStringParser.parse('   '));

      expect(result.error.key).toBe('databases.connectionString.errors.empty');
    });

    it('should return error for unrecognized format', () => {
      const result = expectError(ConnectionStringParser.parse('some random text'));

      expect(result.error.key).toBe('databases.connectionString.errors.unrecognizedFormat');
    });

    it('should return error for missing username in URI', () => {
      const result = expectError(
        ConnectionStringParser.parse('postgresql://:password@host:5432/db'),
      );

      expect(result.error.key).toBe('databases.connectionString.errors.usernameMissing');
    });

    it('should return error for missing password in URI', () => {
      const result = expectError(ConnectionStringParser.parse('postgresql://user@host:5432/db'));

      expect(result.error.key).toBe('databases.connectionString.errors.passwordMissing');
    });

    it('should return error for missing database in URI', () => {
      const result = expectError(ConnectionStringParser.parse('postgresql://user:pass@host:5432/'));

      expect(result.error.key).toBe('databases.connectionString.errors.databaseMissing');
    });

    it('should return the parse failure without the thrown error text', () => {
      const result = expectError(
        ConnectionStringParser.parse('postgresql://user:pass@host:notaport/db'),
      );

      expect(result.error).toEqual({ key: 'databases.connectionString.errors.parseFailed' });
      expect(result.format).toBe('URI');
    });

    it('should return error for invalid JDBC format', () => {
      const result = expectError(ConnectionStringParser.parse('jdbc:postgresql://invalid'));

      expect(result.error.key).toBe('databases.connectionString.errors.jdbc.invalidFormat');
      expect(result.format).toBe('JDBC');
    });
  });

  describe('Edge Cases', () => {
    it('should handle special characters in password', () => {
      const result = expectSuccess(
        ConnectionStringParser.parse('postgresql://user:p%40ss%3Aw%2Ford@host:5432/db'),
      );

      expect(result.password).toBe('p@ss:w/ord');
    });

    it('should handle numeric database names', () => {
      const result = expectSuccess(
        ConnectionStringParser.parse('postgresql://user:pass@host:5432/12345'),
      );

      expect(result.database).toBe('12345');
    });

    it('should handle hyphenated host names', () => {
      const result = expectSuccess(
        ConnectionStringParser.parse('postgresql://user:pass@my-database-host.example.com:5432/db'),
      );

      expect(result.host).toBe('my-database-host.example.com');
    });

    it('should handle connection string with extra query parameters', () => {
      const result = expectSuccess(
        ConnectionStringParser.parse(
          'postgresql://user:pass@host:5432/db?sslmode=require&connect_timeout=10&application_name=myapp',
        ),
      );

      expect(result.sslMode).toBe(PostgresSslMode.Require);
      expect(result.database).toBe('db');
    });

    it('should trim whitespace from connection string', () => {
      const result = expectSuccess(
        ConnectionStringParser.parse('  postgresql://user:pass@host:5432/db  '),
      );

      expect(result.host).toBe('host');
    });
  });
});
