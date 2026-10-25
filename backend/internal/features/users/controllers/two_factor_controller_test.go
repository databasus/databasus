package users_controllers

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/oauth2"

	"databasus-backend/internal/features/email"
	users_dto "databasus-backend/internal/features/users/dto"
	users_models "databasus-backend/internal/features/users/models"
	users_repositories "databasus-backend/internal/features/users/repositories"
	users_services "databasus-backend/internal/features/users/services"
	users_testing "databasus-backend/internal/features/users/testing"
	"databasus-backend/internal/storage"
	cloudflare_turnstile "databasus-backend/internal/util/cloudflare_turnstile"
	test_utils "databasus-backend/internal/util/testing"
)

const twoFactorTestPassword = "twofactorpassword123"

func Test_SignIn_WhenTheSecondFactorIsOn_AnswersWithAPendingSignInAndSendsOneCode(t *testing.T) {
	router, mailSender := createTwoFactorTestRouter(t)
	account := registerTwoFactorTestAccount(t, router)

	pendingSignIn := submitPassword(t, router, account, http.StatusOK)

	assert.NotEqual(t, uuid.Nil, pendingSignIn.PendingSignInID)
	assert.Equal(t, account, pendingSignIn.Email)

	require.Len(t, mailSender.SentEmails, 1)
	assert.Equal(t, account, mailSender.SentEmails[0].To)

	pendingCode := readPendingSignIn(t, pendingSignIn.PendingSignInID)
	assert.False(t, pendingCode.IsUsed)
}

func Test_SignIn_WhenTheSecondFactorIsOn_CarriesNoToken(t *testing.T) {
	router, _ := createTwoFactorTestRouter(t)
	account := registerTwoFactorTestAccount(t, router)

	var response users_dto.SignInResponseDTO
	test_utils.MakePostRequestAndUnmarshal(
		t,
		router,
		"/api/v1/users/signin",
		"",
		users_dto.SignInRequestDTO{Email: account, Password: twoFactorTestPassword},
		http.StatusOK,
		&response,
	)

	assert.Empty(t, response.Token)
}

func Test_SignIn_WhenTheSecondFactorIsOn_StoresTheCodeOnlyHashed(t *testing.T) {
	router, mailSender := createTwoFactorTestRouter(t)
	account := registerTwoFactorTestAccount(t, router)

	pendingSignIn := submitPassword(t, router, account, http.StatusOK)
	code := extractCodeFromEmail(mailSender.SentEmails[0].Body)
	require.Len(t, code, 6)

	pendingCode := readPendingSignIn(t, pendingSignIn.PendingSignInID)
	assert.NotContains(t, pendingCode.HashedCode, code)
}

func Test_SignIn_CodeMessage_CarriesTheCodeAndTheRulesItFollows(t *testing.T) {
	router, mailSender := createTwoFactorTestRouter(t)
	account := registerTwoFactorTestAccount(t, router)

	submitPassword(t, router, account, http.StatusOK)

	require.Len(t, mailSender.SentEmails, 1)
	message := mailSender.SentEmails[0].Body

	assert.Contains(t, message, extractCodeFromEmail(message))
	assert.Contains(t, message, "stops working")
	assert.Contains(t, message, "10 minutes")
	assert.Contains(t, message, "Requesting another code replaces this one")
}

func Test_VerifySignInCode_WithTheCodeThatArrived_ReturnsAToken(t *testing.T) {
	router, mailSender := createTwoFactorTestRouter(t)
	account := registerTwoFactorTestAccount(t, router)

	pendingSignIn := submitPassword(t, router, account, http.StatusOK)
	code := extractCodeFromEmail(mailSender.SentEmails[0].Body)

	var response users_dto.SignInResponseDTO
	test_utils.MakePostRequestAndUnmarshal(
		t,
		router,
		"/api/v1/users/verify-signin-code",
		"",
		users_dto.VerifySignInCodeRequestDTO{PendingSignInID: pendingSignIn.PendingSignInID, Code: code},
		http.StatusOK,
		&response,
	)

	assert.NotEmpty(t, response.Token)
	assert.Equal(t, account, response.Email)
	assert.True(t, readPendingSignIn(t, pendingSignIn.PendingSignInID).IsUsed)
}

func Test_VerifySignInCode_WhenTheCodeAlreadyCompletedASignIn_IsRefused(t *testing.T) {
	router, mailSender := createTwoFactorTestRouter(t)
	account := registerTwoFactorTestAccount(t, router)

	pendingSignIn := submitPassword(t, router, account, http.StatusOK)
	code := extractCodeFromEmail(mailSender.SentEmails[0].Body)

	verifyCode(t, router, pendingSignIn.PendingSignInID, code, http.StatusOK)
	verifyCode(t, router, pendingSignIn.PendingSignInID, code, http.StatusGone)
}

