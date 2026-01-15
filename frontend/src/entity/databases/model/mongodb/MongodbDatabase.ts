import type { MongodbVersion } from './MongodbVersion';

export interface MongodbDatabase {
  id: string;
  version: MongodbVersion;
  host: string;
  port: number;
  username: string;
  password: string;
  database: string;
  authDatabase: string;
  isHttps: boolean;
  cpuCount: number;
  tlsCaFile?: string;
  tlsCertFile?: string;
  tlsCertKeyFile?: string;
}
