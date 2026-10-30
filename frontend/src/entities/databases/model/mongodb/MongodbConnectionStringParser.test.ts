import { describe, expect, it } from 'vitest';

import {
  MongodbConnectionStringParser,
  type ParseError,
  type ParseResult,
} from './MongodbConnectionStringParser';

describe('MongodbConnectionStringParser', () => {
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

  describe('Standard MongoDB URI (mongodb://)', () => {
    it('should parse basic mongodb:// connection string', () => {
      const result = expectSuccess(
        MongodbConnectionStringParser.parse('mongodb://myuser:mypassword@localhost:27017/mydb'),
      );

      expect(result.host).toBe('localhost');
      expect(result.port).toBe(27017);
      expect(result.username).toBe('myuser');
      expect(result.password).toBe('mypassword');
      expect(result.database).toBe('mydb');
      expect(result.authDatabase).toBe('admin');
      expect(result.useTls).toBe(false);
      expect(result.isSrv).toBe(false);
    });

    it('should parse connection string without database', () => {
      const result = expectSuccess(
        MongodbConnectionStringParser.parse('mongodb://root:rostislav123!@82.146.56.0:27017'),
      );

      expect(result.host).toBe('82.146.56.0');
      expect(result.port).toBe(27017);
      expect(result.username).toBe('root');
      expect(result.password).toBe('rostislav123!');
      expect(result.database).toBe('');
      expect(result.authDatabase).toBe('admin');
      expect(result.useTls).toBe(false);
      expect(result.isSrv).toBe(false);
    });

    it('should default port to 27017 when not specified', () => {
      const result = expectSuccess(
        MongodbConnectionStringParser.parse('mongodb://user:pass@host/db'),
      );

      expect(result.port).toBe(27017);
    });

    it('should handle URL-encoded passwords', () => {
      const result = expectSuccess(
        MongodbConnectionStringParser.parse('mongodb://user:p%40ss%23word@host:27017/db'),
      );

      expect(result.password).toBe('p@ss#word');
    });

    it('should handle URL-encoded usernames', () => {
      const result = expectSuccess(
        MongodbConnectionStringParser.parse('mongodb://user%40domain:password@host:27017/db'),
      );

      expect(result.username).toBe('user@domain');
    });

    it('should parse authSource from query string', () => {
      const result = expectSuccess(
        MongodbConnectionStringParser.parse(
          'mongodb://user:pass@host:27017/mydb?authSource=authdb',
        ),
      );

      expect(result.authDatabase).toBe('authdb');
    });

    it('should parse authDatabase from query string', () => {
      const result = expectSuccess(
        MongodbConnectionStringParser.parse(
          'mongodb://user:pass@host:27017/mydb?authDatabase=authdb',
        ),
      );

      expect(result.authDatabase).toBe('authdb');
    });
  });

  describe('MongoDB Atlas SRV URI (mongodb+srv://)', () => {
    it('should parse mongodb+srv:// connection string', () => {
      const result = expectSuccess(
        MongodbConnectionStringParser.parse(
          'mongodb+srv://atlasuser:atlaspass@cluster0.abc123.mongodb.net/mydb',
        ),
      );

      expect(result.host).toBe('cluster0.abc123.mongodb.net');
      expect(result.port).toBe(27017);
      expect(result.username).toBe('atlasuser');
      expect(result.password).toBe('atlaspass');
      expect(result.database).toBe('mydb');
      expect(result.useTls).toBe(true); // SRV connections use TLS by default
      expect(result.isSrv).toBe(true);
    });

    it('should parse mongodb+srv:// without database', () => {
      const result = expectSuccess(
        MongodbConnectionStringParser.parse(
          'mongodb+srv://atlasuser:atlaspass@cluster0.abc123.mongodb.net',
        ),
      );

      expect(result.host).toBe('cluster0.abc123.mongodb.net');
      expect(result.database).toBe('');
      expect(result.useTls).toBe(true);
      expect(result.isSrv).toBe(true);
    });
  });

  describe('TLS/SSL Mode Handling', () => {
    it('should set useTls=true for tls=true', () => {
      const result = expectSuccess(
        MongodbConnectionStringParser.parse('mongodb://u:p@host:27017/db?tls=true'),
      );

      expect(result.useTls).toBe(true);
    });

    it('should set useTls=true for ssl=true', () => {
      const result = expectSuccess(
        MongodbConnectionStringParser.parse('mongodb://u:p@host:27017/db?ssl=true'),
      );

      expect(result.useTls).toBe(true);
    });

    it('should set useTls=true for tls=yes', () => {
      const result = expectSuccess(
        MongodbConnectionStringParser.parse('mongodb://u:p@host:27017/db?tls=yes'),
      );

      expect(result.useTls).toBe(true);
    });

    it('should set useTls=true for tls=1', () => {
      const result = expectSuccess(
        MongodbConnectionStringParser.parse('mongodb://u:p@host:27017/db?tls=1'),
      );

      expect(result.useTls).toBe(true);
    });

    it('should set useTls=false for tls=false', () => {
      const result = expectSuccess(
        MongodbConnectionStringParser.parse('mongodb://u:p@host:27017/db?tls=false'),
      );

      expect(result.useTls).toBe(false);
    });

    it('should set useTls=false when no tls/ssl specified', () => {
      const result = expectSuccess(
        MongodbConnectionStringParser.parse('mongodb://u:p@host:27017/db'),
      );

      expect(result.useTls).toBe(false);
    });
  });

  describe('Key-Value Format', () => {
    it('should parse key-value format connection string', () => {
      const result = expectSuccess(
        MongodbConnectionStringParser.parse(
          'host=localhost port=27017 database=mydb user=admin password=secret',
        ),
      );

      expect(result.host).toBe('localhost');
      expect(result.port).toBe(27017);
      expect(result.username).toBe('admin');
      expect(result.password).toBe('secret');
      expect(result.database).toBe('mydb');
    });

    it('should parse key-value format without database', () => {
      const result = expectSuccess(
        MongodbConnectionStringParser.parse('host=localhost port=27017 user=admin password=secret'),
      );

      expect(result.host).toBe('localhost');
      expect(result.database).toBe('');
    });

    it('should parse key-value format with quoted password containing spaces', () => {
      const result = expectSuccess(
        MongodbConnectionStringParser.parse(
          "host=localhost port=27017 database=mydb user=admin password='my secret pass'",
        ),
      );

      expect(result.password).toBe('my secret pass');
    });

    it('should default port to 27017 when not specified in key-value format', () => {
      const result = expectSuccess(
        MongodbConnectionStringParser.parse(
          'host=localhost database=mydb user=admin password=secret',
        ),
      );

      expect(result.port).toBe(27017);
    });

    it('should handle hostaddr as alternative to host', () => {
      const result = expectSuccess(
        MongodbConnectionStringParser.parse(
          'hostaddr=192.168.1.1 port=27017 database=mydb user=admin password=secret',
        ),
      );

      expect(result.host).toBe('192.168.1.1');
    });

    it('should handle dbname as alternative to database', () => {
      const result = expectSuccess(
        MongodbConnectionStringParser.parse(
          'host=localhost port=27017 dbname=mydb user=admin password=secret',
        ),
      );

      expect(result.database).toBe('mydb');
    });

    it('should handle db as alternative to database', () => {
      const result = expectSuccess(
        MongodbConnectionStringParser.parse(
          'host=localhost port=27017 db=mydb user=admin password=secret',
        ),
      );

      expect(result.database).toBe('mydb');
    });

    it('should handle username as alternative to user', () => {
      const result = expectSuccess(
        MongodbConnectionStringParser.parse(
          'host=localhost port=27017 database=mydb username=admin password=secret',
        ),
      );

      expect(result.username).toBe('admin');
    });

    it('should parse authSource in key-value format', () => {
      const result = expectSuccess(
        MongodbConnectionStringParser.parse(
          'host=localhost database=mydb user=admin password=secret authSource=authdb',
        ),
      );

      expect(result.authDatabase).toBe('authdb');
    });

    it('should parse authDatabase in key-value format', () => {
      const result = expectSuccess(
        MongodbConnectionStringParser.parse(
          'host=localhost database=mydb user=admin password=secret authDatabase=authdb',
        ),
      );

      expect(result.authDatabase).toBe('authdb');
    });

    it('should parse tls in key-value format', () => {
      const result = expectSuccess(
        MongodbConnectionStringParser.parse(
          'host=localhost database=mydb user=admin password=secret tls=true',
        ),
      );

      expect(result.useTls).toBe(true);
    });

    it('should parse ssl in key-value format', () => {
      const result = expectSuccess(
        MongodbConnectionStringParser.parse(
          'host=localhost database=mydb user=admin password=secret ssl=true',
        ),
      );

      expect(result.useTls).toBe(true);
    });

    it('should return error for key-value format missing host', () => {
      const result = expectError(
        MongodbConnectionStringParser.parse('port=27017 database=mydb user=admin password=secret'),
      );

      expect(result.error.key).toBe('databases.connectionString.errors.keyValue.hostMissing');
      expect(result.format).toBe('key-value');
    });

    it('should return error for key-value format missing user', () => {
      const result = expectError(
        MongodbConnectionStringParser.parse('host=localhost database=mydb password=secret'),
      );

      expect(result.error.key).toBe('databases.connectionString.errors.keyValue.usernameMissing');
      expect(result.format).toBe('key-value');
    });

    it('should allow missing password in key-value format (returns empty password)', () => {
      const result = expectSuccess(
        MongodbConnectionStringParser.parse('host=localhost database=mydb user=admin'),
      );

      expect(result.host).toBe('localhost');
      expect(result.username).toBe('admin');
      expect(result.password).toBe('');
      expect(result.database).toBe('mydb');
    });
  });

  describe('Error Cases', () => {
    it('should return error for empty string', () => {
      const result = expectError(MongodbConnectionStringParser.parse(''));

      expect(result.error.key).toBe('databases.connectionString.errors.empty');
    });

    it('should return error for whitespace-only string', () => {
      const result = expectError(MongodbConnectionStringParser.parse('   '));

      expect(result.error.key).toBe('databases.connectionString.errors.empty');
    });

    it('should return error for unrecognized format', () => {
      const result = expectError(MongodbConnectionStringParser.parse('some random text'));

      expect(result.error.key).toBe('databases.connectionString.errors.unrecognizedFormat');
    });

    it('should return error for missing username in URI', () => {
      const result = expectError(
        MongodbConnectionStringParser.parse('mongodb://:password@host:27017/db'),
      );

      expect(result.error.key).toBe('databases.connectionString.errors.usernameMissing');
    });

    it('should allow missing password in URI (returns empty password)', () => {
      const result = expectSuccess(
        MongodbConnectionStringParser.parse('mongodb://user@host:27017/db'),
      );

      expect(result.username).toBe('user');
      expect(result.password).toBe('');
      expect(result.host).toBe('host');
      expect(result.database).toBe('db');
    });

    it('should return error for mysql:// format (wrong database type)', () => {
      const result = expectError(
        MongodbConnectionStringParser.parse('mysql://user:pass@host:3306/db'),
      );

      expect(result.error.key).toBe('databases.connectionString.errors.unrecognizedFormat');
    });

    it('should return error for postgresql:// format (wrong database type)', () => {
      const result = expectError(
        MongodbConnectionStringParser.parse('postgresql://user:pass@host:5432/db'),
      );

      expect(result.error.key).toBe('databases.connectionString.errors.unrecognizedFormat');
    });
  });

  describe('Edge Cases', () => {
    it('should handle special characters in password', () => {
      const result = expectSuccess(
        MongodbConnectionStringParser.parse('mongodb://user:p%40ss%3Aw%2Ford@host:27017/db'),
      );

      expect(result.password).toBe('p@ss:w/ord');
    });

    it('should handle password with exclamation mark', () => {
      const result = expectSuccess(
        MongodbConnectionStringParser.parse('mongodb://root:rostislav123!@82.146.56.0:27017'),
      );

      expect(result.password).toBe('rostislav123!');
    });

    it('should handle numeric database names', () => {
      const result = expectSuccess(
        MongodbConnectionStringParser.parse('mongodb://user:pass@host:27017/12345'),
      );

      expect(result.database).toBe('12345');
    });

    it('should handle hyphenated host names', () => {
      const result = expectSuccess(
        MongodbConnectionStringParser.parse(
          'mongodb://user:pass@my-database-host.example.com:27017/db',
        ),
      );

      expect(result.host).toBe('my-database-host.example.com');
    });

    it('should handle IP address as host', () => {
      const result = expectSuccess(
        MongodbConnectionStringParser.parse('mongodb://user:pass@192.168.1.100:27017/db'),
      );

      expect(result.host).toBe('192.168.1.100');
    });

    it('should handle connection string with extra query parameters', () => {
      const result = expectSuccess(
        MongodbConnectionStringParser.parse(
          'mongodb://user:pass@host:27017/db?tls=true&connectTimeoutMS=10000&retryWrites=true',
        ),
      );

      expect(result.useTls).toBe(true);
      expect(result.database).toBe('db');
    });

    it('should trim whitespace from connection string', () => {
      const result = expectSuccess(
        MongodbConnectionStringParser.parse('  mongodb://user:pass@host:27017/db  '),
      );

      expect(result.host).toBe('host');
    });

    it('should handle trailing slash without database', () => {
      const result = expectSuccess(
        MongodbConnectionStringParser.parse('mongodb://user:pass@host:27017/'),
      );

      expect(result.database).toBe('');
    });
  });

  describe('Direct Connection Handling', () => {
    it('should parse directConnection=true from URI', () => {
      const result = expectSuccess(
        MongodbConnectionStringParser.parse(
          'mongodb://user:pass@host:27017/db?authSource=admin&directConnection=true',
        ),
      );

      expect(result.isDirectConnection).toBe(true);
    });

    it('should default isDirectConnection to false when not specified in URI', () => {
      const result = expectSuccess(
        MongodbConnectionStringParser.parse('mongodb://user:pass@host:27017/db'),
      );

      expect(result.isDirectConnection).toBe(false);
    });

    it('should parse isDirectConnection=true from key-value format', () => {
      const result = expectSuccess(
        MongodbConnectionStringParser.parse(
          'host=localhost port=27017 database=mydb user=admin password=secret directConnection=true',
        ),
      );

      expect(result.isDirectConnection).toBe(true);
    });

    it('should default isDirectConnection to false in key-value format when not specified', () => {
      const result = expectSuccess(
        MongodbConnectionStringParser.parse(
          'host=localhost port=27017 database=mydb user=admin password=secret',
        ),
      );

      expect(result.isDirectConnection).toBe(false);
    });
  });

  describe('Password Placeholder Handling', () => {
    it('should treat <db_password> placeholder as empty password in URI format', () => {
      const result = expectSuccess(
        MongodbConnectionStringParser.parse('mongodb://user:<db_password>@host:27017/db'),
      );

      expect(result.username).toBe('user');
      expect(result.password).toBe('');
      expect(result.host).toBe('host');
      expect(result.database).toBe('db');
    });

    it('should treat <password> placeholder as empty password in URI format', () => {
      const result = expectSuccess(
        MongodbConnectionStringParser.parse('mongodb://user:<password>@host:27017/db'),
      );

      expect(result.username).toBe('user');
      expect(result.password).toBe('');
      expect(result.host).toBe('host');
      expect(result.database).toBe('db');
    });

    it('should treat <db_password> placeholder as empty password in SRV format', () => {
      const result = expectSuccess(
        MongodbConnectionStringParser.parse(
          'mongodb+srv://user:<db_password>@cluster0.mongodb.net/db',
        ),
      );

      expect(result.username).toBe('user');
      expect(result.password).toBe('');
      expect(result.host).toBe('cluster0.mongodb.net');
      expect(result.isSrv).toBe(true);
    });

    it('should treat <db_password> placeholder as empty password in key-value format', () => {
      const result = expectSuccess(
        MongodbConnectionStringParser.parse(
          'host=localhost database=mydb user=admin password=<db_password>',
        ),
      );

      expect(result.host).toBe('localhost');
      expect(result.username).toBe('admin');
      expect(result.password).toBe('');
      expect(result.database).toBe('mydb');
    });

    it('should treat <password> placeholder as empty password in key-value format', () => {
      const result = expectSuccess(
        MongodbConnectionStringParser.parse(
          'host=localhost database=mydb user=admin password=<password>',
        ),
      );

      expect(result.host).toBe('localhost');
      expect(result.username).toBe('admin');
      expect(result.password).toBe('');
      expect(result.database).toBe('mydb');
    });
  });
});