func Test_VerifySignInCode_WhenTheCodeHasExpired_IsRefused(t *testing.T) {
	router, mailSender := createTwoFactorTestRouter(t)
	account := registerTwoFactorTestAccount(t, router)

	pendingSignIn := submitPassword(t, router, account, http.StatusOK)
	code := extractCodeFromEmail(mailSender.SentEmails[0].Body)

	expirePendingSignIn(t, pendingSignIn.PendingSignInID)

	verifyCode(t, router, pendingSignIn.PendingSignInID, code, http.StatusGone)
}

func Test_VerifySignInCode_WithACodeFromAnotherPendingSignIn_IsRefused(t *testing.T) {
	router, mailSender := createTwoFactorTestRouter(t)
	firstAccount := registerTwoFactorTestAccount(t, router)
	secondAccount := registerTwoFactorTestAccount(t, router)

	firstPendingSignIn := submitPassword(t, router, firstAccount, http.StatusOK)
	firstCode := extractCodeFromEmail(mailSender.SentEmails[0].Body)

	secondPendingSignIn := submitPassword(t, router, secondAccount, http.StatusOK)

	verifyCode(t, router, secondPendingSignIn.PendingSignInID, firstCode, http.StatusBadRequest)
	assert.NotEqual(t, firstPendingSignIn.PendingSignInID, secondPendingSignIn.PendingSignInID)
}

func Test_VerifySignInCode_WithAnIdentifierThatNeverExisted_IsRefused(t *testing.T) {
	router, _ := createTwoFactorTestRouter(t)

	verifyCode(t, router, uuid.New(), "123456", http.StatusGone)
}

func Test_VerifySignInCode_AfterFiveWrongCodes_TheCorrectOneNoLongerWorks(t *testing.T) {
	router, mailSender := createTwoFactorTestRouter(t)
	account := registerTwoFactorTestAccount(t, router)
	recorder := users_testing.GetAuditLogRecorder()

	pendingSignIn := submitPassword(t, router, account, http.StatusOK)
	code := extractCodeFromEmail(mailSender.SentEmails[0].Body)

	for range users_models.MaxTwoFactorCodeAttempts - 1 {
		verifyCode(t, router, pendingSignIn.PendingSignInID, wrongCodeFor(code), http.StatusBadRequest)
	}

	// The attempt that exhausts them destroys the pending sign-in, which is a
	// different answer from a wrong guess the user may try again after.
	verifyCode(t, router, pendingSignIn.PendingSignInID, wrongCodeFor(code), http.StatusGone)

	verifyCode(t, router, pendingSignIn.PendingSignInID, code, http.StatusGone)
	assert.True(t, recorder.HasEntryContaining("Two-factor sign-in abandoned", account))
}

func Test_ResendSignInCode_WithinTheSameMinute_IsRefused(t *testing.T) {
	router, mailSender := createTwoFactorTestRouter(t)
	account := registerTwoFactorTestAccount(t, router)

	pendingSignIn := submitPassword(t, router, account, http.StatusOK)

	var resent users_dto.PendingSignInResponseDTO
	test_utils.MakePostRequestAndUnmarshal(
		t,
		router,
		"/api/v1/users/resend-signin-code",
		"",
		users_dto.ResendSignInCodeRequestDTO{PendingSignInID: pendingSignIn.PendingSignInID},
		http.StatusOK,
		&resent,
	)
	messagesAfterFirstResend := len(mailSender.SentEmails)

	resendCode(t, router, resent.PendingSignInID, http.StatusTooManyRequests)
	assert.Len(t, mailSender.SentEmails, messagesAfterFirstResend)
}

func Test_ResendSignInCode_AfterASuccessfulResend_ThePreviousCodeStopsWorking(t *testing.T) {
	router, mailSender := createTwoFactorTestRouter(t)
	account := registerTwoFactorTestAccount(t, router)

	pendingSignIn := submitPassword(t, router, account, http.StatusOK)
	firstCode := extractCodeFromEmail(mailSender.SentEmails[0].Body)

	var resent users_dto.PendingSignInResponseDTO
	test_utils.MakePostRequestAndUnmarshal(
		t,
		router,
		"/api/v1/users/resend-signin-code",
		"",
		users_dto.ResendSignInCodeRequestDTO{PendingSignInID: pendingSignIn.PendingSignInID},
		http.StatusOK,
		&resent,
	)

	require.Len(t, mailSender.SentEmails, 2)
	secondCode := extractCodeFromEmail(mailSender.SentEmails[1].Body)

	verifyCode(t, router, pendingSignIn.PendingSignInID, firstCode, http.StatusGone)
	verifyCode(t, router, resent.PendingSignInID, secondCode, http.StatusOK)
}

