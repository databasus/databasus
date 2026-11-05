import { useLayoutEffect, useRef, useState } from 'react';

import {
  type DatabaseSettingsTarget,
  type DatabaseTestResult,
  databaseApi,
} from '@/entities/databases';
import { getSessionSnapshot, isCurrentSession, subscribeSession } from '@/shared/auth';
import { RequestRevision } from '@/shared/lib/RequestRevision';

export type DatabaseConnectionTestFailure =
  | { kind: 'result'; result: DatabaseTestResult }
  | { kind: 'api'; error: unknown };

interface Options<Change> {
  onChange: (change: Change) => void;
  onTestSucceeded: () => void;
}

export function useDatabaseConnectionTest<Change>(options: Options<Change>) {
  const [isTesting, setIsTesting] = useState(false);
  const [isTested, setIsTested] = useState(false);
  const [failure, setFailure] = useState<DatabaseConnectionTestFailure>();

  const withdrawTest = () => {
    revisions.current.test.Invalidate();
    isTestActive.current = false;
    setIsTesting(false);
    setIsTested(false);
    setFailure(undefined);
  };

  const changeConnection = (change: Change) => {
    revisions.current.settings.Invalidate();
    withdrawTest();
    callbacks.current.onChange(change);
  };

  // For a setting such as a schema or table selection, which a test does not depend on.
  const changeNonConnection = (change: Change) => {
    callbacks.current.onChange(change);
  };

  const testConnection = async (target: DatabaseSettingsTarget) => {
    if (isTestActive.current) return;

    const session = getSessionSnapshot();
    const contextRevision = revisions.current.context.GetCurrent();
    const settingsRevision = revisions.current.settings.GetCurrent();
    const testRevision = revisions.current.test.Invalidate();
    const isCurrent = () =>
      isCurrentSession(session) &&
      revisions.current.context.IsCurrent(contextRevision) &&
      revisions.current.settings.IsCurrent(settingsRevision) &&
      revisions.current.test.IsCurrent(testRevision);

    isTestActive.current = true;
    setIsTesting(true);
    setFailure(undefined);

    try {
      const testResult = await databaseApi.testDatabaseSettings(target);
      if (!isCurrent()) return;

      if (testResult.isConnected) {
        setIsTested(true);
        callbacks.current.onTestSucceeded();
      } else {
        setFailure({ kind: 'result', result: testResult });
      }
    } catch (e) {
      if (isCurrent()) setFailure({ kind: 'api', error: e });
    } finally {
      if (revisions.current.test.IsCurrent(testRevision)) {
        isTestActive.current = false;
        setIsTesting(false);
      }
    }
  };

  const revisions = useRef({
    context: new RequestRevision(),
    settings: new RequestRevision(),
    test: new RequestRevision(),
  });
  const isTestActive = useRef(false);
  const callbacks = useRef(options);

  useLayoutEffect(() => {
    callbacks.current = options;
  });

  useLayoutEffect(() => {
    const { context: contextRevision, test: testRevision } = revisions.current;
    const cancelForNewSession = () => {
      contextRevision.Invalidate();
      withdrawTest();
    };
    const unsubscribe = subscribeSession(cancelForNewSession);

    return () => {
      unsubscribe();
      contextRevision.Invalidate();
      testRevision.Invalidate();
      isTestActive.current = false;
    };
  }, []);

  return { isTesting, isTested, failure, testConnection, changeConnection, changeNonConnection };
}
