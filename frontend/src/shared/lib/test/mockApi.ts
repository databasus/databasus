import { http, HttpResponse } from 'msw';
import { setupServer } from 'msw/node';
import { afterAll, afterEach, beforeAll, vi } from 'vitest';

import { API_BASE_PATH, type ApiPaths } from '../../api';

type Method = 'get' | 'post' | 'put' | 'delete';

type Responses<Path extends keyof ApiPaths, M extends Method> = ApiPaths[Path][M] extends {
  responses: infer DeclaredResponses;
}
  ? DeclaredResponses
  : never;

type PathWith<M extends Method> = {
  [Path in keyof ApiPaths]: [Responses<Path, M>] extends [never] ? never : Path;
}[keyof ApiPaths];

// Any status is allowed when the operation has a default response, which covers the undeclared ones.
type Status<DeclaredResponses> = 'default' extends keyof DeclaredResponses
  ? number
  : Extract<keyof DeclaredResponses, number>;

type ContentOf<Response> = Response extends { content: infer Content }
  ? Content[keyof Content]
  : undefined;

type Body<DeclaredResponses, S> = S extends keyof DeclaredResponses
  ? ContentOf<DeclaredResponses[S]>
  : 'default' extends keyof DeclaredResponses
    ? ContentOf<DeclaredResponses['default']>
    : never;

const toMswPath = (path: string) => `*${API_BASE_PATH}${path.replace(/\{([^}]+)\}/g, ':$1')}`;

const respond =
  <M extends Method>(method: M) =>
  <Path extends PathWith<M>, S extends Status<Responses<Path, M>>>(
    path: Path,
    status: S,
    body: Body<Responses<Path, M>, S>,
  ) =>
    http[method](toMswPath(path), () => HttpResponse.json(body, { status }));

// Each mock takes only a body the contract allows for its status: the declared body for a declared
// status, and a Problem for any other, so a mock of the wrong shape fails pnpm typecheck.
export const mockApi = {
  get: respond('get'),

  // An answer the contract cannot describe, such as the empty 502 of a proxy.
  getWithoutBody: (path: PathWith<'get'>, status: number) =>
    http.get(toMswPath(path), () => new HttpResponse(null, { status })),
};

// Starts the network mock for the test file. Requests without a mock fail the test, and the
// application server resolves to localhost, because the tests run without a browser window.
export const setUpMockApi = () => {
  const server = setupServer();

  beforeAll(() => {
    vi.stubGlobal('window', { location: new URL('http://localhost') });
    server.listen({ onUnhandledRequest: 'error' });
  });

  afterEach(() => server.resetHandlers());

  afterAll(() => {
    server.close();
    vi.unstubAllGlobals();
  });

  return server;
};