func Test_SignIn_WhileAPendingSignInIsStillLive_ReturnsItAndSendsNothing(t *testing.T) {
	router, mailSender := createTwoFactorTestRouter(t)
	account := registerTwoFactorTestAccount(t, router)

	firstPendingSignIn := submitPassword(t, router, account, http.StatusOK)
	code := extractCodeFromEmail(mailSender.SentEmails[0].Body)

	secondPendingSignIn := submitPassword(t, router, account, http.StatusOK)

	assert.Equal(t, firstPendingSignIn.PendingSignInID, secondPendingSignIn.PendingSignInID)
	assert.Len(t, mailSender.SentEmails, 1)
	assert.Equal(t, int64(1), countRecentCodes(t, firstPendingSignIn.PendingSignInID))

	verifyCode(t, router, firstPendingSignIn.PendingSignInID, code, http.StatusOK)
}

func Test_SignIn_WhenTheAccountHasReachedTheHourlyCap_RefusesBeforeSendingMail(t *testing.T) {
	router, mailSender := createTwoFactorTestRouter(t)
	account := registerTwoFactorTestAccount(t, router)

	for range 5 {
		pendingSignIn := submitPassword(t, router, account, http.StatusOK)
		expirePendingSignIn(t, pendingSignIn.PendingSignInID)
	}

	messagesBeforeTheCap := len(mailSender.SentEmails)
	require.Equal(t, 5, messagesBeforeTheCap)

	response := test_utils.MakePostRequest(
		t,
		router,
		"/api/v1/users/signin",
		"",
		users_dto.SignInRequestDTO{Email: account, Password: twoFactorTestPassword},
		http.StatusTooManyRequests,
	)

	assert.Contains(t, string(response.Body), "too many codes")
	assert.Contains(t, string(response.Body), `"code":"too_many_sign_in_codes"`)
	assert.Len(t, mailSender.SentEmails, messagesBeforeTheCap)
}

func Test_SignIn_WhenTheMailServerIsMissing_FailsClosed(t *testing.T) {
	router, mailSender := createTwoFactorTestRouter(t)
	account := registerTwoFactorTestAccount(t, router)

	mailSender.IsMailServerMissing = true

	response := test_utils.MakePostRequest(
		t,
		router,
		"/api/v1/users/signin",
		"",
		users_dto.SignInRequestDTO{Email: account, Password: twoFactorTestPassword},
		http.StatusServiceUnavailable,
	)

	assert.Contains(t, string(response.Body), "could not send the sign-in code")
	assert.Nil(t, findLivePendingSignIn(t, account))
}

func Test_SignIn_WhenTheSendFails_FailsClosed(t *testing.T) {
	router, mailSender := createTwoFactorTestRouter(t)
	account := registerTwoFactorTestAccount(t, router)

	mailSender.ShouldFail = true

	response := test_utils.MakePostRequest(
		t,
		router,
		"/api/v1/users/signin",
		"",
		users_dto.SignInRequestDTO{Email: account, Password: twoFactorTestPassword},
		http.StatusServiceUnavailable,
	)

	assert.Contains(t, string(response.Body), "could not send the sign-in code")
	assert.Nil(t, findLivePendingSignIn(t, account))
}

func Test_VerifySignInCode_WhenTheAccountWasDeactivated_IsRefused(t *testing.T) {
	router, mailSender := createTwoFactorTestRouter(t)
	account := registerTwoFactorTestAccount(t, router)

	pendingSignIn := submitPassword(t, router, account, http.StatusOK)
	code := extractCodeFromEmail(mailSender.SentEmails[0].Body)

	pendingCode := readPendingSignIn(t, pendingSignIn.PendingSignInID)
	users_testing.DeactivateTestUser(t.Context(), pendingCode.UserID)

	verifyCode(t, router, pendingSignIn.PendingSignInID, code, http.StatusGone)
}

func Test_VerifySignInCode_WhenThePasswordChangedInTheMeantime_IsRefused(t *testing.T) {
	router, mailSender := createTwoFactorTestRouter(t)
	account := registerTwoFactorTestAccount(t, router)

	pendingSignIn := submitPassword(t, router, account, http.StatusOK)
	code := extractCodeFromEmail(mailSender.SentEmails[0].Body)

	pendingCode := readPendingSignIn(t, pendingSignIn.PendingSignInID)
	require.NoError(
		t,
		users_services.GetUserService().ChangeUserPassword(t.Context(), pendingCode.UserID, "another-password-123"),
	)

	verifyCode(t, router, pendingSignIn.PendingSignInID, code, http.StatusGone)
}

