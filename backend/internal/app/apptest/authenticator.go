package apptest

import (
	"testing"
	"time"

	apiclient "databasus-backend/internal/lib/apiclient"
	"databasus-backend/internal/lib/totp"
)

const authenticatorStep = 30 * time.Second

type ConnectedAuthenticatorApp struct {
	Secret        string
	RecoveryCodes []string
}

func (a *App) ConnectAuthenticatorApp(tb testing.TB, session Session) ConnectedAuthenticatorApp {
	tb.Helper()

	startedSetup, err := session.Client.StartAuthenticatorAppSetupWithResponse(
		tb.Context(), session.UserID, apiclient.SecondFactorProof{Code: nil},
	)
	if err != nil {
		tb.Fatalf("start an authenticator setup: %v", err)
	}

	if startedSetup.JSON201 == nil {
		tb.Fatalf("start an authenticator setup answered %d: %s", startedSetup.StatusCode(), startedSetup.Body)
	}

	code := a.GenerateAuthenticatorCode(tb, startedSetup.JSON201.Secret)

	confirmedSetup, err := session.Client.ConfirmAuthenticatorAppSetupWithResponse(
		tb.Context(), session.UserID, startedSetup.JSON201.AuthenticatorAppSetupId,
		apiclient.AuthenticatorAppCode{Code: &code},
	)
	if err != nil {
		tb.Fatalf("confirm the authenticator setup: %v", err)
	}

	if confirmedSetup.JSON200 == nil {
		tb.Fatalf("confirm the authenticator setup answered %d: %s", confirmedSetup.StatusCode(), confirmedSetup.Body)
	}

	return ConnectedAuthenticatorApp{
		Secret:        startedSetup.JSON201.Secret,
		RecoveryCodes: confirmedSetup.JSON200.RecoveryCodes,
	}
}

// GenerateAuthenticatorCode moves the clock to the next 30-second period and returns the code the
// secret generates for it, so every call yields a code the instance has not seen.
func (a *App) GenerateAuthenticatorCode(tb testing.TB, secret string) string {
	tb.Helper()

	if a.Clock == nil {
		tb.Fatal("authenticator codes need the manual clock")
	}

	a.Clock.Advance(authenticatorStep)

	return a.ComputeAuthenticatorCode(tb, secret, 0)
}

func (a *App) ComputeAuthenticatorCode(tb testing.TB, secret string, stepOffset int64) string {
	tb.Helper()

	code, err := totp.ComputeCode(secret, totp.GetStep(a.Clock.Now())+stepOffset)
	if err != nil {
		tb.Fatalf("compute an authenticator code: %v", err)
	}

	return code
}
