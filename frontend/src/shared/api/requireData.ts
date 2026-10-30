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

  return (
    typeof candidate.type === 'string' &&
    typeof candidate.title === 'string' &&
    typeof candidate.status === 'number' &&
    Number.isInteger(candidate.status) &&
    typeof candidate.code === 'string' &&
    typeof candidate.detail === 'string'
  );
};

const toApiError = ({ error, response }: FetchResult<unknown>): ApiError => {
  if (isProblem(error)) {
    return new ApiError({ message: error.detail, code: error.code, status: response.status });
  }

  return new ApiError({ code: API_ERROR_CODES.requestFailed, status: response.status });
};

// Turns the result of a contract operation into its data, or throws the ApiError that
// translateApiError shows by code. An answer without a Problem body becomes request_failed.
export const requireData = <Data>(result: FetchResult<Data>): Data => {
  if (result.response.ok && result.data !== undefined) {
    return result.data;
  }

  throw toApiError(result);
};

// The same for an operation that answers 204 without a body.
export const requireSuccess = (result: FetchResult<unknown>): void => {
  if (!result.response.ok) {
    throw toApiError(result);
  }
};
