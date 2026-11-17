package e2e

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"mime/multipart"
	"net/http"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"

	apiclient "databasus-backend/internal/lib/apiclient"
	"databasus-backend/internal/lib/testwait"
)

// publicationLimit bounds an accepted file import: draining the application, swapping the
// databases and starting the recovered application again.
const publicationLimit = 5 * time.Minute

// uploadArchive imports the archive file with the source master key typed into the text field.
func uploadArchive(t *testing.T, root accountSession, archive, masterKey []byte) apiclient.InstanceRecovery {
	t.Helper()

	var body bytes.Buffer

	writer := multipart.NewWriter(&body)
	require.NoError(t, writer.WriteField("masterKey", string(masterKey)))
	archivePart, err := writer.CreateFormFile("archive", "databasus.dbbackup")
	require.NoError(t, err)
	_, err = archivePart.Write(archive)
	require.NoError(t, err)
	require.NoError(t, writer.Close())

	response, err := root.Client.CreateInstanceRecoveryWithBodyWithResponse(t.Context(),
		writer.FormDataContentType(), &body)
	require.NoError(t, err)
	require.Equal(t, http.StatusAccepted, response.StatusCode(), string(response.Body))

	return *response.JSON202
}

// waitForPreparation waits until the uploaded archive was validated and imported into staging,
// or refused.
func waitForPreparation(t *testing.T, root accountSession, recoveryID uuid.UUID) apiclient.InstanceRecovery {
	t.Helper()

	var preparedRecovery apiclient.InstanceRecovery

	testwait.Within(publicationLimit).Until(t, "the uploaded archive is prepared", func(ctx context.Context) error {
		response, err := root.Client.GetInstanceRecoveryWithResponse(ctx, recoveryID)
		if err != nil {
			return err
		}

		if response.StatusCode() != http.StatusOK {
			return fmt.Errorf("get the recovery answered %d", response.StatusCode())
		}

		if response.JSON200.Status == apiclient.InstanceRecoveryStatusVALIDATING {
			return fmt.Errorf("the recovery %s is still validating", recoveryID)
		}

		preparedRecovery = *response.JSON200

		return nil
	})

	return preparedRecovery
}

func prepareImport(t *testing.T, root accountSession, archive, masterKey []byte) apiclient.InstanceRecovery {
	t.Helper()

	preparedRecovery := waitForPreparation(t, root, uploadArchive(t, root, archive, masterKey).Id)
	require.Equal(t, apiclient.InstanceRecoveryStatusREADY, preparedRecovery.Status, "%v", preparedRecovery.Failure)

	return preparedRecovery
}

// applyImport confirms the replacement with root admin's password. It returns nil when the answer
// was lost because the process stopped while it answered.
func applyImport(t *testing.T, root accountSession, recoveryID uuid.UUID) *apiclient.ApplyInstanceRecoveryResponse {
	t.Helper()

	response, err := root.Client.ApplyInstanceRecoveryWithResponse(t.Context(), recoveryID,
		apiclient.InstanceRecoveryApply{
			IsReplacementConfirmed: true,
			Proof:                  apiclient.IdentityProof{Password: new(accountPassword)},
		})
	if err != nil {
		t.Logf("the apply's answer was lost: %v", err)

		return nil
	}

	return response
}

// waitForRestoredSignIn waits until the same container serves the recovered installation: the
// replaced installation's session ended and the snapshot's account signs in.
func waitForRestoredSignIn(t *testing.T, instance runningInstance, replacedToken, email string) accountSession {
	t.Helper()

	testwait.Within(publicationLimit).Until(t, "the recovered installation accepts its own account",
		func(ctx context.Context) error {
			instance.requireRunning(t)

			if err := checkSessionEnded(ctx, instance.getURL(t), replacedToken); err != nil {
				return err
			}

			return checkSignIn(ctx, instance.getURL(t), email)
		})

	return signIn(t, instance, email)
}

// checkSignIn signs in over plain HTTP, because an application that is stopping for publication
// answers outside the contract.
func checkSignIn(ctx context.Context, baseURL, email string) error {
	body, err := json.Marshal(apiclient.SignInRequest{
		Email:                    email,
		Password:                 new(accountPassword),
		CloudflareTurnstileToken: nil,
	})
	if err != nil {
		return err
	}

	request, err := http.NewRequestWithContext(ctx, http.MethodPost, baseURL+"/api/v1/sessions", bytes.NewReader(body))
	if err != nil {
		return err
	}

	request.Header.Set("Content-Type", "application/json")

	response, err := http.DefaultClient.Do(request)
	if err != nil {
		return err
	}
	defer func() { _ = response.Body.Close() }()

	if response.StatusCode != http.StatusOK {
		return fmt.Errorf("sign-in of %s answered %d", email, response.StatusCode)
	}

	return nil
}

func checkSessionEnded(ctx context.Context, baseURL, token string) error {
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, baseURL+"/api/v1/instance-backup-statuses", nil)
	if err != nil {
		return err
	}

	request.Header.Set("Authorization", "Bearer "+token)

	response, err := http.DefaultClient.Do(request)
	if err != nil {
		return err
	}
	defer func() { _ = response.Body.Close() }()

	if response.StatusCode != http.StatusUnauthorized {
		return fmt.Errorf("the replaced session still answers %d", response.StatusCode)
	}

	return nil
}
