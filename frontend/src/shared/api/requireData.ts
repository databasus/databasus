import { API_ERROR_CODES, ApiError } from './ApiError';
import type { components } from './schema.gen';

type Problem = components['schemas']['Problem'];

interface FetchResult<Data> {
  data?: Data;
  error?: unknown;
  response: Response;
}

const isProblem = (body: unknown): body is Problem => {
  if (typeof body !== 'object' || body === null) {
    return false;
  }

  const candidate = body as Record<string, unknown>;

  return typeof candidate.code === 'string' && typeof candidate.detail === 'string';
};

// Turns the result of a contract operation into its data, or throws the ApiError that
// translateApiError shows by code. An answer without a Problem body becomes request_failed.
export const requireData = <Data>({ data, error, response }: FetchResult<Data>): Data => {
  if (response.ok && data !== undefined) {
    return data;
  }

  if (isProblem(error)) {
    throw new ApiError({ message: error.detail, code: error.code, status: response.status });
  }

  throw new ApiError({ code: API_ERROR_CODES.requestFailed, status: response.status });
};
