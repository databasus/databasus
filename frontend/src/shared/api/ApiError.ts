// Codes the frontend assigns to failures the backend does not describe itself. They use the same
// snake_case form as the backend's codes, so translateApiError resolves both the same way.
export const API_ERROR_CODES = {
  // The request never got an answer: the server is down or the network is unreachable
  networkUnreachable: 'network_unreachable',
  // A response lacks required data, readable JSON or a usable Problem
  requestFailed: 'request_failed',
} as const;

interface ApiErrorFields {
  // Only the backend's own message, which may be empty
  message?: string;
  // The backend's machine-readable code, or one of API_ERROR_CODES
  code?: string;
  status?: number;
}

// Describes a failed API request. Show it to users through translateApiError.
export class ApiError extends Error {
  readonly code?: string;
  readonly status?: number;

  constructor({ message = '', code, status }: ApiErrorFields) {
    super(message);
    this.name = 'ApiError';
    this.code = code;
    this.status = status;
  }
}
