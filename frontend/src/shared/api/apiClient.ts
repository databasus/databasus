import createClient, { type Middleware } from 'openapi-fetch';

import { API_ERROR_CODES, ApiError } from '@/shared/api/ApiError';
import type { ReadOptions } from '@/shared/api/ReadOptions';
import type { paths } from '@/shared/api/schema.gen';
import { getSessionSnapshot, invalidateSession, type SessionSnapshot } from '@/shared/auth';
import { getApplicationServer } from '@/shared/config';

export const API_BASE_PATH = '/api/v1';

const authorizedRequests = new WeakMap<Request, SessionSnapshot>();

const apiMiddleware: Middleware = {
  onRequest({ request }) {
    const session = getSessionSnapshot();
    const token = session.token;
    if (token) {
      request.headers.set('Authorization', `Bearer ${token}`);
      authorizedRequests.set(request, session);
    }

    return request;
  },

  async onResponse({ request, response, options }) {
    const session = authorizedRequests.get(request);
    authorizedRequests.delete(request);
    if (response.status === 401 && session) invalidateSession(session);

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
export const getApiClient = (options: ReadOptions = {}) => {
  const client = createClient<paths>({
    baseUrl: `${getApplicationServer()}${API_BASE_PATH}`,
    signal: options.signal,
  });
  client.use(apiMiddleware);

  return client;
};
