import type { TFunction } from 'i18next';

import { API_ERROR_CODES, type ApiComponents, ApiError } from '@/shared/api';
import type { TranslationKey } from '@/shared/i18n/TranslationKey';

// Every code the contract declares. A code added to the contract without a key here fails
// pnpm typecheck, so it never reaches the user as raw text.
const CONTRACT_ERROR_CODE_KEYS = {
  invalid_request: 'errors.invalidRequest',
  unauthenticated: 'errors.unauthenticated',
  permission_denied: 'errors.permissionDenied',
  not_found: 'errors.notFound',
  rate_limited: 'errors.rateLimited',
  human_verification_failed: 'errors.humanVerificationFailed',
  invalid_credentials: 'errors.invalidCredentials',
  user_deactivated: 'errors.userDeactivated',
  registration_closed: 'errors.registrationClosed',
  email_taken: 'errors.emailTaken',
  invalid_reset_code: 'errors.invalidResetCode',
  oauth_not_configured: 'errors.oauthNotConfigured',
  oauth_failed: 'errors.oauthFailed',
  pending_sign_in_not_usable: 'errors.pendingSignInNotUsable',
  sign_in_code_not_sent: 'errors.signInCodeNotSent',
  too_many_sign_in_codes: 'errors.tooManySignInCodes',
  sign_in_code_resent_too_soon: 'errors.signInCodeResentTooSoon',
  sign_in_code_incorrect: 'errors.signInCodeIncorrect',
  email_not_configured: 'errors.emailNotConfigured',
  admin_email_missing: 'errors.adminEmailMissing',
  already_member: 'errors.alreadyMember',
  cannot_change_own_role: 'errors.cannotChangeOwnRole',
  owner_requires_transfer: 'errors.ownerRequiresTransfer',
  workspace_not_empty: 'errors.workspaceNotEmpty',
  in_use: 'errors.inUse',
  cleanup_pending: 'errors.cleanupPending',
  database_connection_failed: 'errors.databaseConnectionFailed',
  backup_not_completed: 'errors.backupNotCompleted',
  already_running: 'errors.alreadyRunning',
  no_extendable_chain: 'errors.noExtendableChain',
  no_restorable_backup: 'errors.noRestorableBackup',
  target_time_unreachable: 'errors.targetTimeUnreachable',
  not_running: 'errors.notRunning',
  instance_unhealthy: 'errors.instanceUnhealthy',
  insufficient_disk_space: 'errors.insufficientDiskSpace',
  internal: 'errors.internal',
} as const satisfies Record<ApiComponents['schemas']['ProblemCode'], TranslationKey>;

const API_ERROR_CODE_KEYS: Record<string, TranslationKey> = {
  ...CONTRACT_ERROR_CODE_KEYS,
  [API_ERROR_CODES.networkUnreachable]: 'errors.networkUnreachable',
  [API_ERROR_CODES.requestFailed]: 'errors.requestFailed',
};

// The one way an error reaches the screen. The message of anything that is not an ApiError comes
// from the browser or a library, never from the backend, so it is not shown.
export const translateApiError = (error: unknown, t: TFunction): string => {
  if (!(error instanceof ApiError)) {
    return t('errors.unknown');
  }

  const codeKey =
    error.code && Object.prototype.hasOwnProperty.call(API_ERROR_CODE_KEYS, error.code)
      ? API_ERROR_CODE_KEYS[error.code]
      : undefined;
  if (!codeKey) return t('errors.unknown');

  const detail =
    error.code === 'database_connection_failed' || error.code === 'insufficient_disk_space'
      ? error.message
      : '';
  return t(codeKey, { status: error.status ?? '', detail });
};
