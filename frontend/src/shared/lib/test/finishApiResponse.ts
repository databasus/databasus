import { act } from '@testing-library/react';

import type { setUpMockApi } from '@/shared/lib/test/mockApi';

export const finishApiResponse = async (
  server: ReturnType<typeof setUpMockApi>,
  pathname: string,
  respond: () => void,
) => {
  const processed = new Promise<void>((resolve) => {
    const onResponse = ({ request }: { request: Request }) => {
      if (new URL(request.url).pathname !== pathname) return;

      // eslint-disable-next-line i18next/no-literal-string -- MSW lifecycle event
      server.events.removeListener('response:mocked', onResponse);
      resolve();
    };
    // eslint-disable-next-line i18next/no-literal-string -- MSW lifecycle event
    server.events.on('response:mocked', onResponse);
  });

  await act(async () => {
    respond();
    await processed;
  });
};
