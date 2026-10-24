import createClient from 'openapi-fetch';

import { getApplicationServer } from '../../constants';
import type { paths } from './schema.gen';

export const API_BASE_PATH = '/api/v1';

// A client is created per call. getApplicationServer reads window.location, and a client created at
// import time would keep the fetch it found then, before a test's network mock replaces it.
export const getApiClient = () =>
  createClient<paths>({ baseUrl: `${getApplicationServer()}${API_BASE_PATH}` });
