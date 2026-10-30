import createClient, { type Middleware } from 'openapi-fetch';

import { getApplicationServer } from '../../constants';
import { accessTokenHelper } from './accessTokenHelper';
import { API_ERROR_CODES, ApiError } from './ApiError';
import type { paths } from './schema.gen';

export const API_BASE_PATH = '/api/v1';

// A 401 to a request that carried the session means the session ended, so the token is cleared and
// the page reloads into the sign-in screen. A 401 to a request without one is a refused sign-in,
// which the screen shows itself.
const apiMiddleware: Middleware = {
  onRequest({ request }) {
    const token = accessTokenHelper.getAccessToken();
    if (token) {
      request.headers.set('Authorization', `Bearer ${token}`);
    }

    return request;
  },

  async onResponse({ request, response, options }) {
    if (response.status === 401 && request.headers.has('Authorization')) {
      accessTokenHelper.clearStoredSession();
      window.location.reload();
    }

    if (
      response.status !== 204 &&
      request.method !== 'HEAD' &&
      (!response.ok || options.parseAs === 'json')
    ) {
      try {
        const body = await response.clone().text();
        if (response.ok && body) JSON.parse(body);
      } catch (error) {
        if (request.signal.aborted) throw error;
        throw new ApiError({ code: API_ERROR_CODES.requestFailed, status: response.status });
      }
    }
    return response;
  },

  onError({ request, error }) {
    if (request.signal.aborted) throw error;
    return new ApiError({ code: API_ERROR_CODES.networkUnreachable });
  },
};

// A client is created per call. getApplicationServer reads window.location, and a client created at
// import time would keep the fetch it found then, before a test's network mock replaces it.
export const getApiClient = () => {
  const client = createClient<paths>({ baseUrl: `${getApplicationServer()}${API_BASE_PATH}` });
  client.use(apiMiddleware);

  return client;
};
