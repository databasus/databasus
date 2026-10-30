import type { LocalizedText } from '../../../../shared/i18n';
import { PostgresSslMode } from './PostgresSslMode';

const IPV4_PATTERN = /^\d{1,3}(\.\d{1,3}){3}$/;

export type ParseResult = {
  host: string;
  port: number;
  username: string;
  password: string;
  database: string;
  sslMode: PostgresSslMode;
};

export type ParseError = {
  error: LocalizedText;
  format?: string;
};

export class ConnectionStringParser {
  /**
   * Parses a PostgreSQL connection string in various formats.
   *
   * Supported formats:
   * 1. Standard PostgreSQL URI: postgresql://user:pass@host:port/db
   * 2. Postgres URI: postgres://user:pass@host:port/db
   * 3. Supabase Direct: postgresql://postgres:pass@db.xxx.supabase.co:5432/postgres
   * 4. Supabase Pooler Session: postgres://postgres.ref:pass@aws-0-region.pooler.supabase.com:5432/postgres
   * 5. Supabase Pooler Transaction: same as above with port 6543
   * 6. JDBC: jdbc:postgresql://host:port/db?user=x&password=y
   * 7. Neon: postgresql://user:pass@ep-xxx.neon.tech/db
   * 8. Railway: postgresql://postgres:pass@xxx.railway.app:port/railway
   * 9. Render: postgresql://user:pass@xxx.render.com/db
   * 10. DigitalOcean: postgresql://user:pass@xxx.ondigitalocean.com:port/db?sslmode=require
   * 11. AWS RDS: postgresql://user:pass@xxx.rds.amazonaws.com:port/db
   * 12. Azure: postgresql://user@server:pass@xxx.postgres.database.azure.com:port/db?sslmode=require
   * 13. Heroku: postgres://user:pass@ec2-xxx.amazonaws.com:port/db
   * 14. CockroachDB: postgresql://user:pass@xxx.cockroachlabs.cloud:port/db?sslmode=verify-full
   * 15. With SSL params: postgresql://user:pass@host:port/db?sslmode=require
   * 16. libpq key-value: host=x port=5432 dbname=db user=u password=p
   */
  static parse(connectionString: string): ParseResult | ParseError {
    const trimmed = connectionString.trim();

    if (!trimmed) {
      return { error: { key: 'databases.connectionString.errors.empty' } };
    }

    // Try JDBC format first (starts with jdbc:)
    if (trimmed.startsWith('jdbc:postgresql://')) {
      return this.parseJdbc(trimmed);
    }

    // Try libpq key-value format (contains key=value pairs without ://)
    if (this.isLibpqFormat(trimmed)) {
      return this.parseLibpq(trimmed);
    }

    // Try URI format (postgresql:// or postgres://)
    if (trimmed.startsWith('postgresql://') || trimmed.startsWith('postgres://')) {
      return this.parseUri(trimmed);
    }

    return {
      error: { key: 'databases.connectionString.errors.unrecognizedFormat' },
    };
  }

  private static isLibpqFormat(str: string): boolean {
    // libpq format has key=value pairs separated by spaces
    // Must contain at least host= or dbname= to be considered libpq format
    return (
      !str.includes('://') &&
      (str.includes('host=') || str.includes('dbname=')) &&
      str.includes('=')
    );
  }

  private static parseUri(connectionString: string): ParseResult | ParseError {
    try {
      // Handle Azure format where username contains @: user@server:pass
      // Azure format: postgresql://user@servername:password@host:port/db
      const azureMatch = connectionString.match(
        /^postgres(?:ql)?:\/\/([^@:]+)@([^:]+):([^@]+)@([^:/?]+):?(\d+)?\/([^?]+)(?:\?(.*))?$/,
      );

      if (azureMatch) {
        const [, user, , password, host, port, database, queryString] = azureMatch;
        const sslMode = this.checkSslMode(queryString, host);

        return {
          host: host,
          port: port ? parseInt(port, 10) : 5432,
          username: decodeURIComponent(user),
          password: decodeURIComponent(password),
          database: decodeURIComponent(database),
          sslMode,
        };
      }

      // Standard URI parsing using URL API
      const url = new URL(connectionString);

      const host = url.hostname;
      const port = url.port ? parseInt(url.port, 10) : 5432;
      const username = decodeURIComponent(url.username);
      const password = decodeURIComponent(url.password);
      const database = decodeURIComponent(url.pathname.slice(1)); // Remove leading /
      const sslMode = this.checkSslMode(url.search, host);

      // Validate required fields
      if (!host) {
        return { error: { key: 'databases.connectionString.errors.hostMissing' } };
      }

      if (!username) {
        return { error: { key: 'databases.connectionString.errors.usernameMissing' } };
      }

      if (!password) {
        return { error: { key: 'databases.connectionString.errors.passwordMissing' } };
      }

      if (!database) {
        return { error: { key: 'databases.connectionString.errors.databaseMissing' } };
      }

      return {
        host,
        port,
        username,
        password,
        database,
        sslMode,
      };
    } catch {
      return {
        error: { key: 'databases.connectionString.errors.parseFailed' },
        format: 'URI',
      };
    }
  }

