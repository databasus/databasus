package instancebackups_test

import (
	"net/http"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"databasus-backend/internal/app/apptest"
	apiclient "databasus-backend/internal/lib/apiclient"
)

func TestApplyInstanceRecovery(t *testing.T) {
	t.Parallel()

	t.Run("accepts the apply without a code while two-factor sign-in is off and repeats it", func(t *testing.T) {
		t.Parallel()
		owner := newInstanceOwner(t)
		prepared := owner.App.PrepareRecovery(t)

		firstResponse := applyRecovery(t, owner.RootAdmin, prepared.ID, nil)
		repeatedResponse := applyRecovery(t, owner.RootAdmin, prepared.ID, nil)

		require.Equal(t, http.StatusAccepted, firstResponse.StatusCode(), string(firstResponse.Body))
		require.Equal(t, http.StatusAccepted, repeatedResponse.StatusCode(), string(repeatedResponse.Body))
		assert.Equal(t, apiclient.InstanceRecoveryStatusAPPLYING, firstResponse.JSON202.Status)
		assert.Equal(t, *firstResponse.JSON202, *repeatedResponse.JSON202)
		assert.Equal(t, apiclient.InstanceRecoveryStatusAPPLYING, getRecovery(t, owner.RootAdmin, prepared.ID).Status)
	})

	t.Run("accepts the apply without a code and answers 409 to a confirmation request when nothing is "+
		"required although root admin has an app", func(t *testing.T) {
		t.Parallel()
		owner := newInstanceOwner(t)
		owner.App.ConnectAuthenticatorApp(t, owner.RootAdmin)
		prepared := owner.App.PrepareRecovery(t)

		confirmationResponse := requestRecoveryConfirmation(t, owner.RootAdmin, prepared.ID)
		applyResponse := applyRecovery(t, owner.RootAdmin, prepared.ID, nil)

		requireProblemCode(t, confirmationResponse.StatusCode(), confirmationResponse.Body,
			http.StatusConflict, apiclient.ProblemCodeIdentityProofUnavailable)
		assert.Empty(t, owner.App.Mail.GetMessagesTo(owner.RootAdmin.Email))
		require.Equal(t, http.StatusAccepted, applyResponse.StatusCode(), string(applyResponse.Body))
		assert.Equal(t, apiclient.InstanceRecoveryStatusAPPLYING, applyResponse.JSON202.Status)
	})

	t.Run("accepts the apply with a code emailed for this recovery", func(t *testing.T) {
		t.Parallel()
		owner := newTwoFactorInstanceOwner(t)
		prepared := owner.App.PrepareRecovery(t)

		code := requireRecoveryCode(t, owner, prepared.ID)
		response := applyRecovery(t, owner.RootAdmin, prepared.ID, new(code))

		require.Equal(t, http.StatusAccepted, response.StatusCode(), string(response.Body))
		assert.Equal(t, apiclient.InstanceRecoveryStatusAPPLYING, response.JSON202.Status)
	})

	t.Run(
		"accepts the apply without a code once two-factor sign-in is turned off after a code was sent",
		func(t *testing.T) {
			t.Parallel()
			owner := newTwoFactorInstanceOwner(t)
			prepared := owner.App.PrepareRecovery(t)
			requireRecoveryCode(t, owner, prepared.ID)
			setTwoFactorRequirement(t, owner.RootAdmin, apiclient.TwoFactorRequirementOFF)

			response := applyRecovery(t, owner.RootAdmin, prepared.ID, nil)

			require.Equal(t, http.StatusAccepted, response.StatusCode(), string(response.Body))
			assert.Equal(t, apiclient.InstanceRecoveryStatusAPPLYING, response.JSON202.Status)
		},
	)

	t.Run("returns 400 identity_proof_invalid without a code while two-factor sign-in is required", func(t *testing.T) {
		t.Parallel()
		owner := newTwoFactorInstanceOwner(t)
		prepared := owner.App.PrepareRecovery(t)

		response := applyRecovery(t, owner.RootAdmin, prepared.ID, nil)

		requireProblemCode(t, response.StatusCode(), response.Body,
			http.StatusBadRequest, apiclient.ProblemCodeIdentityProofInvalid)
		assert.Equal(t, apiclient.InstanceRecoveryStatusREADY, getRecovery(t, owner.RootAdmin, prepared.ID).Status)
	})

	t.Run("returns 400 identity_proof_invalid for a master-key export code and keeps it ready", func(t *testing.T) {
		t.Parallel()
		owner := newTwoFactorInstanceOwner(t)
		prepared := owner.App.PrepareRecovery(t)
		exportCode := requireMasterKeyCode(t, owner)

		response := applyRecovery(t, owner.RootAdmin, prepared.ID, new(exportCode))

		requireProblemCode(t, response.StatusCode(), response.Body,
			http.StatusBadRequest, apiclient.ProblemCodeIdentityProofInvalid)
		assert.Equal(t, apiclient.InstanceRecoveryStatusREADY, getRecovery(t, owner.RootAdmin, prepared.ID).Status)
	})

	t.Run("returns 400 identity_proof_invalid for a code emailed for a cancelled recovery", func(t *testing.T) {
		t.Parallel()
		owner := newTwoFactorInstanceOwner(t)
		cancelledRecovery := owner.App.PrepareRecovery(t)
		cancelledRecoveryCode := requireRecoveryCode(t, owner, cancelledRecovery.ID)
		cancelResponse, err := owner.RootAdmin.Client.CancelInstanceRecoveryWithResponse(
			t.Context(), cancelledRecovery.ID,
		)
		require.NoError(t, err)
		require.Equal(t, http.StatusOK, cancelResponse.StatusCode(), string(cancelResponse.Body))
		nextRecovery := owner.App.PrepareRecovery(t)

		response := applyRecovery(t, owner.RootAdmin, nextRecovery.ID, new(cancelledRecoveryCode))

		requireProblemCode(t, response.StatusCode(), response.Body,
			http.StatusBadRequest, apiclient.ProblemCodeIdentityProofInvalid)
		assert.Equal(t, apiclient.InstanceRecoveryStatusREADY, getRecovery(t, owner.RootAdmin, nextRecovery.ID).Status)
	})

	t.Run("returns 400 identity_proof_invalid for a code requested in another session", func(t *testing.T) {
		t.Parallel()
		owner := newInstanceOwner(t)
		// A session token carries its issue time in seconds, so the second sign-in needs a later one.
		// It signs in before two-factor sign-in is required, which would ask it for an emailed code.
		owner.App.Clock.Advance(time.Second)
		otherSession := signInAgain(t, owner.App, owner.RootAdmin.Email)
		requireTwoFactorSignIn(t, owner.RootAdmin)
		prepared := owner.App.PrepareRecovery(t)
		code := requireRecoveryCode(t, owner, prepared.ID)

		response := applyRecovery(t, otherSession, prepared.ID, new(code))

		requireProblemCode(t, response.StatusCode(), response.Body,
			http.StatusBadRequest, apiclient.ProblemCodeIdentityProofInvalid)
		assert.Equal(t, apiclient.InstanceRecoveryStatusREADY, getRecovery(t, owner.RootAdmin, prepared.ID).Status)
	})

	t.Run("returns 400 identity_proof_invalid for a wrong code and keeps it ready", func(t *testing.T) {
		t.Parallel()
		owner := newTwoFactorInstanceOwner(t)
		prepared := owner.App.PrepareRecovery(t)
		code := requireRecoveryCode(t, owner, prepared.ID)

		response := applyRecovery(t, owner.RootAdmin, prepared.ID, new(getWrongCode(code)))

		requireProblemCode(t, response.StatusCode(), response.Body,
			http.StatusBadRequest, apiclient.ProblemCodeIdentityProofInvalid)
		assert.Equal(t, apiclient.InstanceRecoveryStatusREADY, getRecovery(t, owner.RootAdmin, prepared.ID).Status)
	})

	t.Run("returns 400 invalid_request without the replacement confirmation", func(t *testing.T) {
		t.Parallel()
		owner := newInstanceOwner(t)
		prepared := owner.App.PrepareRecovery(t)

		response, err := owner.RootAdmin.Client.ApplyInstanceRecoveryWithResponse(t.Context(), prepared.ID,
			apiclient.InstanceRecoveryApply{IsReplacementConfirmed: false, Code: nil})

		require.NoError(t, err)
		requireProblemCode(t, response.StatusCode(), response.Body,
			http.StatusBadRequest, apiclient.ProblemCodeInvalidRequest)
		assert.Equal(t, apiclient.InstanceRecoveryStatusREADY, getRecovery(t, owner.RootAdmin, prepared.ID).Status)
	})

	t.Run("returns 403 permission_denied to another administrator", func(t *testing.T) {
		t.Parallel()
		owner := newInstanceOwner(t)
		prepared := owner.App.PrepareRecovery(t)

		response := applyRecovery(t, owner.OtherAdmin, prepared.ID, nil)

		requireProblemCode(t, response.StatusCode(), response.Body,
			http.StatusForbidden, apiclient.ProblemCodePermissionDenied)
		assert.Equal(t, apiclient.InstanceRecoveryStatusREADY, getRecovery(t, owner.RootAdmin, prepared.ID).Status)
	})

	t.Run("returns 401 unauthenticated without a session", func(t *testing.T) {
		t.Parallel()
		owner := newInstanceOwner(t)
		prepared := owner.App.PrepareRecovery(t)

		response, err := owner.App.NewClient(t).ApplyInstanceRecoveryWithResponse(t.Context(), prepared.ID,
			apiclient.InstanceRecoveryApply{IsReplacementConfirmed: true, Code: nil})

		require.NoError(t, err)
		requireProblemCode(t, response.StatusCode(), response.Body,
			http.StatusUnauthorized, apiclient.ProblemCodeUnauthenticated)
	})

	t.Run("returns 409 recovery_not_ready for a recovery still validating", func(t *testing.T) {
		t.Parallel()
		owner := newInstanceOwner(t)
		uploadResponse := uploadRecovery(t, owner.RootAdmin, newRecoveryUpload([]byte("archive")))
		require.Equal(t, http.StatusAccepted, uploadResponse.StatusCode(), string(uploadResponse.Body))

		response := applyRecovery(t, owner.RootAdmin, uploadResponse.JSON202.Id, nil)

		requireProblemCode(t, response.StatusCode(), response.Body,
			http.StatusConflict, apiclient.ProblemCodeRecoveryNotReady)
	})

	t.Run("returns 409 recovery_not_ready once the prepared recovery expired", func(t *testing.T) {
		t.Parallel()
		owner := newInstanceOwner(t)
		prepared := owner.App.PrepareRecovery(t)
		owner.App.Clock.Advance(time.Hour)

		response := applyRecovery(t, owner.RootAdmin, prepared.ID, nil)

		requireProblemCode(t, response.StatusCode(), response.Body,
			http.StatusConflict, apiclient.ProblemCodeRecoveryNotReady)
	})

	t.Run("returns 404 not_found for a recovery the journal does not hold", func(t *testing.T) {
		t.Parallel()
		owner := newInstanceOwner(t)

		response := applyRecovery(t, owner.RootAdmin, uuid.New(), nil)

		requireProblemCode(t, response.StatusCode(), response.Body,
			http.StatusNotFound, apiclient.ProblemCodeNotFound)
	})

	t.Run("refuses to cancel a recovery whose apply was accepted", func(t *testing.T) {
		t.Parallel()
		owner := newInstanceOwner(t)
		prepared := owner.App.PrepareRecovery(t)
		applyResponse := applyRecovery(t, owner.RootAdmin, prepared.ID, nil)
		require.Equal(t, http.StatusAccepted, applyResponse.StatusCode(), string(applyResponse.Body))

		cancelResponse, err := owner.RootAdmin.Client.CancelInstanceRecoveryWithResponse(t.Context(), prepared.ID)

		require.NoError(t, err)
		requireProblemCode(t, cancelResponse.StatusCode(), cancelResponse.Body,
			http.StatusConflict, apiclient.ProblemCodeRecoveryNotReady)
	})
	t.Run("accepts the apply with root admin's app code on an instance without a mail server", func(t *testing.T) {
		t.Parallel()
		owner := newInstanceOwner(t, apptest.WithoutMailServer())
		connectedApp := owner.App.ConnectAuthenticatorApp(t, owner.RootAdmin)
		setTwoFactorRequirement(t, owner.RootAdmin, apiclient.TwoFactorRequirementAUTHENTICATORAPP)
		prepared := owner.App.PrepareRecovery(t)

		response := applyRecovery(t, owner.RootAdmin, prepared.ID,
			new(owner.App.GenerateAuthenticatorCode(t, connectedApp.Secret)))

		require.Equal(t, http.StatusAccepted, response.StatusCode(), string(response.Body))
	})

	t.Run("accepts the apply with a recovery code and spends it", func(t *testing.T) {
		t.Parallel()
		owner := newInstanceOwner(t, apptest.WithoutMailServer())
		connectedApp := owner.App.ConnectAuthenticatorApp(t, owner.RootAdmin)
		setTwoFactorRequirement(t, owner.RootAdmin, apiclient.TwoFactorRequirementAUTHENTICATORAPP)
		prepared := owner.App.PrepareRecovery(t)

		response := applyRecovery(t, owner.RootAdmin, prepared.ID, &connectedApp.RecoveryCodes[0])

		require.Equal(t, http.StatusAccepted, response.StatusCode(), string(response.Body))
		var unusedRecoveryCodeCount int
		owner.App.QuerySQL(t, &unusedRecoveryCodeCount,
			"SELECT count(*) FROM users_recovery_codes WHERE used_at IS NULL")
		assert.Equal(t, 9, unusedRecoveryCodeCount)
	})

	t.Run("returns 400 identity_proof_invalid for an app code already spent on a master-key export",
		func(t *testing.T) {
			t.Parallel()
			owner := newTwoFactorInstanceOwner(t)
			connectedApp := owner.App.ConnectAuthenticatorApp(t, owner.RootAdmin)
			prepared := owner.App.PrepareRecovery(t)
			code := owner.App.GenerateAuthenticatorCode(t, connectedApp.Secret)
			requireMasterKeyExport(t, owner.RootAdmin, &code)

			response := applyRecovery(t, owner.RootAdmin, prepared.ID, &code)

			requireProblemCode(t, response.StatusCode(), response.Body,
				http.StatusBadRequest, apiclient.ProblemCodeIdentityProofInvalid)
			assert.Equal(t, apiclient.InstanceRecoveryStatusREADY, getRecovery(t, owner.RootAdmin, prepared.ID).Status)
		})
}

