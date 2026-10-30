import { accessTokenHelper } from '.';
import { API_ERROR_CODES, ApiError } from './ApiError';
import { RateLimiter } from './RateLimiter';
import RequestOptions from './RequestOptions';

const REPEAT_TRIES_COUNT = 30;
const REPEAT_INTERVAL_MS = 3_000;
const rateLimiter = new RateLimiter(30, 1_000);

const clearTokenAndReloadIfUnauthorized = (response: Response) => {
  if (response.status === 401) {
    accessTokenHelper?.clearStoredSession();
    window.location.reload();
  }
};

const throwIfErrorResponse = async (response: Response) => {
  if (response.status === 502 || response.status === 504) {
    throw new ApiError({ code: API_ERROR_CODES.requestFailed, status: response.status });
  }

  if (response.status >= 400 && response.status <= 600) {
    let body: { message?: string; error?: string; code?: string } | null;

    try {
      body = await response.json();
    } catch {
      throw new ApiError({ code: API_ERROR_CODES.requestFailed, status: response.status });
    }

    throw new ApiError({
      message: body?.message || body?.error,
      code: body?.code,
      status: response.status,
    });
  }
};

// Only the rejection of fetch itself means the server was never reached. ApiErrors thrown for an
// answered request must not be mistaken for it.
const fetchOrThrowNetworkError = async (url: string, optionsWrapper: RequestOptions) => {
  try {
    return await fetch(url, optionsWrapper.toRequestInit());
  } catch {
    throw new ApiError({ code: API_ERROR_CODES.networkUnreachable });
  }
};

const makeRequest = async (
  url: string,
  optionsWrapper: RequestOptions,
  currentTry = 0,
): Promise<Response> => {
  await rateLimiter.acquire();

  try {
    const response = await fetchOrThrowNetworkError(url, optionsWrapper);
    clearTokenAndReloadIfUnauthorized(response);
    await throwIfErrorResponse(response);
    return response;
  } catch (e) {
    if (currentTry < REPEAT_TRIES_COUNT) {
      await new Promise((resolve) => setTimeout(resolve, REPEAT_INTERVAL_MS));
      return makeRequest(url, optionsWrapper, currentTry + 1);
    }

    throw e;
  }
};

