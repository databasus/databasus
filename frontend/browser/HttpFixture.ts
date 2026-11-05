import type { Page } from '@playwright/test';

import type { Method, PathWith, Responses } from '../src/shared/lib/test/apiOperation';

type Body<Response> = Response extends { content: infer Content }
  ? Content[keyof Content]
  : undefined;
export interface HttpFixture {
  method: string;
  path: string;
  status: number;
  body: unknown;
}
export const apiResponse = <
  M extends Method,
  P extends PathWith<M>,
  S extends Extract<keyof Responses<P, M>, number>,
>({
  method,
  path,
  status,
  body,
}: {
  method: M;
  path: P;
  status: S;
  body: Body<Responses<P, M>[S]>;
}): HttpFixture => ({ method: method.toUpperCase(), path, status, body });

export const validateHttpFixtures = (fixtures: HttpFixture[]) => {
  const identities = new Set<string>();
  for (const fixture of fixtures) {
    const path = fixture.path.startsWith('https://')
      ? new URL(fixture.path).pathname
      : fixture.path;
    if (
      !/^(?:GET|POST|PUT|DELETE)$/.test(fixture.method) ||
      !path.startsWith('/') ||
      /[ *+?()]/.test(path)
    )
      throw new Error(
        `HTTP fixtures require an explicit method and finite pathname: ${fixture.method} ${fixture.path}`,
      );

    const identity = `${fixture.method} ${fixture.path}`;
    if (identities.has(identity)) throw new Error(`Duplicate HTTP fixture: ${identity}`);
    identities.add(identity);
  }
};

export const installHttpFixtures = async ({
  page,
  fixtures,
}: {
  page: Page;
  fixtures: HttpFixture[];
}) => {
  validateHttpFixtures(fixtures);
  const resourceUrls: string[] = [];
  const context = page.context();
  context.on('request', (request) => resourceUrls.push(request.url()));
  const unexpected: string[] = [];
  const requests: { method: string; path: string; body: unknown }[] = [];
  const pageErrors: string[] = [];
  const observePage = (observedPage: Page) =>
    observedPage.on('pageerror', (error) => pageErrors.push(error.message));
  observePage(page);
  context.on('page', observePage);

  await context.route('**/*', async (route) => {
    const request = route.request();
    const url = new URL(request.url());
    const applicationOrigin = new URL(process.env.BROWSER_BASE_URL!).origin;

    if (
      url.origin === applicationOrigin &&
      !/^\/api(?:\/|$)/.test(url.pathname) &&
      request.method() === 'GET' &&
      (request.isNavigationRequest() ||
        ['script', 'stylesheet', 'image', 'font', 'media', 'manifest'].includes(
          request.resourceType(),
        ))
    )
      return route.continue();

    const pathname = url.pathname.replace(/^\/api\/v1/, '');
    const fixture = fixtures.find(
      (candidate) =>
        candidate.method === request.method() &&
        (candidate.path.startsWith('https://')
          ? candidate.path === `${url.origin}${url.pathname}`
          : url.origin === applicationOrigin &&
            url.pathname.startsWith('/api/v1/') &&
            // Fixture path literals are escaped before named path parameters are substituted.
            // nosemgrep: javascript.lang.security.audit.detect-non-literal-regexp.detect-non-literal-regexp
            new RegExp(
              `^${candidate.path.replace(/[.*+?^$()|[\]\\]/g, '\\$&').replace(/\{[^}]+\}/g, '[^/]+')}$`,
            ).test(pathname)),
    );

    requests.push({
      method: request.method(),
      path: pathname,
      body: fixture
        ? request.headers()['content-type']?.startsWith('application/x-www-form-urlencoded')
          ? Object.fromEntries(new URLSearchParams(request.postData() ?? ''))
          : request.postDataJSON()
        : request.postData(),
    });
    if (!fixture) {
      unexpected.push(`${request.method()} ${url.origin}${url.pathname}`);
      await route.fulfill({
        status: 599,
        contentType: 'application/json',
        body: '{"code":"unexpected_request"}',
      });
      return;
    }
    await route.fulfill({
      status: fixture.status,
      contentType: 'application/json',
      body: fixture.body === undefined ? undefined : JSON.stringify(fixture.body),
    });
  });

  return {
    requests,
    resourceUrls,
    assertNoFailures: () => {
      if (unexpected.length || pageErrors.length)
        throw new Error(
          `Browser boundary failures:\n${[...unexpected.map((request) => `Unexpected HTTP request: ${request}`), ...pageErrors.map((error) => `Page error: ${error}`)].join('\n')}`,
        );
    },
  };
};
