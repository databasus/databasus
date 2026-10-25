// Package apierror answers the errors of contract operations with the Problem body that
// backend/api/openapi.yaml declares.
package apierror

import (
	"encoding/json"
	"net/http"

	"github.com/gin-gonic/gin"
)

// Code mirrors the ProblemCode enum of the contract; a test keeps the two equal.
type Code string

const (
	CodeInvalidRequest          Code = "invalid_request"
	CodeUnauthenticated         Code = "unauthenticated"
	CodePermissionDenied        Code = "permission_denied"
	CodeNotFound                Code = "not_found"
	CodeRateLimited             Code = "rate_limited"
	CodeHumanVerificationFailed Code = "human_verification_failed"
	CodeInvalidCredentials      Code = "invalid_credentials"
	CodeUserDeactivated         Code = "user_deactivated"
	CodeRegistrationClosed      Code = "registration_closed"
	CodeEmailTaken              Code = "email_taken"
	CodeInvalidResetCode        Code = "invalid_reset_code"
	CodeOAuthNotConfigured      Code = "oauth_not_configured"
	CodeOAuthFailed             Code = "oauth_failed"
	CodePendingSignInNotUsable  Code = "pending_sign_in_not_usable"
	CodeSignInCodeNotSent       Code = "sign_in_code_not_sent"
	CodeTooManySignInCodes      Code = "too_many_sign_in_codes"
	CodeSignInCodeResentTooSoon Code = "sign_in_code_resent_too_soon"
	CodeSignInCodeIncorrect     Code = "sign_in_code_incorrect"
	CodeEmailNotConfigured      Code = "email_not_configured"
	CodeAdminEmailMissing       Code = "admin_email_missing"
	CodeInternal                Code = "internal"
)

const contentType = "application/problem+json"

type problemBody struct {
	Type   string       `json:"type"`
	Title  string       `json:"title"`
	Status int          `json:"status"`
	Detail string       `json:"detail"`
	Code   Code         `json:"code"`
	Errors []FieldError `json:"errors,omitzero"`
}

// The detail of every internal error is the same, so a failure never reveals SQL, paths or secrets.
const internalErrorDetail = "an internal error occurred"

// NewInternalProblem is the problem of every failure the client did not cause.
func NewInternalProblem() *Problem {
	return &Problem{Code: CodeInternal, Detail: internalErrorDetail, Errors: nil}
}

// NewInvalidFieldsProblem is the problem of a request that breaks a validation rule the contract
// cannot express, naming each field.
func NewInvalidFieldsProblem(fieldErrors []FieldError) *Problem {
	return &Problem{Code: CodeInvalidRequest, Detail: "the request breaks a validation rule", Errors: fieldErrors}
}

// Write answers the problem and aborts the request, so no later handler writes a second body.
func Write(ctx *gin.Context, problem *Problem) {
	status, title := describeCode(problem.Code)

	ctx.Render(status, problemRender{body: problemBody{
		Type:   "urn:databasus:problem:" + string(problem.Code),
		Title:  title,
		Status: status,
		Detail: problem.Detail,
		Code:   problem.Code,
		Errors: problem.Errors,
	}})
	ctx.Abort()
}

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
	case CodeInternal:
		return http.StatusInternalServerError, "Internal error"
	}

	return http.StatusInternalServerError, "Internal error"
}

type problemRender struct {
	body problemBody
}

func (r problemRender) Render(writer http.ResponseWriter) error {
	r.WriteContentType(writer)

	return json.NewEncoder(writer).Encode(r.body)
}

func (r problemRender) WriteContentType(writer http.ResponseWriter) {
	writer.Header().Set("Content-Type", contentType)
}
