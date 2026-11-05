import { describe, expect, it } from 'vitest';

import { API_ERROR_CODES, ApiError } from '../../../shared/api';
import { mockApi, setUpMockApi } from '../../../shared/lib/test/mockApi';
import { systemApi } from './systemApi';

const mockServer = setUpMockApi();

const catchApiError = async (request: Promise<unknown>): Promise<ApiError> => {
  try {
    await request;
  } catch (e) {
    if (e instanceof ApiError) {
      return e;
    }
    throw e;
  }
  throw new Error('request did not fail');
};

describe('systemApi.getVersion', () => {
  it('returns the version when the server answers 200', async () => {
    mockServer.use(mockApi.get('/system/version', 200, { version: '3.26.0' }));

    const versionResponse = await systemApi.getVersion();

    expect(versionResponse.version).toBe('3.26.0');
  });

  it('throws an ApiError with the Problem code when the server answers 500', async () => {
    mockServer.use(
      mockApi.get('/system/version', 500, {
        type: 'urn:databasus:problem:internal',
        title: 'Internal error',
        status: 500,
        detail: 'an internal error occurred',
        code: 'internal',
      }),
    );

    const apiError = await catchApiError(systemApi.getVersion());

    expect(apiError.code).toBe('internal');
    expect(apiError.status).toBe(500);
  });

  it('throws request_failed when the answer has no body', async () => {
    mockServer.use(mockApi.getWithoutBody('/system/version', 502));

    const apiError = await catchApiError(systemApi.getVersion());

    expect(apiError.code).toBe(API_ERROR_CODES.requestFailed);
    expect(apiError.status).toBe(502);
  });

  it('rejects at type level a mock without the version field', () => {
    // @ts-expect-error the contract requires version in the body of 200
    mockApi.get('/system/version', 200, {});
  });
});
