package users_test

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/stretchr/testify/require"

	"databasus-backend/internal/app/apptest"
	apiclient "databasus-backend/internal/lib/apiclient"
	"databasus-backend/internal/lib/crypto"
)

func TestSessionToken(t *testing.T) {
	t.Parallel()

	for _, scenario := range []struct {
		name          string
		subject       any
		isSubjectKept bool
		isTampered    bool
		isWrongMethod bool
		status        int
	}{
		{name: "accepts a pre-migration token when signed with HS256", isSubjectKept: true, status: http.StatusOK},
		{name: "returns 401 when the signature is tampered", isSubjectKept: true, isTampered: true, status: http.StatusUnauthorized},
		{name: "returns 401 when a valid signature uses a disallowed algorithm", isSubjectKept: true, isWrongMethod: true, status: http.StatusUnauthorized},
		{name: "returns 401 when the subject is absent", status: http.StatusUnauthorized},
		{name: "returns 401 when the subject is numeric", subject: 123, status: http.StatusUnauthorized},
		{name: "returns 401 when the subject is not a user ID", subject: "invalid", status: http.StatusUnauthorized},
	} {
		t.Run(scenario.name, func(t *testing.T) {
			t.Parallel()

			app := apptest.New(t)
			admin := app.CreateAdmin(t)
			var passwordVersion string
			app.QuerySQL(t, &passwordVersion, "SELECT password_version FROM users WHERE id = ?", admin.UserID)
			issuedAt := app.Clock.Now()
			claims := map[string]any{
				"sub":             admin.UserID.String(),
				"exp":             issuedAt.Add(10 * 365 * 24 * time.Hour).Unix(),
				"iat":             issuedAt.Unix(),
				"role":            "ADMIN",
				"passwordVersion": passwordVersion,
			}

			if !scenario.isSubjectKept {
				delete(claims, "sub")

				if scenario.subject != nil {
					claims["sub"] = scenario.subject
				}
			}

			masterKey, err := crypto.OpenOrCreateMasterKey(app.GetServerConfig().SecretKeyPath)
			require.NoError(t, err)

			token := signPreMigrationSessionToken(t, claims, masterKey.GetSecret())

			if scenario.isTampered {
				signatureStart := strings.LastIndexByte(token, '.') + 1
				replacement := "A"
				if token[signatureStart] == 'A' {
					replacement = "B"
				}

				token = token[:signatureStart] + replacement + token[signatureStart+1:]
			}

			if scenario.isWrongMethod {
				token, err = jwt.NewWithClaims(jwt.SigningMethodHS384, jwt.MapClaims(claims)).
					SignedString([]byte(masterKey.GetSecret()))
				require.NoError(t, err)
			}

			response := getUser(t, app.NewSessionClient(t, token), admin.UserID)

			require.Equal(t, scenario.status, response.StatusCode(), string(response.Body))

			if scenario.status == http.StatusUnauthorized {
				requireProblem(
					t,
					response.StatusCode(),
					response.Body,
					scenario.status,
					apiclient.ProblemCodeUnauthenticated,
				)
			}
		})
	}
}

func TestSessionTokenExpiry(t *testing.T) {
	t.Parallel()

	for _, scenario := range []struct {
		name   string
		offset time.Duration
		status int
	}{
		{name: "accepts a token when the injected clock is just before expiry", offset: -time.Nanosecond, status: http.StatusOK},
		{name: "returns 401 when the injected clock reaches expiry", offset: 0, status: http.StatusUnauthorized},
		{name: "returns 401 when the injected clock passes expiry", offset: time.Nanosecond, status: http.StatusUnauthorized},
	} {
		t.Run(scenario.name, func(t *testing.T) {
			t.Parallel()

			app := apptest.New(t)
			admin := app.CreateAdmin(t)
			claims := readSessionTokenClaims(t, admin.Token)
			expiresAt, isNumber := claims["exp"].(float64)
			require.True(t, isNumber)

			app.Clock.Set(time.Unix(int64(expiresAt), 0).Add(scenario.offset))

			response := getUser(t, admin.Client, admin.UserID)

			require.Equal(t, scenario.status, response.StatusCode(), string(response.Body))

			if scenario.status == http.StatusUnauthorized {
				requireProblem(
					t,
					response.StatusCode(),
					response.Body,
					scenario.status,
					apiclient.ProblemCodeUnauthenticated,
				)
			}
		})
	}
}

func readSessionTokenClaims(t *testing.T, token string) map[string]any {
	t.Helper()
	parts := strings.Split(token, ".")
	require.Len(t, parts, 3)

	payload, err := base64.RawURLEncoding.DecodeString(parts[1])
	require.NoError(t, err)

	var claims map[string]any
	require.NoError(t, json.Unmarshal(payload, &claims))

	return claims
}

func signPreMigrationSessionToken(t *testing.T, claims map[string]any, secret string) string {
	t.Helper()
	payload, err := json.Marshal(claims)
	require.NoError(t, err)
	unsignedToken := base64.RawURLEncoding.EncodeToString(
		[]byte(`{"alg":"HS256","typ":"JWT"}`),
	) + "." + base64.RawURLEncoding.EncodeToString(
		payload,
	)

	signature := hmac.New(sha256.New, []byte(secret))
	_, err = signature.Write([]byte(unsignedToken))
	require.NoError(t, err)

	return unsignedToken + "." + base64.RawURLEncoding.EncodeToString(signature.Sum(nil))
}