func Test_SignIn_AfterThePasswordChangedDuringAPendingSignIn_SendsAFreshCode(t *testing.T) {
	router, mailSender := createTwoFactorTestRouter(t)
	account := registerTwoFactorTestAccount(t, router)

	stalePendingSignIn := submitPassword(t, router, account, http.StatusOK)
	stalePendingCode := readPendingSignIn(t, stalePendingSignIn.PendingSignInID)
	require.NoError(
		t,
		users_services.GetUserService().
			ChangeUserPassword(t.Context(), stalePendingCode.UserID, "another-password-123"),
	)

	var freshPendingSignIn users_dto.PendingSignInResponseDTO
	test_utils.MakePostRequestAndUnmarshal(
		t,
		router,
		"/api/v1/users/signin",
		"",
		users_dto.SignInRequestDTO{Email: account, Password: "another-password-123"},
		http.StatusOK,
		&freshPendingSignIn,
	)

	assert.NotEqual(t, stalePendingSignIn.PendingSignInID, freshPendingSignIn.PendingSignInID)
	require.Len(t, mailSender.SentEmails, 2)

	freshCode := extractCodeFromEmail(mailSender.SentEmails[1].Body)
	verifyCode(t, router, freshPendingSignIn.PendingSignInID, freshCode, http.StatusOK)
}

func Test_VerifySignInCode_WhenTheSameCodeArrivesTwiceAtOnce_IssuesOneToken(t *testing.T) {
	router, mailSender := createTwoFactorTestRouter(t)
	account := registerTwoFactorTestAccount(t, router)

	pendingSignIn := submitPassword(t, router, account, http.StatusOK)
	code := extractCodeFromEmail(mailSender.SentEmails[0].Body)

	statuses := postConcurrently(
		router,
		"/api/v1/users/verify-signin-code",
		users_dto.VerifySignInCodeRequestDTO{PendingSignInID: pendingSignIn.PendingSignInID, Code: code},
		4,
	)

	assert.Equal(t, 1, countStatus(statuses, http.StatusOK))
	assert.Equal(t, 3, countStatus(statuses, http.StatusGone))
}

func Test_VerifySignInCode_WhenWrongCodesArriveAtOnce_ChecksNoMoreThanTheAllowedFive(t *testing.T) {
	router, mailSender := createTwoFactorTestRouter(t)
	account := registerTwoFactorTestAccount(t, router)

	pendingSignIn := submitPassword(t, router, account, http.StatusOK)
	code := extractCodeFromEmail(mailSender.SentEmails[0].Body)

	statuses := postConcurrently(
		router,
		"/api/v1/users/verify-signin-code",
		users_dto.VerifySignInCodeRequestDTO{
			PendingSignInID: pendingSignIn.PendingSignInID,
			Code:            wrongCodeFor(code),
		},
		10,
	)

	// Four guesses are told the code is wrong, the fifth destroys the pending
	// sign-in, and the rest find nothing left to guess at.
	assert.Equal(t, users_models.MaxTwoFactorCodeAttempts-1, countStatus(statuses, http.StatusBadRequest))
	assert.Equal(
		t,
		10-(users_models.MaxTwoFactorCodeAttempts-1),
		countStatus(statuses, http.StatusGone),
	)
	assert.Equal(
		t,
		users_models.MaxTwoFactorCodeAttempts,
		readPendingSignIn(t, pendingSignIn.PendingSignInID).FailedAttemptCount,
	)
}

func Test_SignIn_WhenTwoPasswordStepsArriveAtOnce_SendsOneCode(t *testing.T) {
	router, mailSender := createTwoFactorTestRouter(t)
	account := registerTwoFactorTestAccount(t, router)

	statuses := postConcurrently(
		router,
		"/api/v1/users/signin",
		users_dto.SignInRequestDTO{Email: account, Password: twoFactorTestPassword},
		3,
	)

	assert.Equal(t, 3, countStatus(statuses, http.StatusOK))
	assert.Len(t, mailSender.SentEmails, 1)
}

func Test_SignInCodeRefusals_CarryACodeTheInterfaceCanTranslate(t *testing.T) {
	router, mailSender := createTwoFactorTestRouter(t)
	account := registerTwoFactorTestAccount(t, router)

	pendingSignIn := submitPassword(t, router, account, http.StatusOK)
	code := extractCodeFromEmail(mailSender.SentEmails[0].Body)

	wrongCodeResponse := verifyCode(
		t, router, pendingSignIn.PendingSignInID, wrongCodeFor(code), http.StatusBadRequest,
	)
	assert.Contains(t, string(wrongCodeResponse.Body), `"code":"sign_in_code_incorrect"`)

	var resent users_dto.PendingSignInResponseDTO
	test_utils.MakePostRequestAndUnmarshal(
		t,
		router,
		"/api/v1/users/resend-signin-code",
		"",
		users_dto.ResendSignInCodeRequestDTO{PendingSignInID: pendingSignIn.PendingSignInID},
		http.StatusOK,
		&resent,
	)

	tooSoonResponse := resendCode(t, router, resent.PendingSignInID, http.StatusTooManyRequests)
	assert.Contains(t, string(tooSoonResponse.Body), `"code":"sign_in_code_resent_too_soon"`)

	goneResponse := verifyCode(t, router, pendingSignIn.PendingSignInID, code, http.StatusGone)
	assert.Contains(t, string(goneResponse.Body), `"code":"pending_sign_in_not_usable"`)
}

