import type { LocalizedText } from '../../../../shared/i18n';

export type ParseResult = {
  host: string;
  port: number;
  username: string;
  password: string;
  database: string;
  isHttps: boolean;
};

export type ParseError = {
  error: LocalizedText;
  format?: string;
};

export class MySqlConnectionStringParser {
  /**
   * Parses a MySQL connection string in various formats.
   *
   * Supported formats:
   * 1. Standard MySQL URI: mysql://user:pass@host:port/db
   * 2. JDBC format: jdbc:mysql://host:port/db?user=x&password=y
   * 3. Key-value format: host=x port=3306 database=db user=u password=p
   * 4. With SSL params: mysql://user:pass@host:port/db?ssl=true or ?sslMode=REQUIRED
   * 5. AWS RDS: mysql://user:pass@xxx.rds.amazonaws.com:3306/db
   * 6. PlanetScale: mysql://user:pass@xxx.connect.psdb.cloud/db?sslaccept=strict
   * 7. DigitalOcean: mysql://user:pass@xxx.ondigitalocean.com:25060/db?ssl-mode=REQUIRED
   * 8. Azure MySQL: mysql://user@servername:pass@xxx.mysql.database.azure.com:3306/db
   * 9. Railway: mysql://user:pass@xxx.railway.app:port/db
   */
  static parse(connectionString: string): ParseResult | ParseError {
    const trimmed = connectionString.trim();

    if (!trimmed) {
      return { error: { key: 'databases.connectionString.errors.empty' } };
    }

    // Try JDBC format first (starts with jdbc:)
    if (trimmed.startsWith('jdbc:mysql://')) {
      return this.parseJdbc(trimmed);
    }

    // Try key-value format (contains key=value pairs without ://)
    if (this.isKeyValueFormat(trimmed)) {
      return this.parseKeyValue(trimmed);
    }

    // Try URI format (mysql://)
    if (trimmed.startsWith('mysql://')) {
      return this.parseUri(trimmed);
    }

    return {
      error: { key: 'databases.connectionString.errors.unrecognizedFormat' },
    };
  }

  private static isKeyValueFormat(str: string): boolean {
    // Key-value format has key=value pairs separated by spaces
    // Must contain at least host= or database= to be considered key-value format
    return (
      !str.includes('://') &&
      (str.includes('host=') || str.includes('database=')) &&
      str.includes('=')
    );
  }

  private static parseUri(connectionString: string): ParseResult | ParseError {
    try {
      // Handle Azure format where username contains @: user@server:pass
      // Azure format: mysql://user@servername:password@host:port/db
      const azureMatch = connectionString.match(
        /^mysql:\/\/([^@:]+)@([^:]+):([^@]+)@([^:/?]+):?(\d+)?\/([^?]+)(?:\?(.*))?$/,
      );

      if (azureMatch) {
        const [, user, , password, host, port, database, queryString] = azureMatch;
        const isHttps = this.checkSslMode(queryString);

        return {
          host: host,
          port: port ? parseInt(port, 10) : 3306,
          username: decodeURIComponent(user),
          password: decodeURIComponent(password),
          database: decodeURIComponent(database),
          isHttps,
        };
      }

      // Standard URI parsing using URL API
      const url = new URL(connectionString);

      const host = url.hostname;
      const port = url.port ? parseInt(url.port, 10) : 3306;
      const username = decodeURIComponent(url.username);
      const password = decodeURIComponent(url.password);
      const database = decodeURIComponent(url.pathname.slice(1)); // Remove leading /
      const isHttps = this.checkSslMode(url.search);

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
        isHttps,
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
      // JDBC format: jdbc:mysql://host:port/database?user=x&password=y
      const jdbcRegex = /^jdbc:mysql:\/\/([^:/?]+):?(\d+)?\/([^?]+)(?:\?(.*))?$/;
      const match = connectionString.match(jdbcRegex);

      if (!match) {
        return {
          error: {
            key: 'databases.connectionString.errors.jdbc.invalidFormat',
            params: { expectedFormat: 'jdbc:mysql://host:port/database?user=x&password=y' },
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
      const isHttps = this.checkSslMode(queryString);

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
        port: port ? parseInt(port, 10) : 3306,
        username: decodeURIComponent(username),
        password: decodeURIComponent(password),
        database: decodeURIComponent(database),
        isHttps,
      };
    } catch {
      return {
        error: { key: 'databases.connectionString.errors.jdbc.parseFailed' },
        format: 'JDBC',
      };
    }
  }

  private static parseKeyValue(connectionString: string): ParseResult | ParseError {
    try {
      // Key-value format: host=x port=3306 database=db user=u password=p
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
      const database = params['database'] || params['dbname'];
      const username = params['user'] || params['username'];
      const password = params['password'];
      const ssl = params['ssl'] || params['sslMode'] || params['ssl-mode'] || params['useSSL'];

      if (!host) {
        return {
          error: { key: 'databases.connectionString.errors.keyValue.hostMissing' },
          format: 'key-value',
        };
      }

      if (!username) {
        return {
          error: { key: 'databases.connectionString.errors.keyValue.usernameMissing' },
          format: 'key-value',
        };
      }

      if (!password) {
        return {
          error: { key: 'databases.connectionString.errors.keyValue.passwordMissing' },
          format: 'key-value',
        };
      }

      if (!database) {
        return {
          error: { key: 'databases.connectionString.errors.keyValue.databaseMissing' },
          format: 'key-value',
        };
      }

      const isHttps = this.isSslEnabled(ssl);

      return {
        host,
        port: port ? parseInt(port, 10) : 3306,
        username,
        password,
        database,
        isHttps,
      };
    } catch {
      return {
        error: { key: 'databases.connectionString.errors.keyValue.parseFailed' },
        format: 'key-value',
      };
    }
  }

  private static checkSslMode(queryString: string | undefined | null): boolean {
    if (!queryString) return false;

    const params = new URLSearchParams(
      queryString.startsWith('?') ? queryString.slice(1) : queryString,
    );

    // Check various MySQL SSL parameter names
    const ssl = params.get('ssl');
    const sslMode = params.get('sslMode');
    const sslModeHyphen = params.get('ssl-mode');
    const useSSL = params.get('useSSL');
    const sslaccept = params.get('sslaccept');

    if (ssl) return this.isSslEnabled(ssl);
    if (sslMode) return this.isSslEnabled(sslMode);
    if (sslModeHyphen) return this.isSslEnabled(sslModeHyphen);
    if (useSSL) return this.isSslEnabled(useSSL);
    if (sslaccept) return sslaccept.toLowerCase() === 'strict';

    return false;
  }

  private static isSslEnabled(sslValue: string | null | undefined): boolean {
    if (!sslValue) return false;

    const lowercased = sslValue.toLowerCase();

    // These values indicate SSL is enabled
    // eslint-disable-next-line i18next/no-literal-string -- connection string parameter values
    const enabledValues = ['true', 'required', 'verify_ca', 'verify_identity', 'yes', '1'];
    return enabledValues.includes(lowercased);
  }
}