func TestRequestInstanceRecoveryConfirmation(t *testing.T) {
	t.Parallel()

	t.Run("answers the app method without mail when root admin has an app", func(t *testing.T) {
		t.Parallel()
		owner := newTwoFactorInstanceOwner(t)
		owner.App.ConnectAuthenticatorApp(t, owner.RootAdmin)
		prepared := owner.App.PrepareRecovery(t)

		response := requestRecoveryConfirmation(t, owner.RootAdmin, prepared.ID)

		require.Equal(t, http.StatusCreated, response.StatusCode(), string(response.Body))
		assert.Equal(t, apiclient.IdentityConfirmationMethodAUTHENTICATORAPP, response.JSON201.Method)
		assert.Nil(t, response.JSON201.ExpiresAt)
		assert.Empty(t, owner.App.Mail.GetMessagesTo(owner.RootAdmin.Email))
	})

	t.Run("answers the app method on an instance without a mail server when root admin has an app",
		func(t *testing.T) {
			t.Parallel()
			owner := newInstanceOwner(t, apptest.WithoutMailServer())
			owner.App.ConnectAuthenticatorApp(t, owner.RootAdmin)
			setTwoFactorRequirement(t, owner.RootAdmin, apiclient.TwoFactorRequirementAUTHENTICATORAPP)
			prepared := owner.App.PrepareRecovery(t)

			response := requestRecoveryConfirmation(t, owner.RootAdmin, prepared.ID)

			require.Equal(t, http.StatusCreated, response.StatusCode(), string(response.Body))
			assert.Equal(t, apiclient.IdentityConfirmationMethodAUTHENTICATORAPP, response.JSON201.Method)
			assert.Nil(t, response.JSON201.ExpiresAt)
		})

	t.Run("emails root admin a code that expires in five minutes, one a minute", func(t *testing.T) {
		t.Parallel()
		owner := newTwoFactorInstanceOwner(t)
		prepared := owner.App.PrepareRecovery(t)

		firstResponse := requestRecoveryConfirmation(t, owner.RootAdmin, prepared.ID)
		secondResponse := requestRecoveryConfirmation(t, owner.RootAdmin, prepared.ID)

		require.Equal(t, http.StatusCreated, firstResponse.StatusCode(), string(firstResponse.Body))
		assert.Equal(t, apiclient.IdentityConfirmationMethodEMAIL, firstResponse.JSON201.Method)
		require.NotNil(t, firstResponse.JSON201.ExpiresAt)
		assert.WithinDuration(
			t,
			owner.App.Clock.Now().Add(5*time.Minute),
			*firstResponse.JSON201.ExpiresAt,
			time.Second,
		)
		assert.Len(t, owner.App.Mail.GetMessagesTo(owner.RootAdmin.Email), 1)
		requireProblemCode(t, secondResponse.StatusCode(), secondResponse.Body,
			http.StatusTooManyRequests, apiclient.ProblemCodeRateLimited)
	})

	t.Run("returns 409 identity_proof_unavailable while two-factor sign-in is off", func(t *testing.T) {
		t.Parallel()
		owner := newInstanceOwner(t)
		prepared := owner.App.PrepareRecovery(t)

		response := requestRecoveryConfirmation(t, owner.RootAdmin, prepared.ID)

		requireProblemCode(t, response.StatusCode(), response.Body,
			http.StatusConflict, apiclient.ProblemCodeIdentityProofUnavailable)
		assert.Empty(t, owner.App.Mail.GetMessagesTo(owner.RootAdmin.Email))
	})

	t.Run("returns 409 identity_proof_unavailable without a mail server", func(t *testing.T) {
		t.Parallel()
		owner := newInstanceOwner(t, apptest.WithoutMailServer())
		requireTwoFactorAfterMailServerLoss(t, owner.App)
		prepared := owner.App.PrepareRecovery(t)

		response := requestRecoveryConfirmation(t, owner.RootAdmin, prepared.ID)

		requireProblemCode(t, response.StatusCode(), response.Body,
			http.StatusConflict, apiclient.ProblemCodeIdentityProofUnavailable)
	})

	t.Run("returns 409 recovery_not_ready for a recovery still validating", func(t *testing.T) {
		t.Parallel()
		owner := newInstanceOwner(t)
		uploadResponse := uploadRecovery(t, owner.RootAdmin, newRecoveryUpload([]byte("archive")))
		require.Equal(t, http.StatusAccepted, uploadResponse.StatusCode(), string(uploadResponse.Body))

		response := requestRecoveryConfirmation(t, owner.RootAdmin, uploadResponse.JSON202.Id)

		requireProblemCode(t, response.StatusCode(), response.Body,
			http.StatusConflict, apiclient.ProblemCodeRecoveryNotReady)
		assert.Empty(t, owner.App.Mail.GetMessagesTo(owner.RootAdmin.Email))
	})

	t.Run("returns 403 permission_denied to another administrator", func(t *testing.T) {
		t.Parallel()
		owner := newTwoFactorInstanceOwner(t)
		prepared := owner.App.PrepareRecovery(t)

		response := requestRecoveryConfirmation(t, owner.OtherAdmin, prepared.ID)

		requireProblemCode(t, response.StatusCode(), response.Body,
			http.StatusForbidden, apiclient.ProblemCodePermissionDenied)
	})
}

