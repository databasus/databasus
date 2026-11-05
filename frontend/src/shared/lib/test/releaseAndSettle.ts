import { act } from '@testing-library/react';
import type { MockInstance } from 'vitest';

// A negative check needs a sign that the answer was handled. `apiCall` is a pass-through spy on the
// API method the held request belongs to: the screen awaited its promise first, so the screen's
// continuation has run once this wait ends. By default it follows the latest call of the method.
export const releaseAndSettle = async (
  apiCall: MockInstance,
  release: () => void,
  callIndex = apiCall.mock.results.length - 1,
) => {
  await act(async () => {
    release();
    await Promise.allSettled([apiCall.mock.results[callIndex].value]);
  });
};
