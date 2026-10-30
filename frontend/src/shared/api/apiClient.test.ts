import { HttpResponse, http } from 'msw';
import { beforeEach, describe, expect, it, vi } from 'vitest';

import { buildProblem, mockApi, setUpMockApi } from '../lib/test/mockApi';
import { accessTokenHelper } from './accessTokenHelper';
import { API_BASE_PATH, getApiClient } from './apiClient';
import { API_ERROR_CODES } from './ApiError';
import { requireData, requireSuccess } from './requireData';

const mockServer = setUpMockApi();

const storedItems = new Map<string, string>();

beforeEach(() => {
  storedItems.clear();
  vi.stubGlobal('localStorage', {
    getItem: (key: string) => storedItems.get(key) ?? null,
    setItem: (key: string, value: string) => storedItems.set(key, value),
    removeItem: (key: string) => storedItems.delete(key),
  });
  window.location.reload = vi.fn();
});

describe('getApiClient', () => {
  it('normalizes a network rejection without exposing a URL or replaying a mutation', async () => {
    mockServer.use(mockApi.network('post', '/workspaces'));
    await expect(
      getApiClient().POST('/workspaces', { body: { name: 'Research' } }),
    ).rejects.toMatchObject({
      name: 'ApiError',
      code: API_ERROR_CODES.networkUnreachable,
      message: '',
      status: undefined,
    });
    expect(mockServer.getRequests()).toHaveLength(1);
  });

  it('normalizes unreadable success JSON with its response status', async () => {
    mockServer.use(
      mockApi.malformed({ method: 'get', path: '/system/version', status: 200, body: '{' }),
    );
    await expect(getApiClient().GET('/system/version')).rejects.toMatchObject({
      name: 'ApiError',
      code: API_ERROR_CODES.requestFailed,
      status: 200,
      message: '',
    });
  });

  it.each([
    [502, 'Bad Gateway'],
    [500, ''],
    [422, '{'],
    [400, '{}'],
    [400, '{"code":"internal","detail":"private diagnostic"}'],
  ])('normalizes an invalid error body with status %s and body %s', async (status, body) => {
    mockServer.use(
      mockApi.malformed({ method: 'get', path: '/system/version', status: status, body: body }),
    );
    await expect(getApiClient().GET('/system/version').then(requireData)).rejects.toMatchObject({
      name: 'ApiError',
      code: API_ERROR_CODES.requestFailed,
      status,
      message: '',
    });
  });

  it('retains a valid Problem and rejects missing required success data', async () => {
    mockServer.use(
      mockApi.get('/system/version', 500, {
        ...buildProblem('internal', 500),
        detail: 'diagnostic',
      }),
    );
    await expect(getApiClient().GET('/system/version').then(requireData)).rejects.toMatchObject({
      code: 'internal',
      message: 'diagnostic',
      status: 500,
    });
    mockServer.use(mockApi.getWithoutBody('/system/version', 200));
    await expect(getApiClient().GET('/system/version').then(requireData)).rejects.toMatchObject({
      code: API_ERROR_CODES.requestFailed,
      status: 200,
    });
  });

  it('accepts an empty success without parsing a JSON body', async () => {
    mockServer.use(mockApi.delete('/workspaces/{workspaceId}', 204, undefined));
    await expect(
      getApiClient()
        .DELETE('/workspaces/{workspaceId}', { params: { path: { workspaceId: 'id' } } })
        .then(requireSuccess),
    ).resolves.toBeUndefined();
  });

  it('preserves intentional cancellation without converting it to a network error', async () => {
    const controller = new AbortController();
    controller.abort();
    await expect(
      getApiClient().GET('/system/version', { signal: controller.signal }),
    ).rejects.toMatchObject({ name: 'AbortError' });
  });

  it('sends the stored session token', async () => {
    accessTokenHelper.saveAccessToken('session-token');
    let sentAuthorization: string | null = null;
    mockServer.use(
      http.get(`*${API_BASE_PATH}/system/version`, ({ request }) => {
        sentAuthorization = request.headers.get('Authorization');
        return HttpResponse.json({ version: '3.26.0' });
      }),
    );

    await getApiClient().GET('/system/version');

    expect(sentAuthorization).toBe('Bearer session-token');
  });

  it('clears the token and reloads the page when the session is refused', async () => {
    accessTokenHelper.saveAccessToken('ended-session-token');
    mockServer.use(
      http.get(`*${API_BASE_PATH}/system/version`, () =>
        HttpResponse.json({ code: 'unauthenticated' }, { status: 401 }),
      ),
    );

    await getApiClient().GET('/system/version');

    expect(accessTokenHelper.getAccessToken()).toBeUndefined();
    expect(window.location.reload).toHaveBeenCalledOnce();
  });

  it('keeps the page when a request without a session is refused', async () => {
    mockServer.use(
      http.post(`*${API_BASE_PATH}/sessions`, () =>
        HttpResponse.json({ code: 'invalid_credentials' }, { status: 401 }),
      ),
    );

    await getApiClient().POST('/sessions', { body: { email: 'a@example.com', password: 'wrong' } });

    expect(window.location.reload).not.toHaveBeenCalled();
  });
  it('starts no timers, requests or queues when the API public entry point is imported', async () => {
    vi.useFakeTimers();
    vi.resetModules();
    try {
      await import('./index');
      expect(vi.getTimerCount()).toBe(0);
      expect(mockServer.getRequests()).toHaveLength(0);
    } finally {
      vi.clearAllTimers();
      vi.useRealTimers();
    }
  });
});