func Test_VerifySignInCode_AfterTheSettingWasSwitchedOff_StillCompletes(t *testing.T) {
	router, mailSender := createTwoFactorTestRouter(t)
	account := registerTwoFactorTestAccount(t, router)

	pendingSignIn := submitPassword(t, router, account, http.StatusOK)
	code := extractCodeFromEmail(mailSender.SentEmails[0].Body)

	users_testing.DisableTwoFactorAuth(t.Context())

	verifyCode(t, router, pendingSignIn.PendingSignInID, code, http.StatusOK)
}

func Test_SignIn_WithAWrongPassword_SendsNoMailAndAnswersAsItDoesWithOneFactor(t *testing.T) {
	router, mailSender := createTwoFactorTestRouter(t)
	account := registerTwoFactorTestAccount(t, router)

	response := test_utils.MakePostRequest(
		t,
		router,
		"/api/v1/users/signin",
		"",
		users_dto.SignInRequestDTO{Email: account, Password: "not-the-password"},
		http.StatusBadRequest,
	)

	assert.Contains(t, string(response.Body), "password is incorrect")
	assert.Empty(t, mailSender.SentEmails)
	assert.Nil(t, findLivePendingSignIn(t, account))
}

func Test_SignIn_WithAnUnknownAddress_SendsNoMailAndCreatesNoPendingSignIn(t *testing.T) {
	router, mailSender := createTwoFactorTestRouter(t)
	unknownAddress := "nobody-" + uuid.New().String() + "@example.com"

	response := test_utils.MakePostRequest(
		t,
		router,
		"/api/v1/users/signin",
		"",
		users_dto.SignInRequestDTO{Email: unknownAddress, Password: twoFactorTestPassword},
		http.StatusBadRequest,
	)

	assert.Contains(t, string(response.Body), "does not exist")
	assert.Empty(t, mailSender.SentEmails)
	assert.Nil(t, findLivePendingSignIn(t, unknownAddress))
}

func Test_SignIn_WhenTheSecondFactorIsOff_ReturnsATokenInOneStep(t *testing.T) {
	router, mailSender := createTwoFactorTestRouter(t)
	account := registerTwoFactorTestAccount(t, router)

	users_testing.DisableTwoFactorAuth(t.Context())

	var response users_dto.SignInResponseDTO
	test_utils.MakePostRequestAndUnmarshal(
		t,
		router,
		"/api/v1/users/signin",
		"",
		users_dto.SignInRequestDTO{Email: account, Password: twoFactorTestPassword},
		http.StatusOK,
		&response,
	)

	assert.NotEmpty(t, response.Token)
	assert.Empty(t, mailSender.SentEmails)
}

func Test_SignIn_TheAuditTrailRecordsTheStepThatGrantsAccess(t *testing.T) {
	router, mailSender := createTwoFactorTestRouter(t)
	account := registerTwoFactorTestAccount(t, router)
	recorder := users_testing.GetAuditLogRecorder()

	pendingSignIn := submitPassword(t, router, account, http.StatusOK)
	assert.False(t, recorder.HasEntryContaining("User signed in with email", account))

	code := extractCodeFromEmail(mailSender.SentEmails[0].Body)
	verifyCode(t, router, pendingSignIn.PendingSignInID, code, http.StatusOK)

	assert.Equal(t, 1, countAuditEntries(recorder, "User signed in with email: "+account))
}

func Test_VerifySignInCode_WhenTheChallengeIsEnabled_RefusesWithoutAnAnswer(t *testing.T) {
	router, mailSender := createTwoFactorTestRouter(t)
	account := registerTwoFactorTestAccount(t, router)

	pendingSignIn := submitPassword(t, router, account, http.StatusOK)
	code := extractCodeFromEmail(mailSender.SentEmails[0].Body)

	t.Cleanup(cloudflare_turnstile.EnableChallengeForTesting("test-secret-key"))

	response := test_utils.MakePostRequest(
		t,
		router,
		"/api/v1/users/verify-signin-code",
		"",
		users_dto.VerifySignInCodeRequestDTO{PendingSignInID: pendingSignIn.PendingSignInID, Code: code},
		http.StatusBadRequest,
	)

	assert.Contains(t, string(response.Body), "Turnstile verification required")
	assert.False(t, readPendingSignIn(t, pendingSignIn.PendingSignInID).IsUsed)
}

