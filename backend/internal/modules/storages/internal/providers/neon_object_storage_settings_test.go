package providers_test

import (
	"fmt"
	"maps"
	"os"
	"strconv"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

const (
	neonObjectStorageEndpointVariable   = "TEST_NEON_S3_ENDPOINT"
	neonObjectStorageRegionVariable     = "TEST_NEON_S3_REGION"
	neonObjectStorageBucketVariable     = "TEST_NEON_S3_BUCKET"
	neonObjectStorageAccessKeyVariable  = "TEST_NEON_S3_ACCESS_KEY"
	neonObjectStorageSecretKeyVariable  = "TEST_NEON_S3_SECRET_KEY"
	isNeonObjectStorageRequiredVariable = "TEST_IS_NEON_S3_REQUIRED"
)

// neonObjectStorageSettings name a dedicated bucket of a Neon project that holds no production data.
type neonObjectStorageSettings struct {
	endpoint  string
	region    string
	bucket    string
	accessKey string
	secretKey string
}

type neonObjectStorageRunAction int

const (
	neonObjectStorageRunActionRun neonObjectStorageRunAction = iota
	neonObjectStorageRunActionSkip
	neonObjectStorageRunActionFail
)

type neonObjectStorageRunDecision struct {
	action   neonObjectStorageRunAction
	settings neonObjectStorageSettings
	reason   string
}

// requireNeonObjectStorageSettings returns the Neon bucket settings, skips the test without them,
// and fails it instead where they are required or only partly set, so a mistyped variable name never
// turns the test off silently and CI cannot pass without checking Neon.
func requireNeonObjectStorageSettings(t *testing.T) neonObjectStorageSettings {
	t.Helper()

	//nolint:forbidigo // The test runner or CI supplies the Neon bucket, outside the application's configuration.
	decision := decideNeonObjectStorageRun(os.Getenv)

	switch decision.action {
	case neonObjectStorageRunActionRun:
		return decision.settings
	case neonObjectStorageRunActionSkip:
		t.Skip(decision.reason)
	case neonObjectStorageRunActionFail:
		t.Fatal(decision.reason)
	}

	return neonObjectStorageSettings{}
}

func decideNeonObjectStorageRun(getVariable func(name string) string) neonObjectStorageRunDecision {
	isRequiredValue := getVariable(isNeonObjectStorageRequiredVariable)
	isRequired := false

	if isRequiredValue != "" {
		parsedIsRequired, err := strconv.ParseBool(isRequiredValue)
		if err != nil {
			return failNeonObjectStorageRun(fmt.Sprintf("%s must be true or false, not %q",
				isNeonObjectStorageRequiredVariable, isRequiredValue))
		}

		isRequired = parsedIsRequired
	}

	settings := neonObjectStorageSettings{
		endpoint:  getVariable(neonObjectStorageEndpointVariable),
		region:    getVariable(neonObjectStorageRegionVariable),
		bucket:    getVariable(neonObjectStorageBucketVariable),
		accessKey: getVariable(neonObjectStorageAccessKeyVariable),
		secretKey: getVariable(neonObjectStorageSecretKeyVariable),
	}

	missingVariables := getMissingNeonObjectStorageVariables(settings)

	if len(missingVariables) == 0 {
		return neonObjectStorageRunDecision{action: neonObjectStorageRunActionRun, settings: settings, reason: ""}
	}

	if len(missingVariables) < len(getNeonObjectStorageVariables(settings)) {
		return failNeonObjectStorageRun(fmt.Sprintf(
			"Neon Object Storage settings are only partly set, missing: %s", strings.Join(missingVariables, ", "),
		))
	}

	if isRequired {
		return failNeonObjectStorageRun(fmt.Sprintf(
			"%s are empty while %s=true: set them to a dedicated bucket of the Neon test project",
			strings.Join(missingVariables, ", "), isNeonObjectStorageRequiredVariable,
		))
	}

	return neonObjectStorageRunDecision{
		action:   neonObjectStorageRunActionSkip,
		settings: neonObjectStorageSettings{},
		reason: fmt.Sprintf("%s are empty: set them to a dedicated bucket of a Neon test project to run this test",
			strings.Join(missingVariables, ", ")),
	}
}

func failNeonObjectStorageRun(reason string) neonObjectStorageRunDecision {
	return neonObjectStorageRunDecision{
		action:   neonObjectStorageRunActionFail,
		settings: neonObjectStorageSettings{},
		reason:   reason,
	}
}

type namedSetting struct {
	variable string
	value    string
}

func getNeonObjectStorageVariables(settings neonObjectStorageSettings) []namedSetting {
	return []namedSetting{
		{neonObjectStorageEndpointVariable, settings.endpoint},
		{neonObjectStorageRegionVariable, settings.region},
		{neonObjectStorageBucketVariable, settings.bucket},
		{neonObjectStorageAccessKeyVariable, settings.accessKey},
		{neonObjectStorageSecretKeyVariable, settings.secretKey},
	}
}

func getMissingNeonObjectStorageVariables(settings neonObjectStorageSettings) []string {
	var missingVariables []string

	for _, setting := range getNeonObjectStorageVariables(settings) {
		if setting.value == "" {
			missingVariables = append(missingVariables, setting.variable)
		}
	}

	return missingVariables
}

func TestDecideNeonObjectStorageRun(t *testing.T) {
	t.Parallel()

	completeVariables := map[string]string{
		neonObjectStorageEndpointVariable:  "https://br-example.storage.c-9.us-east-2.aws.neon.tech",
		neonObjectStorageRegionVariable:    "us-east-2",
		neonObjectStorageBucketVariable:    "databasus-test",
		neonObjectStorageAccessKeyVariable: "nak_live_example",
		neonObjectStorageSecretKeyVariable: "nsk_live_example",
	}

	withVariables := func(base, changes map[string]string) func(name string) string {
		variables := maps.Clone(base)
		if variables == nil {
			variables = map[string]string{}
		}

		maps.Copy(variables, changes)

		return func(name string) string { return variables[name] }
	}

	cases := []struct {
		name               string
		getVariable        func(name string) string
		expectedAction     neonObjectStorageRunAction
		expectedReasonPart string
	}{
		{
			"runs when every setting is set",
			withVariables(completeVariables, nil),
			neonObjectStorageRunActionRun, "",
		},
		{
			"runs when every setting is set and required",
			withVariables(completeVariables, map[string]string{isNeonObjectStorageRequiredVariable: "true"}),
			neonObjectStorageRunActionRun, "",
		},
		{
			"skips naming the variables without any setting",
			withVariables(nil, nil),
			neonObjectStorageRunActionSkip, neonObjectStorageSecretKeyVariable,
		},
		{
			"skips without any setting when they are not required",
			withVariables(nil, map[string]string{isNeonObjectStorageRequiredVariable: "false"}),
			neonObjectStorageRunActionSkip, neonObjectStorageEndpointVariable,
		},
		{
			"fails naming the variables without any setting when they are required",
			withVariables(nil, map[string]string{isNeonObjectStorageRequiredVariable: "true"}),
			neonObjectStorageRunActionFail, neonObjectStorageBucketVariable,
		},
		{
			"fails naming the missing variable when settings are partly set",
			withVariables(completeVariables, map[string]string{neonObjectStorageSecretKeyVariable: ""}),
			neonObjectStorageRunActionFail, neonObjectStorageSecretKeyVariable,
		},
		{
			"fails naming the flag when the flag is not a boolean",
			withVariables(completeVariables, map[string]string{isNeonObjectStorageRequiredVariable: "yes"}),
			neonObjectStorageRunActionFail, isNeonObjectStorageRequiredVariable,
		},
	}

	for _, testCase := range cases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			decision := decideNeonObjectStorageRun(testCase.getVariable)

			assert.Equal(t, testCase.expectedAction, decision.action)
			assert.Contains(t, decision.reason, testCase.expectedReasonPart)

			if testCase.expectedAction == neonObjectStorageRunActionRun {
				assert.Equal(t, completeVariables[neonObjectStorageSecretKeyVariable], decision.settings.secretKey)
			}
		})
	}
}
