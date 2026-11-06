import { matchRequestUrl } from 'msw';

import type { ApiPaths } from '@/shared/api';
import { getApiMockPath } from '@/shared/lib/test/apiMockPath';
import type { Method, PathWith } from '@/shared/lib/test/apiOperation';

type OperationParameters<
  Path extends keyof ApiPaths,
  M extends Method,
> = ApiPaths[Path][M] extends { parameters: infer Parameters } ? Parameters : never;
type ParameterGroup<Parameters, Group extends string> = Group extends keyof Parameters
  ? NonNullable<Parameters[Group]>
  : never;
type RequestBody<Path extends keyof ApiPaths, M extends Method> = ApiPaths[Path][M] extends {
  requestBody: { content: infer Content };
}
  ? Content[keyof Content]
  : undefined;

export const inspectApiRequest = async <M extends Method, Path extends PathWith<M>>(
  request: Request,
  operation: { method: M; path: Path },
) => {
  const url = new URL(request.url);
  const matched = matchRequestUrl(url, `${url.origin}${getApiMockPath(operation.path)}`);
  if (request.method.toLowerCase() !== operation.method || !matched.matches) {
    throw new Error(`Request does not match ${operation.method} ${operation.path}`);
  }
  return {
    headers: request.headers,
    params: matched.params as ParameterGroup<OperationParameters<Path, M>, 'path'>,
    query: {
      get: (key: Extract<keyof ParameterGroup<OperationParameters<Path, M>, 'query'>, string>) =>
        url.searchParams.get(key),
    },
    body: (request.body ? await request.clone().json() : undefined) as RequestBody<Path, M>,
  };
};
