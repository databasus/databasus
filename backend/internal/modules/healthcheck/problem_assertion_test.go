package healthcheck_test

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	apiclient "databasus-backend/internal/lib/apiclient"
)

type problemCheck struct {
	Status     int
	Body       []byte
	HTTPStatus int
	Code       apiclient.ProblemCode
}

func requireProblem(t *testing.T, check problemCheck) {
	t.Helper()
	require.Equal(t, check.HTTPStatus, check.Status, string(check.Body))
	var problem apiclient.Problem
	require.NoError(t, json.Unmarshal(check.Body, &problem))
	assert.Equal(t, check.Code, problem.Code)
}