func Test_ResendSignInCode_WhenTheChallengeIsEnabled_RefusesWithoutAnAnswer(t *testing.T) {
	router, mailSender := createTwoFactorTestRouter(t)
	account := registerTwoFactorTestAccount(t, router)

	pendingSignIn := submitPassword(t, router, account, http.StatusOK)
	messagesBeforeTheResend := len(mailSender.SentEmails)

	t.Cleanup(cloudflare_turnstile.EnableChallengeForTesting("test-secret-key"))

	response := test_utils.MakePostRequest(
		t,
		router,
		"/api/v1/users/resend-signin-code",
		"",
		users_dto.ResendSignInCodeRequestDTO{PendingSignInID: pendingSignIn.PendingSignInID},
		http.StatusBadRequest,
	)

	assert.Contains(t, string(response.Body), "Turnstile verification required")
	assert.Len(t, mailSender.SentEmails, messagesBeforeTheResend)
}

func Test_SweepPendingSignIns_RemovesOnlyWhatIsPastItsRetention(t *testing.T) {
	router, mailSender := createTwoFactorTestRouter(t)
	account := registerTwoFactorTestAccount(t, router)

	agedPendingSignIn := submitPassword(t, router, account, http.StatusOK)
	agePendingSignIn(t, agedPendingSignIn.PendingSignInID, 2*time.Hour)
	expirePendingSignIn(t, agedPendingSignIn.PendingSignInID)

	mailSender.SentEmails = nil
	freshPendingSignIn := submitPassword(t, router, account, http.StatusOK)

	require.NoError(t, users_services.GetUserService().SweepPendingSignInsPastRetention(t.Context()))

	assert.Nil(t, findPendingSignIn(t, agedPendingSignIn.PendingSignInID))
	assert.NotNil(t, findPendingSignIn(t, freshPendingSignIn.PendingSignInID))
}

func Test_SignIn_AfterASweep_TheHourlyCapStillCounts(t *testing.T) {
	router, _ := createTwoFactorTestRouter(t)
	account := registerTwoFactorTestAccount(t, router)

	for range 5 {
		pendingSignIn := submitPassword(t, router, account, http.StatusOK)
		expirePendingSignIn(t, pendingSignIn.PendingSignInID)
	}

	require.NoError(t, users_services.GetUserService().SweepPendingSignInsPastRetention(t.Context()))

	test_utils.MakePostRequest(
		t,
		router,
		"/api/v1/users/signin",
		"",
		users_dto.SignInRequestDTO{Email: account, Password: twoFactorTestPassword},
		http.StatusTooManyRequests,
	)
}

func createTwoFactorTestRouter(t *testing.T) (*gin.Engine, *users_testing.MockEmailSender) {
	router := createUserTestRouter()

	mailSender := users_testing.NewMockEmailSender()
	users_services.GetUserService().SetEmailSender(mailSender)

	users_testing.ResetSettingsToDefaults(t.Context())
	users_testing.EnableTwoFactorAuth(t.Context())
	t.Cleanup(func() {
		users_services.GetUserService().SetEmailSender(email.GetInstanceSender())
		users_testing.ResetSettingsToDefaults(context.Background())
	})

	return router, mailSender
}

func registerTwoFactorTestAccount(t *testing.T, router *gin.Engine) string {
	address := "twofactor-" + uuid.New().String() + "@example.com"

	test_utils.MakePostRequest(
		t,
		router,
		"/api/v1/users/signup",
		"",
		users_dto.SignUpRequestDTO{
			Email:    address,
			Password: twoFactorTestPassword,
			Name:     "Two Factor User",
		},
		http.StatusOK,
	)

	return address
}

func submitPassword(
	t *testing.T,
	router *gin.Engine,
	address string,
	expectedStatus int,
) users_dto.PendingSignInResponseDTO {
	var pendingSignIn users_dto.PendingSignInResponseDTO

	test_utils.MakePostRequestAndUnmarshal(
		t,
		router,
		"/api/v1/users/signin",
		"",
		users_dto.SignInRequestDTO{Email: address, Password: twoFactorTestPassword},
		expectedStatus,
		&pendingSignIn,
	)

	return pendingSignIn
}

func verifyCode(
	t *testing.T,
	router *gin.Engine,
	pendingSignInID uuid.UUID,
	code string,
	expectedStatus int,
) *test_utils.TestResponse {
	return test_utils.MakePostRequest(
		t,
		router,
		"/api/v1/users/verify-signin-code",
		"",
		users_dto.VerifySignInCodeRequestDTO{PendingSignInID: pendingSignInID, Code: code},
		expectedStatus,
	)
}

