import createClient, { type Middleware } from 'openapi-fetch';

import { getApplicationServer } from '../../constants';
import { accessTokenHelper } from './accessTokenHelper';
import type { paths } from './schema.gen';

export const API_BASE_PATH = '/api/v1';

// A 401 to a request that carried the session means the session ended, so the token is cleared and
// the page reloads into the sign-in screen. A 401 to a request without one is a refused sign-in,
// which the screen shows itself.
const sessionMiddleware: Middleware = {
  onRequest({ request }) {
    const token = accessTokenHelper.getAccessToken();
    if (token) {
      request.headers.set('Authorization', `Bearer ${token}`);
    }

    return request;
  },

  onResponse({ request, response }) {
    if (response.status === 401 && request.headers.has('Authorization')) {
      accessTokenHelper.clearStoredSession();
      window.location.reload();
    }

    return response;
  },
};

// A client is created per call. getApplicationServer reads window.location, and a client created at
// import time would keep the fetch it found then, before a test's network mock replaces it.
export const getApiClient = () => {
  const client = createClient<paths>({ baseUrl: `${getApplicationServer()}${API_BASE_PATH}` });
  client.use(sessionMiddleware);

  return client;
};
