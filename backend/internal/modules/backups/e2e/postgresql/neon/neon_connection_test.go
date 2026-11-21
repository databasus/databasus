// Package neon checks backups and restores against a real managed PostgreSQL service, Neon,
// through the HTTP API. The same cycle also runs against a container, so its own setup, comparison
// and cleanup are verified without Neon.
package neon

import (
	"fmt"
	"os"
	"strconv"
	"testing"

	"github.com/stretchr/testify/assert"
)

const (
	neonConnectionURLVariable = "TEST_NEON_DATABASE_URL"
	isNeonRequiredVariable    = "TEST_IS_NEON_REQUIRED"
)

type neonRunAction int

const (
	neonRunActionRun neonRunAction = iota
	neonRunActionSkip
	neonRunActionFail
)

type neonRunDecision struct {
	action        neonRunAction
	connectionURL string
	reason        string
}

// requireNeonConnectionURL returns the Neon connection string, skips the test without one, and
// fails it instead where the string is required, so CI cannot pass without checking Neon.
func requireNeonConnectionURL(t *testing.T) string {
	t.Helper()

	//nolint:forbidigo // The test runner or CI supplies the Neon project, outside the application's configuration.
	decision := decideNeonRun(os.Getenv(neonConnectionURLVariable), os.Getenv(isNeonRequiredVariable))

	switch decision.action {
	case neonRunActionRun:
		return decision.connectionURL
	case neonRunActionSkip:
		t.Skip(decision.reason)
	case neonRunActionFail:
		t.Fatal(decision.reason)
	}

	return ""
}

func decideNeonRun(connectionURL, isNeonRequiredValue string) neonRunDecision {
	isNeonRequired := false

	if isNeonRequiredValue != "" {
		parsedIsNeonRequired, err := strconv.ParseBool(isNeonRequiredValue)
		if err != nil {
			return neonRunDecision{
				action:        neonRunActionFail,
				connectionURL: "",
				reason: fmt.Sprintf("%s must be true or false, not %q",
					isNeonRequiredVariable, isNeonRequiredValue),
			}
		}

		isNeonRequired = parsedIsNeonRequired
	}

	if connectionURL != "" {
		return neonRunDecision{action: neonRunActionRun, connectionURL: connectionURL, reason: ""}
	}

	if isNeonRequired {
		return neonRunDecision{
			action:        neonRunActionFail,
			connectionURL: "",
			reason: fmt.Sprintf(
				"%s is empty while %s=true: set it to the connection string of the dedicated Neon test project",
				neonConnectionURLVariable,
				isNeonRequiredVariable,
			),
		}
	}

	return neonRunDecision{
		action:        neonRunActionSkip,
		connectionURL: "",
		reason: fmt.Sprintf(
			"%s is empty: set it to the connection string of a dedicated Neon test project to run this test",
			neonConnectionURLVariable,
		),
	}
}

func TestDecideNeonRun(t *testing.T) {
	t.Parallel()

	const connectionURL = "postgresql://owner:secret@ep-young-dew-123456.c-8.us-east-2.aws.neon.tech/neondb"

	cases := []struct {
		name                string
		connectionURL       string
		isNeonRequiredValue string
		expectedAction      neonRunAction
		expectedReasonPart  string
	}{
		{"runs when the connection string is set", connectionURL, "", neonRunActionRun, ""},
		{"runs when the connection string is set and required", connectionURL, "true", neonRunActionRun, ""},
		{
			"skips naming the variable without the connection string",
			"",
			"",
			neonRunActionSkip,
			neonConnectionURLVariable,
		},
		{
			"skips without the connection string when it is not required", "", "false", neonRunActionSkip,
			neonConnectionURLVariable,
		},
		{
			"fails naming the variable without the connection string when it is required", "", "true",
			neonRunActionFail, neonConnectionURLVariable,
		},
		{
			"fails naming the flag when the flag is not a boolean", connectionURL, "yes please", neonRunActionFail,
			isNeonRequiredVariable,
		},
	}

	for _, testCase := range cases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			decision := decideNeonRun(testCase.connectionURL, testCase.isNeonRequiredValue)

			assert.Equal(t, testCase.expectedAction, decision.action)
			assert.Contains(t, decision.reason, testCase.expectedReasonPart)

			if testCase.expectedAction == neonRunActionRun {
				assert.Equal(t, testCase.connectionURL, decision.connectionURL)
			}
		})
	}
}