func resendCode(
	t *testing.T,
	router *gin.Engine,
	pendingSignInID uuid.UUID,
	expectedStatus int,
) *test_utils.TestResponse {
	return test_utils.MakePostRequest(
		t,
		router,
		"/api/v1/users/resend-signin-code",
		"",
		users_dto.ResendSignInCodeRequestDTO{PendingSignInID: pendingSignInID},
		expectedStatus,
	)
}

func readPendingSignIn(t *testing.T, pendingSignInID uuid.UUID) *users_models.TwoFactorCode {
	pendingCode := findPendingSignIn(t, pendingSignInID)
	require.NotNil(t, pendingCode)

	return pendingCode
}

func findPendingSignIn(t *testing.T, pendingSignInID uuid.UUID) *users_models.TwoFactorCode {
	pendingCode, err := (&users_repositories.TwoFactorRepository{}).
		GetCodeByID(t.Context(), pendingSignInID)
	require.NoError(t, err)

	return pendingCode
}

func findLivePendingSignIn(t *testing.T, address string) *users_models.TwoFactorCode {
	user, err := (&users_repositories.UserRepository{}).GetUserByEmail(t.Context(), address)
	require.NoError(t, err)

	if user == nil {
		return nil
	}

	var pendingCodes []users_models.TwoFactorCode

	require.NoError(t, storage.GetDb().WithContext(t.Context()).
		Where(
			"user_id = ? AND is_used = ? AND failed_attempt_count < ? AND expires_at > ?",
			user.ID,
			false,
			users_models.MaxTwoFactorCodeAttempts,
			time.Now().UTC(),
		).
		Order("created_at DESC").
		Limit(1).
		Find(&pendingCodes).Error)

	if len(pendingCodes) == 0 {
		return nil
	}

	return &pendingCodes[0]
}

func countRecentCodes(t *testing.T, pendingSignInID uuid.UUID) int64 {
	pendingCode := readPendingSignIn(t, pendingSignInID)

	var count int64

	require.NoError(t, storage.GetDb().WithContext(t.Context()).
		Model(&users_models.TwoFactorCode{}).
		Where("user_id = ? AND created_at > ?", pendingCode.UserID, time.Now().UTC().Add(-time.Hour)).
		Count(&count).Error)

	return count
}

func expirePendingSignIn(t *testing.T, pendingSignInID uuid.UUID) {
	require.NoError(t, storage.GetDb().Model(&users_models.TwoFactorCode{}).
		Where("id = ?", pendingSignInID).
		Update("expires_at", time.Now().UTC().Add(-time.Minute)).Error)
}

func agePendingSignIn(t *testing.T, pendingSignInID uuid.UUID, age time.Duration) {
	require.NoError(t, storage.GetDb().Model(&users_models.TwoFactorCode{}).
		Where("id = ?", pendingSignInID).
		Update("created_at", time.Now().UTC().Add(-age)).Error)
}

// Requests go straight to the router rather than through test_utils, whose
// assertions may not run outside the test goroutine.
func postConcurrently(router *gin.Engine, url string, body any, requestCount int) []int {
	encodedBody, err := json.Marshal(body)
	if err != nil {
		panic(err)
	}

	statuses := make([]int, requestCount)

	var requestsDone sync.WaitGroup

	for requestIndex := range requestCount {
		requestsDone.Go(func() {
			request := httptest.NewRequest(http.MethodPost, url, bytes.NewReader(encodedBody))
			request.Header.Set("Content-Type", "application/json")

			recorder := httptest.NewRecorder()
			router.ServeHTTP(recorder, request)

			statuses[requestIndex] = recorder.Code
		})
	}

	requestsDone.Wait()

	return statuses
}

func countStatus(statuses []int, status int) int {
	matchingCount := 0

	for _, receivedStatus := range statuses {
		if receivedStatus == status {
			matchingCount++
		}
	}

	return matchingCount
}

func wrongCodeFor(code string) string {
	if code == "000000" {
		return "111111"
	}

	return "000000"
}

func Test_GitHubOAuth_WhenTheSecondFactorIsOn_StillIssuesATokenDirectly(t *testing.T) {
	_, mailSender := createTwoFactorTestRouter(t)

	address := "github-twofactor-" + uuid.New().String() + "@example.com"
	oauthServer := startGitHubOAuthMockServer(t, address)

	response, err := users_services.GetUserService().HandleGitHubOAuthWithMockEndpoint(
		t.Context(),
		"test-code",
		"http://localhost:3000/auth/callback",
		oauth2.Endpoint{
			AuthURL:  oauthServer.URL + "/login/oauth/authorize",
			TokenURL: oauthServer.URL + "/login/oauth/access_token",
		},
		oauthServer.URL+"/user",
	)

	require.NoError(t, err)
	assert.NotEmpty(t, response.Token)
	assert.Empty(t, mailSender.SentEmails)
}

