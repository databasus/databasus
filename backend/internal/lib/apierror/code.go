package apierror

import (
	"net/http"
)

// Code lists the ProblemCode values of the user and agent contracts.
type Code string

const (
	CodeInvalidRequest           Code = "invalid_request"
	CodeUnauthenticated          Code = "unauthenticated"
	CodePermissionDenied         Code = "permission_denied"
	CodeNotFound                 Code = "not_found"
	CodeRateLimited              Code = "rate_limited"
	CodeHumanVerificationFailed  Code = "human_verification_failed"
	CodeInvalidCredentials       Code = "invalid_credentials"
	CodeUserDeactivated          Code = "user_deactivated"
	CodeRegistrationClosed       Code = "registration_closed"
	CodeEmailTaken               Code = "email_taken"
	CodeInvalidResetCode         Code = "invalid_reset_code"
	CodeOAuthNotConfigured       Code = "oauth_not_configured"
	CodeOAuthFailed              Code = "oauth_failed"
	CodePendingSignInNotUsable   Code = "pending_sign_in_not_usable"
	CodeSignInCodeNotSent        Code = "sign_in_code_not_sent"
	CodeTooManySignInCodes       Code = "too_many_sign_in_codes"
	CodeSignInCodeResentTooSoon  Code = "sign_in_code_resent_too_soon"
	CodeSignInCodeIncorrect      Code = "sign_in_code_incorrect"
	CodeEmailNotConfigured       Code = "email_not_configured"
	CodeAdminEmailMissing        Code = "admin_email_missing"
	CodeAlreadyMember            Code = "already_member"
	CodeCannotChangeOwnRole      Code = "cannot_change_own_role"
	CodeOwnerRequiresTransfer    Code = "owner_requires_transfer"
	CodeWorkspaceNotEmpty        Code = "workspace_not_empty"
	CodeInUse                    Code = "in_use"
	CodeCleanupPending           Code = "cleanup_pending"
	CodeDatabaseConnectionFailed Code = "database_connection_failed"
	CodeAlreadyRunning           Code = "already_running"
	CodeNoExtendableChain        Code = "no_extendable_chain"
	CodeNoRestorableBackup       Code = "no_restorable_backup"
	CodeTargetTimeUnreachable    Code = "target_time_unreachable"
	CodeBackupNotCompleted       Code = "backup_not_completed"
	CodeNotRunning               Code = "not_running"
	CodeInsufficientDiskSpace    Code = "insufficient_disk_space"
	CodeVerificationGone         Code = "verification_gone"
	CodeInstanceUnhealthy        Code = "instance_unhealthy"
	CodeInternal                 Code = "internal"
)

func describeCode(code Code) (int, string) {
	switch code {
	case CodeInvalidRequest:
		return http.StatusBadRequest, "Invalid request"
	case CodeUnauthenticated:
		return http.StatusUnauthorized, "Unauthenticated"
	case CodePermissionDenied:
		return http.StatusForbidden, "Permission denied"
	case CodeNotFound:
		return http.StatusNotFound, "Not found"
	case CodeRateLimited:
		return http.StatusTooManyRequests, "Rate limited"
	case CodeHumanVerificationFailed:
		return http.StatusBadRequest, "Human verification failed"
	case CodeInvalidCredentials:
		return http.StatusUnauthorized, "Invalid credentials"
	case CodeUserDeactivated:
		return http.StatusForbidden, "User deactivated"
	case CodeRegistrationClosed:
		return http.StatusForbidden, "Registration closed"
	case CodeEmailTaken:
		return http.StatusConflict, "Email taken"
	case CodeInvalidResetCode:
		return http.StatusBadRequest, "Invalid reset code"
	case CodeOAuthNotConfigured:
		return http.StatusConflict, "OAuth not configured"
	case CodeOAuthFailed:
		return http.StatusBadRequest, "OAuth failed"
	case CodePendingSignInNotUsable:
		return http.StatusGone, "Pending sign-in not usable"
	case CodeSignInCodeNotSent:
		return http.StatusServiceUnavailable, "Sign-in code not sent"
	case CodeTooManySignInCodes:
		return http.StatusTooManyRequests, "Too many sign-in codes"
	case CodeSignInCodeResentTooSoon:
		return http.StatusTooManyRequests, "Sign-in code resent too soon"
	case CodeSignInCodeIncorrect:
		return http.StatusBadRequest, "Sign-in code incorrect"
	case CodeEmailNotConfigured:
		return http.StatusConflict, "Email not configured"
	case CodeAdminEmailMissing:
		return http.StatusConflict, "Admin email missing"
	case CodeAlreadyMember:
		return http.StatusConflict, "Already a member"
	case CodeCannotChangeOwnRole:
		return http.StatusConflict, "Cannot change own role"
	case CodeOwnerRequiresTransfer:
		return http.StatusConflict, "Owner requires transfer"
	case CodeWorkspaceNotEmpty:
		return http.StatusConflict, "Workspace not empty"
	case CodeCleanupPending:
		return http.StatusConflict, "Cleanup pending"
	case CodeInUse:
		return http.StatusConflict, "In use"
	case CodeDatabaseConnectionFailed:
		return http.StatusBadRequest, "Database connection failed"
	case CodeAlreadyRunning:
		return http.StatusConflict, "Already running"
	case CodeNoExtendableChain:
		return http.StatusConflict, "No extendable chain"
	case CodeNoRestorableBackup:
		return http.StatusConflict, "No restorable backup"
	case CodeTargetTimeUnreachable:
		return http.StatusConflict, "Target time unreachable"
	case CodeBackupNotCompleted:
		return http.StatusConflict, "Backup not completed"
	case CodeNotRunning:
		return http.StatusConflict, "Not running"
	case CodeInsufficientDiskSpace:
		return http.StatusConflict, "Insufficient disk space"
	case CodeVerificationGone:
		return http.StatusGone, "Verification gone"
	case CodeInstanceUnhealthy:
		return http.StatusServiceUnavailable, "Instance unhealthy"
	case CodeInternal:
		return http.StatusInternalServerError, "Internal error"
	}

	return http.StatusInternalServerError, "Internal error"
}
