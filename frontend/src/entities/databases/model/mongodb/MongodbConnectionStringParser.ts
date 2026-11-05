import type { LocalizedText } from '../../../../shared/i18n';

export type ParseResult = {
  host: string;
  port: number;
  username: string;
  password: string;
  database: string;
  authDatabase: string;
  useTls: boolean;
  isSrv: boolean;
  isDirectConnection: boolean;
};

export type ParseError = {
  error: LocalizedText;
  format?: string;
};

// eslint-disable-next-line i18next/no-literal-string -- MongoDB's built-in authentication database
const DEFAULT_AUTH_DATABASE = 'admin';

export class MongodbConnectionStringParser {
  /**
   * Parses a MongoDB connection string in various formats.
   *
   * Supported formats:
   * 1. Standard MongoDB URI: mongodb://user:pass@host:port/db?authSource=admin
   * 2. MongoDB Atlas SRV: mongodb+srv://user:pass@cluster.mongodb.net/db
   * 3. Key-value format: host=x port=27017 database=db user=u password=p authSource=admin
   * 4. With TLS params: mongodb://user:pass@host:port/db?tls=true or ?ssl=true
   */
  static parse(connectionString: string): ParseResult | ParseError {
    const trimmed = connectionString.trim();

    if (!trimmed) {
      return { error: { key: 'databases.connectionString.errors.empty' } };
    }

    // Try key-value format (contains key=value pairs without ://)
    if (this.isKeyValueFormat(trimmed)) {
      return this.parseKeyValue(trimmed);
    }

    // Try URI format (mongodb:// or mongodb+srv://)
    if (trimmed.startsWith('mongodb://') || trimmed.startsWith('mongodb+srv://')) {
      return this.parseUri(trimmed);
    }

    return {
      error: { key: 'databases.connectionString.errors.unrecognizedFormat' },
    };
  }

  private static isKeyValueFormat(str: string): boolean {
    return (
      !str.includes('://') &&
      (str.includes('host=') || str.includes('database=')) &&
      str.includes('=')
    );
  }

  private static parseUri(connectionString: string): ParseResult | ParseError {
    try {
      const isSrv = connectionString.startsWith('mongodb+srv://');

      // Standard URI parsing using URL API
      const url = new URL(connectionString);

      const host = url.hostname;
      const port = url.port ? parseInt(url.port, 10) : isSrv ? 27017 : 27017;
      const username = decodeURIComponent(url.username);
      const rawPassword = decodeURIComponent(url.password);
      const password = this.isPasswordPlaceholder(rawPassword) ? '' : rawPassword;
      const database = decodeURIComponent(url.pathname.slice(1));
      const authDatabase = this.getAuthSource(url.search) || DEFAULT_AUTH_DATABASE;
      const useTls = isSrv ? true : this.checkTlsMode(url.search);
      const isDirectConnection = this.checkDirectConnection(url.search);

      if (!host) {
        return { error: { key: 'databases.connectionString.errors.hostMissing' } };
      }

      if (!username) {
        return { error: { key: 'databases.connectionString.errors.usernameMissing' } };
      }

      return {
        host,
        port,
        username,
        password,
        database: database || '',
        authDatabase,
        useTls,
        isSrv,
        isDirectConnection,
      };
    } catch {
      return {
        error: { key: 'databases.connectionString.errors.parseFailed' },
        format: 'URI',
      };
    }
  }

  private static parseKeyValue(connectionString: string): ParseResult | ParseError {
    try {
      const params: Record<string, string> = {};

      const regex = /(\w+)=(?:'([^']*)'|(\S+))/g;
      let match;

      while ((match = regex.exec(connectionString)) !== null) {
        const key = match[1];
        const value = match[2] !== undefined ? match[2] : match[3];
        params[key] = value;
      }

      const host = params['host'] || params['hostaddr'];
      const port = params['port'];
      const database = params['database'] || params['dbname'] || params['db'];
      const username = params['user'] || params['username'];
      const rawPassword = params['password'];
      const password = this.isPasswordPlaceholder(rawPassword) ? '' : rawPassword || '';
      const authDatabase = params['authSource'] || params['authDatabase'] || DEFAULT_AUTH_DATABASE;
      const tls = params['tls'] || params['ssl'];

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

      const useTls = this.isTlsEnabled(tls);
      const isDirectConnection = params['directConnection'] === 'true';

      return {
        host,
        port: port ? parseInt(port, 10) : 27017,
        username,
        password,
        database: database || '',
        authDatabase,
        useTls,
        isSrv: false,
        isDirectConnection,
      };
    } catch {
      return {
        error: { key: 'databases.connectionString.errors.keyValue.parseFailed' },
        format: 'key-value',
      };
    }
  }

  private static getAuthSource(queryString: string | undefined | null): string | undefined {
    if (!queryString) return undefined;

    const params = new URLSearchParams(
      queryString.startsWith('?') ? queryString.slice(1) : queryString,
    );

    return params.get('authSource') || params.get('authDatabase') || undefined;
  }

  private static checkDirectConnection(queryString: string | undefined | null): boolean {
    if (!queryString) return false;

    const params = new URLSearchParams(
      queryString.startsWith('?') ? queryString.slice(1) : queryString,
    );

    return params.get('directConnection') === 'true';
  }

  private static checkTlsMode(queryString: string | undefined | null): boolean {
    if (!queryString) return false;

    const params = new URLSearchParams(
      queryString.startsWith('?') ? queryString.slice(1) : queryString,
    );

    const tls = params.get('tls');
    const ssl = params.get('ssl');

    if (tls) return this.isTlsEnabled(tls);
    if (ssl) return this.isTlsEnabled(ssl);

    return false;
  }

  private static isTlsEnabled(tlsValue: string | null | undefined): boolean {
    if (!tlsValue) return false;

    const lowercased = tlsValue.toLowerCase();
    // eslint-disable-next-line i18next/no-literal-string -- connection string parameter values
    const enabledValues = ['true', 'yes', '1'];
    return enabledValues.includes(lowercased);
  }

  private static isPasswordPlaceholder(password: string | null | undefined): boolean {
    if (!password) return false;

    const trimmed = password.trim();
    return trimmed === '<db_password>' || trimmed === '<password>';
  }
}
