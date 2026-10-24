package google_drive_storage

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"reflect"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/oauth2"
	drive "google.golang.org/api/drive/v3"

	"databasus-backend/internal/util/encryption"
	test_utils "databasus-backend/internal/util/testing"
)

func Test_BuildHTTPClient_Transport_ResolvesProxyFromEnvironment(t *testing.T) {
	transport := getGoogleDriveBaseTransport(t)

	require.NotNil(t, transport.Proxy)
	assert.Equal(t,
		reflect.ValueOf(http.ProxyFromEnvironment).Pointer(),
		reflect.ValueOf(transport.Proxy).Pointer(),
	)
}

func Test_BuildHTTPClient_TransportThroughProxy_TunnelsAndRejectsUntrustedCertificate(t *testing.T) {
	exchange := test_utils.SendThroughTunnelProxy(t, getGoogleDriveBaseTransport(t), "Bearer access-token")

	test_utils.AssertProxyReceivedOnlyTunnelRequest(t, exchange)
	test_utils.AssertCertificateRejected(t, exchange)
}

func getGoogleDriveBaseTransport(t *testing.T) *http.Transport {
	t.Helper()

	client := (&GoogleDriveStorage{}).buildHTTPClient(
		oauth2.StaticTokenSource(&oauth2.Token{AccessToken: "access-token"}),
	)

	oauthTransport, isOAuthTransport := client.Transport.(*oauth2.Transport)
	require.True(t, isOAuthTransport, "Google Drive client must authenticate through oauth2.Transport")

	baseTransport, isTransport := oauthTransport.Base.(*http.Transport)
	require.True(t, isTransport, "Google Drive OAuth transport must sit on an *http.Transport")

	return baseTransport
}

const (
	staleAccessToken   = "stale-access-token-0123456789abcdef"
	storedRefreshToken = "stored-refresh-token-0123456789abcdef"
	freshAccessToken   = "fresh-access-token-0123456789abcdef"
)

type tokenRefreshOutcome struct {
	operationErr   error
	operationCalls int
	loggedOutput   string
	standardOutput string
}

func Test_WithRetryOnAuth_WhenTokenRefreshSucceeds_LogsRefreshWithoutTokens(t *testing.T) {
	tokenEndpoint := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, _ *http.Request) {
		writer.Header().Set("Content-Type", "application/json")
		_, _ = fmt.Fprintf(writer, `{"access_token":%q,"token_type":"Bearer","expires_in":3600}`, freshAccessToken)
	}))
	t.Cleanup(tokenEndpoint.Close)

	outcome := runOperationThatNeedsTokenRefresh(t, tokenEndpoint.URL)

	require.NoError(t, outcome.operationErr)
	assert.Equal(t, 2, outcome.operationCalls)
	assert.Contains(t, outcome.loggedOutput, "google drive token refreshed")
	assertNoTokenFragments(t, outcome.loggedOutput+outcome.standardOutput)
}

func Test_WithRetryOnAuth_WhenTokenRefreshFails_ReturnsRefreshErrorWithoutTokens(t *testing.T) {
	tokenEndpoint := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, _ *http.Request) {
		writer.Header().Set("Content-Type", "application/json")
		writer.WriteHeader(http.StatusInternalServerError)
		_, _ = writer.Write([]byte(`{"error":"server_error"}`))
	}))
	t.Cleanup(tokenEndpoint.Close)

	outcome := runOperationThatNeedsTokenRefresh(t, tokenEndpoint.URL)

	require.Error(t, outcome.operationErr)
	assert.Contains(t, outcome.operationErr.Error(), "failed to refresh token")
	assert.Equal(t, 1, outcome.operationCalls)
	assert.Contains(t, outcome.loggedOutput, "google drive token refresh failed")
	assertNoTokenFragments(t, outcome.operationErr.Error()+outcome.loggedOutput+outcome.standardOutput)
}

// The first run of the operation fails with an authentication error, so the storage refreshes the
// token against tokenEndpointURL and runs the operation again.
func runOperationThatNeedsTokenRefresh(t *testing.T, tokenEndpointURL string) tokenRefreshOutcome {
	t.Helper()

	storedToken, err := json.Marshal(oauth2.Token{
		AccessToken:  staleAccessToken,
		RefreshToken: storedRefreshToken,
		TokenType:    "Bearer",
		Expiry:       time.Now().UTC().Add(time.Hour),
	})
	require.NoError(t, err)

	googleDriveStorage := &GoogleDriveStorage{
		ClientID:     "client-id",
		ClientSecret: "client-secret",
		TokenJSON:    string(storedToken),
	}

	tokenEndpoint, err := url.Parse(tokenEndpointURL)
	require.NoError(t, err)

	tokenRequestCtx := context.WithValue(t.Context(), oauth2.HTTPClient, &http.Client{
		Transport: tokenEndpointRedirect{target: tokenEndpoint},
	})

	var loggedOutput bytes.Buffer
	operationLogger := slog.New(slog.NewTextHandler(&loggedOutput, nil))

	outcome := tokenRefreshOutcome{}
	outcome.standardOutput = captureStandardOutput(t, func() {
		outcome.operationErr = googleDriveStorage.withRetryOnAuth(
			tokenRequestCtx,
			operationLogger,
			encryption.GetFieldEncryptor(),
			func(*drive.Service) error {
				outcome.operationCalls++
				if outcome.operationCalls == 1 {
					return errors.New("googleapi: Error 401: Invalid Credentials, authError")
				}

				return nil
			},
		)
	})
	outcome.loggedOutput = loggedOutput.String()

	return outcome
}

type tokenEndpointRedirect struct {
	target *url.URL
}

func (r tokenEndpointRedirect) RoundTrip(request *http.Request) (*http.Response, error) {
	redirectedRequest := request.Clone(request.Context())
	redirectedRequest.URL.Scheme = r.target.Scheme
	redirectedRequest.URL.Host = r.target.Host
	redirectedRequest.Host = r.target.Host

	return http.DefaultTransport.RoundTrip(redirectedRequest)
}

func captureStandardOutput(t *testing.T, run func()) string {
	t.Helper()

	reader, writer, err := os.Pipe()
	require.NoError(t, err)

	originalStdout := os.Stdout
	os.Stdout = writer

	capturedOutput := make(chan string, 1)
	go func() {
		output, _ := io.ReadAll(reader)
		capturedOutput <- string(output)
	}()

	run()

	os.Stdout = originalStdout
	require.NoError(t, writer.Close())

	return <-capturedOutput
}

func assertNoTokenFragments(t *testing.T, output string) {
	t.Helper()

	for _, tokenFragment := range []string{"stale-access", "stored-refresh", "fresh-access", "0123456789"} {
		assert.NotContains(t, output, tokenFragment)
	}
}
