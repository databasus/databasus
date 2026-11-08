import { act } from '@testing-library/react';
import { expect } from 'vitest';

import type { setUpMockApi } from '@/shared/lib/test/mockApi';

export const finishCancelledApiResponse = async (
  server: ReturnType<typeof setUpMockApi>,
  pathname: string,
  respond: () => void,
) => {
  const cancelledRequest = server
    .getRequests()
    .find((request) => new URL(request.url).pathname === pathname && request.signal.aborted);
  expect(cancelledRequest).toBeDefined();

  await act(async () => {
    respond();
  });
};