func applyRecovery(
	t *testing.T,
	session apptest.Session,
	recoveryID uuid.UUID,
	code *string,
) *apiclient.ApplyInstanceRecoveryResponse {
	t.Helper()

	response, err := session.Client.ApplyInstanceRecoveryWithResponse(t.Context(), recoveryID,
		apiclient.InstanceRecoveryApply{IsReplacementConfirmed: true, Code: code})
	require.NoError(t, err)

	return response
}

func requestRecoveryConfirmation(
	t *testing.T,
	session apptest.Session,
	recoveryID uuid.UUID,
) *apiclient.RequestInstanceRecoveryConfirmationResponse {
	t.Helper()

	response, err := session.Client.RequestInstanceRecoveryConfirmationWithResponse(t.Context(), recoveryID)
	require.NoError(t, err)

	return response
}

func requireRecoveryCode(t *testing.T, owner instanceOwner, recoveryID uuid.UUID) string {
	t.Helper()

	response := requestRecoveryConfirmation(t, owner.RootAdmin, recoveryID)
	require.Equal(t, http.StatusCreated, response.StatusCode(), string(response.Body))

	return readMailedCode(t, owner.App, owner.RootAdmin.Email)
}

func getWrongCode(code string) string {
	if code == "000000" {
		return "111111"
	}

	return "000000"
}