  private static parseJdbc(connectionString: string): ParseResult | ParseError {
    try {
      // JDBC format: jdbc:postgresql://host:port/database?user=x&password=y
      const jdbcRegex = /^jdbc:postgresql:\/\/([^:/?]+):?(\d+)?\/([^?]+)(?:\?(.*))?$/;
      const match = connectionString.match(jdbcRegex);

      if (!match) {
        return {
          error: {
            key: 'databases.connectionString.errors.jdbc.invalidFormat',
            params: { expectedFormat: 'jdbc:postgresql://host:port/database?user=x&password=y' },
          },
          format: 'JDBC',
        };
      }

      const [, host, port, database, queryString] = match;

      if (!queryString) {
        return {
          error: { key: 'databases.connectionString.errors.jdbc.queryParametersMissing' },
          format: 'JDBC',
        };
      }

      const params = new URLSearchParams(queryString);
      const username = params.get('user');
      const password = params.get('password');
      const sslMode = this.checkSslMode(queryString, host);

      if (!username) {
        return {
          error: { key: 'databases.connectionString.errors.jdbc.usernameMissing' },
          format: 'JDBC',
        };
      }

      if (!password) {
        return {
          error: { key: 'databases.connectionString.errors.jdbc.passwordMissing' },
          format: 'JDBC',
        };
      }

      return {
        host,
        port: port ? parseInt(port, 10) : 5432,
        username: decodeURIComponent(username),
        password: decodeURIComponent(password),
        database: decodeURIComponent(database),
        sslMode,
      };
    } catch {
      return {
        error: { key: 'databases.connectionString.errors.jdbc.parseFailed' },
        format: 'JDBC',
      };
    }
  }

  private static parseLibpq(connectionString: string): ParseResult | ParseError {
    try {
      // libpq format: host=x port=5432 dbname=db user=u password=p
      // Values can be quoted with single quotes: password='my pass'
      const params: Record<string, string> = {};

      // Match key=value or key='quoted value'
      const regex = /(\w+)=(?:'([^']*)'|(\S+))/g;
      let match;

      while ((match = regex.exec(connectionString)) !== null) {
        const key = match[1];
        const value = match[2] !== undefined ? match[2] : match[3];
        params[key] = value;
      }

      const host = params['host'] || params['hostaddr'];
      const port = params['port'];
      const database = params['dbname'] || params['database'];
      const username = params['user'] || params['username'];
      const password = params['password'];
      const sslmode = params['sslmode'];

      if (!host) {
        return {
          error: { key: 'databases.connectionString.errors.keyValue.hostMissing' },
          format: 'libpq',
        };
      }

      if (!username) {
        return {
          error: { key: 'databases.connectionString.errors.keyValue.usernameMissing' },
          format: 'libpq',
        };
      }

      if (!password) {
        return {
          error: { key: 'databases.connectionString.errors.keyValue.passwordMissing' },
          format: 'libpq',
        };
      }

      if (!database) {
        return {
          error: { key: 'databases.connectionString.errors.libpq.databaseMissing' },
          format: 'libpq',
        };
      }

      const sslMode = this.resolveSslMode(sslmode, host);

      return {
        host,
        port: port ? parseInt(port, 10) : 5432,
        username,
        password,
        database,
        sslMode,
      };
    } catch {
      return {
        error: { key: 'databases.connectionString.errors.libpq.parseFailed' },
        format: 'libpq',
      };
    }
  }

  private static checkSslMode(
    queryString: string | undefined | null,
    host: string,
  ): PostgresSslMode {
    const normalizedQuery = queryString?.startsWith('?')
      ? queryString.slice(1)
      : (queryString ?? '');
    const params = new URLSearchParams(normalizedQuery);

    return this.resolveSslMode(params.get('sslmode'), host);
  }

  private static resolveSslMode(sslmode: string | null | undefined, host: string): PostgresSslMode {
    return this.parseSslMode(sslmode) ?? this.deriveSslModeFromHost(host);
  }

  private static parseSslMode(sslmode: string | null | undefined): PostgresSslMode | null {
    switch (sslmode?.toLowerCase()) {
      case 'require':
        return PostgresSslMode.Require;
      case 'verify-ca':
        return PostgresSslMode.VerifyCa;
      case 'verify-full':
        return PostgresSslMode.VerifyFull;
      case 'disable':
      case 'prefer':
      case 'allow':
        return PostgresSslMode.Disable;
      default:
        return null;
    }
  }

  private static deriveSslModeFromHost(host: string): PostgresSslMode {
    const bareHost = host.trim().toLowerCase();

    if (bareHost === 'localhost' || IPV4_PATTERN.test(bareHost)) {
      return PostgresSslMode.Disable;
    }

    return PostgresSslMode.Require;
  }
}
