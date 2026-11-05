import type { ApiPaths } from '../../api';

export type Method = 'get' | 'post' | 'put' | 'delete';

export type Responses<Path extends keyof ApiPaths, M extends Method> = [ApiPaths[Path][M]] extends [
  undefined | never,
]
  ? never
  : ApiPaths[Path][M] extends {
        responses: infer DeclaredResponses;
      }
    ? DeclaredResponses
    : never;

export type PathWith<M extends Method> = {
  [Path in keyof ApiPaths]: [Responses<Path, M>] extends [never] ? never : Path;
}[keyof ApiPaths];
