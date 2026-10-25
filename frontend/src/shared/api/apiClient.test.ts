import { HttpResponse, http } from 'msw';
import { beforeEach, describe, expect, it, vi } from 'vitest';

import { setUpMockApi } from '../lib/test/mockApi';
import { accessTokenHelper } from './accessTokenHelper';
import { API_BASE_PATH, getApiClient } from './apiClient';

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
});