export const apiHelper = {
  fetchPostJson: async <T>(
    url: string,
    requestOptions?: RequestOptions,
    isRetryOnError = false,
  ): Promise<T> => {
    const optionsWrapper = (requestOptions ?? new RequestOptions())
      .setMethod('POST')
      .addHeader('Content-Type', 'application/json')
      .addHeader('Access-Control-Allow-Methods', 'POST')
      .addHeader('Accept', 'application/json')
      .addHeader('Authorization', accessTokenHelper.getAccessToken());

    const response = await makeRequest(
      url,
      optionsWrapper,
      isRetryOnError ? 0 : REPEAT_TRIES_COUNT,
    );

    return response.json();
  },

  fetchPostRaw: async (
    url: string,
    requestOptions?: RequestOptions,
    isRetryOnError = false,
  ): Promise<string> => {
    const optionsWrapper = (requestOptions ?? new RequestOptions())
      .setMethod('POST')
      .addHeader('Content-Type', 'application/json')
      .addHeader('Access-Control-Allow-Methods', 'POST')
      .addHeader('Accept', 'application/json')
      .addHeader('Authorization', accessTokenHelper.getAccessToken());

    const response = await makeRequest(
      url,
      optionsWrapper,
      isRetryOnError ? 0 : REPEAT_TRIES_COUNT,
    );

    return response.text();
  },

  fetchPostBlob: async (
    url: string,
    requestOptions?: RequestOptions,
    isRetryOnError = false,
  ): Promise<Blob> => {
    const optionsWrapper = (requestOptions ?? new RequestOptions())
      .setMethod('POST')
      .addHeader('Content-Type', 'application/json')
      .addHeader('Access-Control-Allow-Methods', 'POST')
      .addHeader('Authorization', accessTokenHelper.getAccessToken());

    const response = await makeRequest(
      url,
      optionsWrapper,
      isRetryOnError ? 0 : REPEAT_TRIES_COUNT,
    );

    return response.blob();
  },

  fetchGetJson: async <T>(
    url: string,
    requestOptions?: RequestOptions,
    isRetryOnError = false,
  ): Promise<T> => {
    const optionsWrapper = (requestOptions ?? new RequestOptions())
      .addHeader('Content-Type', 'application/json')
      .addHeader('Access-Control-Allow-Methods', 'GET')
      .addHeader('Accept', 'application/json')
      .addHeader('Authorization', accessTokenHelper.getAccessToken());

    const response = await makeRequest(
      url,
      optionsWrapper,
      isRetryOnError ? 0 : REPEAT_TRIES_COUNT,
    );

    return response.json();
  },

  fetchGetRaw: async (
    url: string,
    requestOptions?: RequestOptions,
    isRetryOnError = false,
  ): Promise<string> => {
    const optionsWrapper = (requestOptions ?? new RequestOptions())
      .addHeader('Content-Type', 'application/json')
      .addHeader('Access-Control-Allow-Methods', 'GET')
      .addHeader('Accept', 'application/json')
      .addHeader('Authorization', accessTokenHelper.getAccessToken());

    const response = await makeRequest(
      url,
      optionsWrapper,
      isRetryOnError ? 0 : REPEAT_TRIES_COUNT,
    );

    return response.text();
  },

  fetchGetBlob: async (
    url: string,
    requestOptions?: RequestOptions,
    isRetryOnError = false,
  ): Promise<Blob> => {
    const optionsWrapper = (requestOptions ?? new RequestOptions())
      .addHeader('Access-Control-Allow-Methods', 'GET')
      .addHeader('Authorization', accessTokenHelper.getAccessToken());

    const response = await makeRequest(
      url,
      optionsWrapper,
      isRetryOnError ? 0 : REPEAT_TRIES_COUNT,
    );

    return response.blob();
  },

  fetchGetBlobWithHeaders: async (
    url: string,
    requestOptions?: RequestOptions,
    isRetryOnError = false,
  ): Promise<{ blob: Blob; headers: Headers }> => {
    const optionsWrapper = (requestOptions ?? new RequestOptions())
      .addHeader('Access-Control-Allow-Methods', 'GET')
      .addHeader('Authorization', accessTokenHelper.getAccessToken());

    const response = await makeRequest(
      url,
      optionsWrapper,
      isRetryOnError ? 0 : REPEAT_TRIES_COUNT,
    );

    const blob = await response.blob();
    return { blob, headers: response.headers };
  },

  fetchPutJson: async <T>(
    url: string,
    requestOptions?: RequestOptions,
    isRetryOnError = false,
  ): Promise<T> => {
    const optionsWrapper = (requestOptions ?? new RequestOptions())
      .setMethod('PUT')
      .addHeader('Content-Type', 'application/json')
      .addHeader('Access-Control-Allow-Methods', 'PUT')
      .addHeader('Accept', 'application/json')
      .addHeader('Authorization', accessTokenHelper.getAccessToken());

    const response = await makeRequest(
      url,
      optionsWrapper,
      isRetryOnError ? 0 : REPEAT_TRIES_COUNT,
    );

    return response.json();
  },

  fetchDeleteJson: async <T>(
    url: string,
    requestOptions?: RequestOptions,
    isRetryOnError = false,
  ): Promise<T> => {
    const optionsWrapper = (requestOptions ?? new RequestOptions())
      .setMethod('DELETE')
      .addHeader('Access-Control-Allow-Methods', 'DELETE')
      .addHeader('Accept', 'application/json')
      .addHeader('Authorization', accessTokenHelper.getAccessToken());

    const response = await makeRequest(
      url,
      optionsWrapper,
      isRetryOnError ? 0 : REPEAT_TRIES_COUNT,
    );

    if (response.status === 204) {
      return undefined as T;
    }

    const text = await response.text();
    return (text ? JSON.parse(text) : undefined) as T;
  },

  fetchDeleteRaw: async (
    url: string,
    requestOptions?: RequestOptions,
    isRetryOnError = false,
  ): Promise<string> => {
    const optionsWrapper = (requestOptions ?? new RequestOptions())
      .setMethod('DELETE')
      .addHeader('Access-Control-Allow-Methods', 'DELETE')
      .addHeader('Accept', 'application/json')
      .addHeader('Authorization', accessTokenHelper.getAccessToken());

    const response = await makeRequest(
      url,
      optionsWrapper,
      isRetryOnError ? 0 : REPEAT_TRIES_COUNT,
    );

    return response.text();
  },
};
