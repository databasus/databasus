import { describe, expect, it } from 'vitest';

import { ConnectionErrorCode } from './ConnectionErrorCode';
import { physicalConnectionErrorContent } from './physicalConnectionErrorContent';

describe('physicalConnectionErrorContent', () => {
  it('quotes a generated replication role in the WAL switch grant command', () => {
    const buildSteps =
      physicalConnectionErrorContent[ConnectionErrorCode.NoWalSwitchPrivilege].buildSteps;

    expect(buildSteps?.({ username: 'databasus-ab12cd34' })).toContainEqual({
      type: 'command',
      command: 'GRANT EXECUTE ON FUNCTION pg_switch_wal() TO "databasus-ab12cd34";',
    });
  });
});
