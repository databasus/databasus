import { fileURLToPath } from 'node:url';
import { describe, expect, it } from 'vitest';

import { API_BASE_PATH } from '../../api';
import { buildProblem, mockApi, setUpMockApi } from './mockApi';
import { runIsolatedScenario } from './runIsolatedScenario';

const server = setUpMockApi();

describe('mockApi', () => {
  it('returns typed JSON, declared errors and empty success with explicit handlers', async () => {
    server.use(
      mockApi.get('/workspaces', 200, { items: [] }),
      mockApi.delete('/workspaces/{workspaceId}', 204, undefined),
    );

    expect(await (await fetch(`http://localhost${API_BASE_PATH}/workspaces`)).json()).toEqual({
      items: [],
    });
    expect(
      await (
        await fetch(`http://localhost${API_BASE_PATH}/workspaces/id`, { method: 'DELETE' })
      ).text(),
    ).toBe('');

    server.use(mockApi.get('/workspaces', 403, buildProblem('permission_denied', 403)));
    expect((await fetch(`http://localhost${API_BASE_PATH}/workspaces`)).status).toBe(403);
  });

  it('exposes network failures, malformed bodies and controlled response release', async () => {
    server.use(mockApi.network('get', '/workspaces'));
    await expect(fetch('http://localhost/api/v1/workspaces')).rejects.toThrow();

    server.use(mockApi.malformed({ method: 'get', path: '/workspaces', status: 200, body: '{' }));
    await expect((await fetch('http://localhost/api/v1/workspaces')).json()).rejects.toThrow();

    const response = mockApi.delayed({
      method: 'get',
      path: '/workspaces',
      status: 200,
      body: { items: [] },
    });
    server.use(response.handler);

    const request = fetch('http://localhost/api/v1/workspaces');
    response.release();
    expect(await (await request).json()).toEqual({ items: [] });
    expect(server.getRequests()).toHaveLength(3);
  });

  it('rejects invalid contract shapes at typecheck time', () => {
    // @ts-expect-error workspace lists require items
    mockApi.get('/workspaces', 200, { value: [] });
    // @ts-expect-error successful deletion has no JSON body
    mockApi.delete('/workspaces/{workspaceId}', 204, { items: [] });
    // @ts-expect-error undeclared statuses require a Problem
    mockApi.get('/workspaces', 599, { items: [] });
    // @ts-expect-error this operation has no POST method
    mockApi.post('/workspaces/{workspaceId}', 200, {});
  });

  it('fails independently when the caller catches an undeclared request', async () => {
    const child = await runIsolatedScenario({
      scenario: `it('catches fetch', async () => { await fetch('http://localhost/api/v1/undeclared').catch(() => undefined); });`,
    });
    expect(child.code).not.toBe(0);
    expect(child.output).toContain('Unexpected HTTP requests:');
    expect(child.output).toContain('GET http://localhost/api/v1/undeclared');
  });

  it.each([
    "http.all('*/api/v1/workspaces', () => new Promise(() => {}))",
    "http.get('*/api/v1/*', () => new Promise(() => {}))",
    "http.get('*/api/v1/:path+', () => new Promise(() => {}))",
  ])('rejects a blanket pending declaration with %s', async (handler) => {
    const child = await runIsolatedScenario({
      scenario: `it('declares handler', () => { server.use(${handler}); });`,
    });
    expect(child.code).not.toBe(0);
    expect(child.output).toContain('HTTP mocks require');
  });

  it('fails when an origin wildcard would consume an extra pathname prefix', async () => {
    const child = await runIsolatedScenario({
      scenario: `it('catches an undeclared prefixed path', async () => {
      server.use(mockApi.get('/workspaces', 200, { items: [] }));
      await fetch('http://localhost/unexpected/api/v1/workspaces').catch(() => undefined);
    });`,
    });
    expect(child.code).not.toBe(0);
    expect(child.output).toContain('Unexpected HTTP requests:');
    expect(child.output).toContain('GET http://localhost/unexpected/api/v1/workspaces');
  });

  it('clears state even when a raw handler does not settle before the cleanup deadline', async () => {
    const child = await runIsolatedScenario({
      scenario: `
it('leaves an unowned raw handler pending', async () => {
  server.use(http.get('*/api/v1/workspaces', () => new Promise(() => {})));
  const started = new Promise(resolve => server.events.on('request:start', resolve));
  void fetch('http://localhost/api/v1/workspaces').catch(() => undefined);
  await started;
});
it('starts clean after cleanup rejects the unowned handler', async () => {
  expect(server.listHandlers()).toHaveLength(0);
  expect(server.getRequests()).toHaveLength(0);
  server.use(mockApi.get('/workspaces', 200, { items: [] }));
  expect(await (await fetch('http://localhost/api/v1/workspaces')).json()).toEqual({ items: [] });
});`,
    });

    expect(child.code).not.toBe(0);
    expect(child.output).toContain(
      'HTTP cleanup timed out: pending handlers must use createPendingApiResponse',
    );
    expect(child.output).toContain('1 failed | 1 passed');
    expect(child.output).not.toContain('Hook timed out');
  });

  it('cleans handlers, pending work and requests after a failing request scenario', async () => {
    const child = await runIsolatedScenario({
      scenario: `
import { createPendingApiResponse } from ${JSON.stringify(fileURLToPath(new URL('./pendingApiResponse.ts', import.meta.url)))};
it('fails before the pending response is released', async () => {
  const pending = createPendingApiResponse();
  server.use(http.get('*/api/v1/workspaces', () => pending.response));
  const started = new Promise(resolve => server.events.on('request:start', resolve));
  void fetch('http://localhost/api/v1/workspaces').catch(() => undefined);
  await started;
  throw new Error('deliberate pending fixture failure');
});
it('starts clean after the failed request scenario', async () => {
  expect(server.listHandlers()).toHaveLength(0);
  expect(server.getRequests()).toHaveLength(0);
  server.use(mockApi.get('/workspaces', 200, { items: [] }));
  expect(await (await fetch('http://localhost/api/v1/workspaces')).json()).toEqual({ items: [] });
});`,
    });

    expect(child.code).not.toBe(0);
    expect(child.output).toContain('deliberate pending fixture failure');
    expect(child.output).toContain('1 failed | 1 passed');
    expect(child.output).not.toContain('Hook timed out');
  });

  it('isolates handlers, request records and pending work between consecutive tests', async () => {
    const child = await runIsolatedScenario({
      scenario: `
it('starts a controlled pending request', async () => {
  server.use(mockApi.pending('get', '/workspaces').handler);
  const started = new Promise(resolve => server.events.on('request:start', resolve));
  void fetch('http://localhost/api/v1/workspaces').catch(() => undefined);
  await started;
  expect(server.getRequests()).toHaveLength(1);
});
it('starts with clean state and completes a new request', async () => {
  expect(server.listHandlers()).toHaveLength(0);
  expect(server.getRequests()).toHaveLength(0);
  server.use(mockApi.get('/workspaces', 200, { items: [] }));
  expect(await (await fetch('http://localhost/api/v1/workspaces')).json()).toEqual({ items: [] });
});`,
    });
    expect(child.output).toContain('2 passed');
    expect(child.code).toBe(0);
  });
});
