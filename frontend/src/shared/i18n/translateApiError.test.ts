import type { TFunction } from 'i18next';
import { beforeAll, describe, expect, it } from 'vitest';

import { API_ERROR_CODES, ApiError } from '../api';
import { createI18n } from './createI18n';
import { translateApiError } from './translateApiError';

let t: TFunction;

beforeAll(() => {
  t = createI18n('ru').t;
});

describe('translateApiError', () => {
  it('translates a known code', () => {
    const error = new ApiError({ code: API_ERROR_CODES.networkUnreachable });

    expect(translateApiError(error, t)).toBe(t('errors.networkUnreachable'));
  });

  it('translates a code of the contract', () => {
    const error = new ApiError({ code: 'internal', status: 500 });

    expect(translateApiError(error, t)).toBe(t('errors.internal'));
    expect(translateApiError(error, t)).not.toBe('internal');
  });

  it('includes the status for request_failed', () => {
    const error = new ApiError({ code: API_ERROR_CODES.requestFailed, status: 502 });

    expect(translateApiError(error, t)).toBe(t('errors.requestFailed', { status: 502 }));
    expect(translateApiError(error, t)).toContain('502');
  });

  it('tells the two sign-in code refusals that share a status apart', () => {
    const resentTooSoon = new ApiError({ code: 'sign_in_code_resent_too_soon', status: 429 });
    const hourlyCapReached = new ApiError({ code: 'too_many_sign_in_codes', status: 429 });

    expect(translateApiError(resentTooSoon, t)).toBe(t('errors.signInCodeResentTooSoon'));
    expect(translateApiError(hourlyCapReached, t)).toBe(t('errors.tooManySignInCodes'));
  });

  it('shows the backend message for an unknown code', () => {
    const error = new ApiError({
      message: 'storage quota exceeded',
      code: 'quota_exceeded',
      status: 400,
    });

    expect(translateApiError(error, t)).toBe('Storage quota exceeded');
  });

  it('shows the general message for an unknown code without a message', () => {
    const error = new ApiError({ code: 'quota_exceeded', status: 400 });

    expect(translateApiError(error, t)).toBe(t('errors.unknown'));
  });

  it('shows the backend message when there is no code', () => {
    const error = new ApiError({ message: 'database not found', status: 404 });

    expect(translateApiError(error, t)).toBe('Database not found');
  });

  it('never shows the message of an error that is not an ApiError', () => {
    const error = new TypeError('Failed to fetch');

    expect(translateApiError(error, t)).toBe(t('errors.unknown'));
  });
});