func Test_GoogleOAuth_WhenTheSecondFactorIsOn_StillIssuesATokenDirectly(t *testing.T) {
	_, mailSender := createTwoFactorTestRouter(t)

	address := "google-twofactor-" + uuid.New().String() + "@example.com"
	oauthServer := startGoogleOAuthMockServer(t, address)

	response, err := users_services.GetUserService().HandleGoogleOAuthWithMockEndpoint(
		t.Context(),
		"test-code",
		"http://localhost:3000/auth/callback",
		oauth2.Endpoint{
			AuthURL:  oauthServer.URL + "/o/oauth2/auth",
			TokenURL: oauthServer.URL + "/token",
		},
		oauthServer.URL+"/userinfo",
	)

	require.NoError(t, err)
	assert.NotEmpty(t, response.Token)
	assert.Empty(t, mailSender.SentEmails)
}

func startGitHubOAuthMockServer(t *testing.T, address string) *httptest.Server {
	oauthID := int64(uuid.New().ID())

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		switch r.URL.Path {
		case "/login/oauth/access_token":
			_ = json.NewEncoder(w).Encode(map[string]string{
				"access_token": "mock-access-token",
				"token_type":   "bearer",
			})
		case "/user":
			_ = json.NewEncoder(w).Encode(map[string]any{
				"id":    oauthID,
				"email": address,
				"name":  "GitHub Two Factor User",
				"login": "githubtwofactor",
			})
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	t.Cleanup(server.Close)

	return server
}

func startGoogleOAuthMockServer(t *testing.T, address string) *httptest.Server {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		switch r.URL.Path {
		case "/token":
			_ = json.NewEncoder(w).Encode(map[string]string{
				"access_token": "mock-access-token",
				"token_type":   "bearer",
			})
		case "/userinfo":
			_ = json.NewEncoder(w).Encode(map[string]any{
				"id":    uuid.New().String(),
				"email": address,
				"name":  "Google Two Factor User",
			})
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	t.Cleanup(server.Close)

	return server
}

func Test_SignIn_WhenTheAccountIsTheBootstrapAdministrator_StillAsksForACode(t *testing.T) {
	router, mailSender := createTwoFactorTestRouter(t)
	users_testing.DeleteAllUsers()

	// The first account on an empty instance administers it, and the second
	// factor covers it like any other account.
	account := registerTwoFactorTestAccount(t, router)

	rootAdmin, err := (&users_repositories.UserRepository{}).GetRootAdmin(t.Context())
	require.NoError(t, err)
	require.NotNil(t, rootAdmin)
	require.Equal(t, account, rootAdmin.Email)

	pendingSignIn := submitPassword(t, router, account, http.StatusOK)

	assert.NotEqual(t, uuid.Nil, pendingSignIn.PendingSignInID)
	require.Len(t, mailSender.SentEmails, 1)
}

func Test_TurningTheSecondFactorOn_LeavesSessionsThatAlreadyExistWorking(t *testing.T) {
	router, _ := createTwoFactorTestRouter(t)
	users_testing.DisableTwoFactorAuth(t.Context())

	account := registerTwoFactorTestAccount(t, router)

	var signIn users_dto.SignInResponseDTO
	test_utils.MakePostRequestAndUnmarshal(
		t,
		router,
		"/api/v1/users/signin",
		"",
		users_dto.SignInRequestDTO{Email: account, Password: twoFactorTestPassword},
		http.StatusOK,
		&signIn,
	)
	require.NotEmpty(t, signIn.Token)

	users_testing.EnableTwoFactorAuth(t.Context())

	test_utils.MakeGetRequest(t, router, "/api/v1/users/me", "Bearer "+signIn.Token, http.StatusOK)
}

func Test_VerifyAndResend_WithAnAddressInsteadOfTheIdentifier_AreRefused(t *testing.T) {
	router, mailSender := createTwoFactorTestRouter(t)
	account := registerTwoFactorTestAccount(t, router)

	submitPassword(t, router, account, http.StatusOK)
	messagesBefore := len(mailSender.SentEmails)

	// Neither request has a field to put an address in, so a caller who knows
	// one cannot name the account with it.
	for _, url := range []string{"/api/v1/users/verify-signin-code", "/api/v1/users/resend-signin-code"} {
		test_utils.MakeRequest(t, router, test_utils.RequestOptions{
			Method:         "POST",
			URL:            url,
			Body:           map[string]any{"email": account, "code": "123456"},
			ExpectedStatus: http.StatusBadRequest,
		})
	}

	assert.Len(t, mailSender.SentEmails, messagesBefore)
}
