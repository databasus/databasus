import { http, HttpResponse } from 'msw';

import { type ApiComponents } from '../../api';
import { getApiMockPath } from './apiMockPath';
import type { Method, PathWith, Responses } from './apiOperation';
import { createPendingApiResponse } from './pendingApiResponse';
export { setUpMockApi } from './mockApiLifecycle';

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

interface MalformedApiResponse<M extends Method> {
  method: M;
  path: PathWith<M>;
  status: number;
  body: string;
}

interface DelayedApiResponse<
  M extends Method,
  Path extends PathWith<M>,
  S extends Status<Responses<Path, M>>,
> {
  method: M;
  path: Path;
  status: S;
  body: Body<Responses<Path, M>, S>;
}

const respond =
  <M extends Method>(method: M) =>
  <Path extends PathWith<M>, S extends Status<Responses<Path, M>>>(
    path: Path,
    status: S,
    body: Body<Responses<Path, M>, S>,
  ) =>
    http[method](`*${getApiMockPath(path)}`, () =>
      body === undefined ? new HttpResponse(null, { status }) : HttpResponse.json(body, { status }),
    );

// Each mock takes only a body the contract allows for its status: the declared body for a declared
// status, and a Problem for any other, so a mock of the wrong shape fails pnpm typecheck.
export const mockApi = {
  get: respond('get'),
  post: respond('post'),
  put: respond('put'),
  delete: respond('delete'),

  network: <M extends Method>(method: M, path: PathWith<M>) =>
    http[method](`*${getApiMockPath(path)}`, () => HttpResponse.error()),

  malformed: <M extends Method>({ method, path, status, body }: MalformedApiResponse<M>) =>
    http[method](
      `*${getApiMockPath(path)}`,
      () => new HttpResponse(body, { status, headers: { 'Content-Type': 'application/json' } }),
    ),

  pending: <M extends Method>(method: M, path: PathWith<M>) => {
    const pending = createPendingApiResponse();
    return {
      ...pending,
      handler: http[method](`*${getApiMockPath(path)}`, () => pending.response),
    };
  },

  delayed: <M extends Method, Path extends PathWith<M>, S extends Status<Responses<Path, M>>>({
    method,
    path,
    status,
    body,
  }: DelayedApiResponse<M, Path, S>) => {
    const pending = createPendingApiResponse();
    return {
      handler: http[method](`*${getApiMockPath(path)}`, () => pending.response),
      release: () =>
        pending.respond(
          body === undefined
            ? new HttpResponse(null, { status })
            : HttpResponse.json(body, { status }),
        ),
      cancel: pending.cancel,
    };
  },

  // An answer the contract cannot describe, such as the empty 502 of a proxy.
  getWithoutBody: (path: PathWith<'get'>, status: number) =>
    http.get(`*${getApiMockPath(path)}`, () => new HttpResponse(null, { status })),
};

// A Problem body for a refusal, so a test names only the code and the status it mocks.
export const buildProblem = (
  code: ApiComponents['schemas']['ProblemCode'],
  status: number,
): ApiComponents['schemas']['Problem'] => ({
  type: `urn:databasus:problem:${code}`,
  title: code,
  status,
  detail: code,
  code,
});
