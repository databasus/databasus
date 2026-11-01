import { configure } from '@testing-library/react';
import { vi } from 'vitest';

// A waiter that gives up before the test limit reports the screen and its last failed check, which
// the test limit cannot do.
const CONDITION_WAIT_LIMIT_MS = 30_000;

configure({ asyncUtilTimeout: CONDITION_WAIT_LIMIT_MS });

// For a test that runs fake timers or has no document, where the waiter of Testing Library cannot
// run.
export const waitForCondition = <Result>(condition: () => Result | Promise<Result>) =>
  vi.waitFor(condition, { timeout: CONDITION_WAIT_LIMIT_MS });
