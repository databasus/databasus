import { cleanup } from '@testing-library/react';
import { HttpHandler, matchRequestUrl } from 'msw';
import { setupServer } from 'msw/node';
import { setTimeout, clearTimeout } from 'node:timers';
import { afterAll, afterEach, beforeAll, beforeEach, vi } from 'vitest';

import { cancelPendingApiResponses } from './pendingApiResponse';

const validateHandler = (handler: unknown) => {
  if (
    !(handler instanceof HttpHandler) ||
    typeof handler.info.method !== 'string' ||
    !/^(GET|POST|PUT|PATCH|DELETE|HEAD|OPTIONS)$/.test(handler.info.method) ||
    typeof handler.info.path !== 'string'
  ) {
    throw new Error('HTTP mocks require an explicit method and finite pathname');
  }
  const pathname = handler.info.path.startsWith('*')
    ? handler.info.path.slice(1)
    : new URL(handler.info.path, 'http://localhost').pathname;
  if (!pathname.startsWith('/') || /[ *+?(){}]/.test(pathname)) {
    throw new Error(`HTTP mocks require a finite pathname: ${handler.info.path}`);
  }

  const origin =
    handler.info.path.startsWith('*') || handler.info.path.startsWith('/')
      ? undefined
      : new URL(handler.info.path).origin;
  return { method: handler.info.method, pathname, origin };
};

export const setUpMockApi = () => {
  const server = setupServer();
  const use = server.use.bind(server);
  const resetHandlers = server.resetHandlers.bind(server);
  const requests: Request[] = [];
  const unexpectedRequests: string[] = [];
  const inFlight = new Map<
    string,
    { settled: Promise<void>; finish: () => void; request: Request }
  >();

  server.use = (...handlers) => {
    handlers.forEach(validateHandler);
    use(...handlers);
  };
  server.resetHandlers = (...handlers) => {
    handlers.forEach(validateHandler);
    resetHandlers(...handlers);
  };

  beforeAll(() =>
    server.listen({
      onUnhandledRequest: (request, print) => {
        const description = `${request.method} ${request.url}`;
        if (!unexpectedRequests.includes(description)) unexpectedRequests.push(description);
        for (const [requestId, pending] of inFlight) {
          if (pending.request === request) {
            pending.finish();
            inFlight.delete(requestId);
          }
        }
        print.error();
      },
    }),
  );

  beforeEach(() => {
    if (typeof document === 'undefined')
      vi.stubGlobal('window', { location: new URL('http://localhost') });
    // eslint-disable-next-line i18next/no-literal-string -- MSW lifecycle event
    server.events.on('request:start', ({ request, requestId }) => {
      requests.push(request.clone());
      const isDeclared = server.listHandlers().some((handler) => {
        const httpHandler = validateHandler(handler);
        const url = new URL(request.url);
        return (
          httpHandler.method === request.method &&
          (httpHandler.origin === undefined || httpHandler.origin === url.origin) &&
          matchRequestUrl(
            new URL(url.pathname, 'http://mock.local'),
            `http://mock.local${httpHandler.pathname}`,
          ).matches
        );
      });

      if (!isDeclared) {
        unexpectedRequests.push(`${request.method} ${request.url}`);
        return;
      }

      let finish = () => {};
      const settled = new Promise<void>((resolve) => {
        finish = resolve;
      });
      inFlight.set(requestId, { settled, finish, request });
    });

    const finishRequest = ({ requestId }: { requestId: string }) => {
      inFlight.get(requestId)?.finish();
      inFlight.delete(requestId);
    };
    // eslint-disable-next-line i18next/no-literal-string -- MSW lifecycle event
    server.events.on('request:end', finishRequest);
    server.events.on('unhandledException', finishRequest);
  });

  afterEach(async () => {
    const cleanupErrors: unknown[] = [];
    const violations: string[] = [];
    let cleanupTimer: ReturnType<typeof setTimeout> | undefined;

    try {
      try {
        if (typeof document !== 'undefined') cleanup();
      } finally {
        cancelPendingApiResponses();
        await Promise.race([
          Promise.all([...inFlight.values()].map((request) => request.settled)),
          new Promise<never>((_, reject) => {
            cleanupTimer = setTimeout(
              () =>
                reject(
                  new Error(
                    'HTTP cleanup timed out: pending handlers must use createPendingApiResponse',
                  ),
                ),
              5_000,
            );
          }),
        ]);
      }
    } catch (error) {
      cleanupErrors.push(error);
    } finally {
      clearTimeout(cleanupTimer);
      violations.push(...unexpectedRequests);
      resetHandlers();
      server.events.removeAllListeners();
      requests.length = 0;
      unexpectedRequests.length = 0;
      inFlight.clear();
      vi.unstubAllGlobals();
    }

    if (violations.length)
      cleanupErrors.push(new Error(`Unexpected HTTP requests:\n${violations.join('\n')}`));
    if (cleanupErrors.length === 1) throw cleanupErrors[0];
    if (cleanupErrors.length > 1)
      throw new Error(
        cleanupErrors
          .map((error) => (error instanceof Error ? error.message : String(error)))
          .join('\n'),
      );
  });

  afterAll(() => server.close());

  return Object.assign(server, { getRequests: () => [...requests] });
};
