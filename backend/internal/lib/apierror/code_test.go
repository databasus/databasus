package apierror_test

import (
	"fmt"
	"net/http"
	"strconv"
	"testing"

	"github.com/getkin/kin-openapi/openapi3"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	contract "databasus-backend/api"
	"databasus-backend/internal/lib/apierror"
)

func TestCode(t *testing.T) {
	t.Parallel()

	codeStatuses := []struct {
		code           apierror.Code
		expectedStatus int
	}{
		{apierror.CodeInvalidRequest, http.StatusBadRequest},
		{apierror.CodeUnauthenticated, http.StatusUnauthorized},
		{apierror.CodePermissionDenied, http.StatusForbidden},
		{apierror.CodeNotFound, http.StatusNotFound},
		{apierror.CodeRateLimited, http.StatusTooManyRequests},
		{apierror.CodeHumanVerificationFailed, http.StatusBadRequest},
		{apierror.CodeInvalidCredentials, http.StatusUnauthorized},
		{apierror.CodeUserDeactivated, http.StatusForbidden},
		{apierror.CodeRegistrationClosed, http.StatusForbidden},
		{apierror.CodeEmailTaken, http.StatusConflict},
		{apierror.CodeInvalidResetCode, http.StatusBadRequest},
		{apierror.CodeOAuthNotConfigured, http.StatusConflict},
		{apierror.CodeOAuthFailed, http.StatusBadRequest},
		{apierror.CodePendingSignInNotUsable, http.StatusGone},
		{apierror.CodeSignInCodeNotSent, http.StatusServiceUnavailable},
		{apierror.CodeTooManySignInCodes, http.StatusTooManyRequests},
		{apierror.CodeSignInCodeResentTooSoon, http.StatusTooManyRequests},
		{apierror.CodeSignInCodeIncorrect, http.StatusBadRequest},
		{apierror.CodeEmailNotConfigured, http.StatusConflict},
		{apierror.CodeAdminEmailMissing, http.StatusConflict},
		{apierror.CodeAlreadyMember, http.StatusConflict},
		{apierror.CodeCannotChangeOwnRole, http.StatusConflict},
		{apierror.CodeOwnerRequiresTransfer, http.StatusConflict},
		{apierror.CodeWorkspaceNotEmpty, http.StatusConflict},
		{apierror.CodeInUse, http.StatusConflict},
		{apierror.CodeCleanupPending, http.StatusConflict},
		{apierror.CodeDatabaseConnectionFailed, http.StatusBadRequest},
		{apierror.CodeAlreadyRunning, http.StatusConflict},
		{apierror.CodeNotRunning, http.StatusConflict},
		{apierror.CodeBackupNotCompleted, http.StatusConflict},
		{apierror.CodeNoExtendableChain, http.StatusConflict},
		{apierror.CodeNoRestorableBackup, http.StatusConflict},
		{apierror.CodeTargetTimeUnreachable, http.StatusConflict},
		{apierror.CodeInsufficientDiskSpace, http.StatusConflict},
		{apierror.CodeInternal, http.StatusInternalServerError},
		{apierror.CodeVerificationGone, http.StatusGone},
		{apierror.CodeInstanceUnhealthy, http.StatusServiceUnavailable},
	}

	for _, codeStatus := range codeStatuses {
		t.Run("answers "+strconv.Itoa(codeStatus.expectedStatus)+" when the code is "+string(codeStatus.code),
			func(t *testing.T) {
				t.Parallel()
				router := gin.New()
				router.GET("/api/v1/items", func(ctx *gin.Context) {
					apierror.Write(ctx, &apierror.Problem{Code: codeStatus.code, Detail: "detail", Errors: nil})
				})

				response := serve(t, router, "/api/v1/items")

				require.Equal(t, codeStatus.expectedStatus, response.Code)
				assert.Equal(t, string(codeStatus.code), decodeProblem(t, response).Code)
			})
	}

	t.Run("covers both contracts and appears in at least one contract", func(t *testing.T) {
		t.Parallel()
		goCodes := []apierror.Code{
			apierror.CodeInvalidRequest,
			apierror.CodeUnauthenticated,
			apierror.CodePermissionDenied,
			apierror.CodeNotFound,
			apierror.CodeRateLimited,
			apierror.CodeHumanVerificationFailed,
			apierror.CodeInvalidCredentials,
			apierror.CodeUserDeactivated,
			apierror.CodeRegistrationClosed,
			apierror.CodeEmailTaken,
			apierror.CodeInvalidResetCode,
			apierror.CodeOAuthNotConfigured,
			apierror.CodeOAuthFailed,
			apierror.CodePendingSignInNotUsable,
			apierror.CodeSignInCodeNotSent,
			apierror.CodeTooManySignInCodes,
			apierror.CodeSignInCodeResentTooSoon,
			apierror.CodeSignInCodeIncorrect,
			apierror.CodeEmailNotConfigured,
			apierror.CodeAdminEmailMissing,
			apierror.CodeAlreadyMember,
			apierror.CodeCannotChangeOwnRole,
			apierror.CodeOwnerRequiresTransfer,
			apierror.CodeWorkspaceNotEmpty,
			apierror.CodeInUse,
			apierror.CodeCleanupPending,
			apierror.CodeDatabaseConnectionFailed,
			apierror.CodeAlreadyRunning,
			apierror.CodeNotRunning,
			apierror.CodeBackupNotCompleted,
			apierror.CodeNoExtendableChain,
			apierror.CodeNoRestorableBackup,
			apierror.CodeTargetTimeUnreachable,
			apierror.CodeInsufficientDiskSpace,
			apierror.CodeInternal,
			apierror.CodeVerificationGone,
			apierror.CodeInstanceUnhealthy,
		}

		contractCodes := []string{}
		for _, document := range [][]byte{contract.OpenAPI, contract.Agent} {
			loadedContract, err := openapi3.NewLoader().LoadFromData(document)
			require.NoError(t, err)
			problemCodeSchema := loadedContract.Components.Schemas["ProblemCode"]
			require.NotNil(t, problemCodeSchema)
			for _, contractCode := range problemCodeSchema.Value.Enum {
				code := fmt.Sprint(contractCode)
				assert.Contains(t, goCodes, apierror.Code(code))
				contractCodes = append(contractCodes, code)
			}
		}

		declaredCodes := make([]string, 0, len(goCodes))
		for _, goCode := range goCodes {
			declaredCodes = append(declaredCodes, string(goCode))
		}

		for _, code := range declaredCodes {
			assert.Contains(t, contractCodes, code)
		}
	})
}
