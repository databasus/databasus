import type { Database } from './Database';
import { HealthStatus } from './HealthStatus';

// A database that cannot be reached needs attention first; the rest keep the order the server
// listed them in.
export const sortUnavailableDatabasesFirst = (databases: Database[]): Database[] =>
  [...databases].sort(
    (left, right) =>
      Number(right.healthStatus === HealthStatus.UNAVAILABLE) -
      Number(left.healthStatus === HealthStatus.UNAVAILABLE),
  );
