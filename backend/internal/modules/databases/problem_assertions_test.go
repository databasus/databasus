package databases_test

import (
	"encoding/json"
	"net/http"
	"testing"

	"github.com/stretchr/testify/require"

	apiclient "databasus-backend/internal/lib/apiclient"
)

func requireProblem(t *testing.T, status int, body []byte, expectedStatus int, expectedCode apiclient.ProblemCode) {
	t.Helper()

	require.Equal(t, expectedStatus, status, string(body))

	var problem apiclient.Problem
	require.NoError(t, json.Unmarshal(body, &problem), string(body))
	require.Equal(t, expectedCode, problem.Code, string(body))
}

func requireInvalidField(t *testing.T, status int, body []byte, expectedField string) {
	t.Helper()

	requireProblem(t, status, body, http.StatusBadRequest, apiclient.ProblemCodeInvalidRequest)

	var problem apiclient.Problem
	require.NoError(t, json.Unmarshal(body, &problem), string(body))
	require.NotNil(t, problem.Errors, string(body))

	fields := make([]string, 0, len(*problem.Errors))
	for _, fieldError := range *problem.Errors {
		fields = append(fields, fieldError.Field)
	}

	require.Contains(t, fields, expectedField, string(body))
}
