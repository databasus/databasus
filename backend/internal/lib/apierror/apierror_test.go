package apierror_test

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/getkin/kin-openapi/openapi3"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	contract "databasus-backend/api"
	"databasus-backend/internal/lib/apierror"
)

// gin's mode is process-wide, so it is set once before the parallel tests start.
func TestMain(m *testing.M) {
	gin.SetMode(gin.TestMode)
	m.Run()
}

func TestWrite(t *testing.T) {
	t.Parallel()

	t.Run("answers the problem with its status, title and code when a handler writes it", func(t *testing.T) {
		t.Parallel()
		router := gin.New()
		router.GET("/api/v1/items", func(ctx *gin.Context) {
			apierror.Write(ctx, &apierror.Problem{
				Code:   apierror.CodeInvalidRequest,
				Detail: "name is required",
				Errors: []apierror.FieldError{{Field: "name", Detail: "is required"}},
			})
		})

		response := serve(t, router, "/api/v1/items")

		require.Equal(t, http.StatusBadRequest, response.Code)
		assert.Equal(t, "application/problem+json", response.Header().Get("Content-Type"))

		writtenProblem := decodeProblem(t, response)
		assert.Equal(t, "urn:databasus:problem:invalid_request", writtenProblem.Type)
		assert.Equal(t, "Invalid request", writtenProblem.Title)
		assert.Equal(t, http.StatusBadRequest, writtenProblem.Status)
		assert.Equal(t, "name is required", writtenProblem.Detail)
		assert.Equal(t, "invalid_request", writtenProblem.Code)
		assert.Equal(t, []fieldErrorBody{{Field: "name", Detail: "is required"}}, writtenProblem.Errors)
	})
}

func TestCode(t *testing.T) {
	t.Parallel()

	t.Run("equals the ProblemCode enum of the contract", func(t *testing.T) {
		t.Parallel()
		goCodes := []apierror.Code{apierror.CodeInvalidRequest, apierror.CodeNotFound, apierror.CodeInternal}

		loadedContract, err := openapi3.NewLoader().LoadFromData(contract.OpenAPI)
		require.NoError(t, err)

		problemCodeSchema := loadedContract.Components.Schemas["ProblemCode"]
		require.NotNil(t, problemCodeSchema)

		contractCodes := make([]string, 0, len(problemCodeSchema.Value.Enum))
		for _, contractCode := range problemCodeSchema.Value.Enum {
			contractCodes = append(contractCodes, fmt.Sprint(contractCode))
		}

		declaredCodes := make([]string, 0, len(goCodes))
		for _, goCode := range goCodes {
			declaredCodes = append(declaredCodes, string(goCode))
		}

		assert.ElementsMatch(t, contractCodes, declaredCodes)
	})
}

type problemBody struct {
	Type   string           `json:"type"`
	Title  string           `json:"title"`
	Status int              `json:"status"`
	Detail string           `json:"detail"`
	Code   string           `json:"code"`
	Errors []fieldErrorBody `json:"errors"`
}

type fieldErrorBody struct {
	Field  string `json:"field"`
	Detail string `json:"detail"`
}

func serve(t *testing.T, router *gin.Engine, path string) *httptest.ResponseRecorder {
	t.Helper()

	return serveMethod(t, router, http.MethodGet, path)
}

func serveMethod(t *testing.T, router *gin.Engine, method, path string) *httptest.ResponseRecorder {
	t.Helper()

	response := httptest.NewRecorder()
	router.ServeHTTP(response, httptest.NewRequestWithContext(t.Context(), method, path, nil))

	return response
}

func requireProblem(
	t *testing.T,
	response *httptest.ResponseRecorder,
	expectedStatus int,
	expectedCode string,
) problemBody {
	t.Helper()

	require.Equal(t, expectedStatus, response.Code)
	require.Equal(t, "application/problem+json", response.Header().Get("Content-Type"))

	writtenProblem := decodeProblem(t, response)
	assert.Equal(t, expectedStatus, writtenProblem.Status)
	assert.Equal(t, expectedCode, writtenProblem.Code)

	return writtenProblem
}

func decodeProblem(t *testing.T, response *httptest.ResponseRecorder) problemBody {
	t.Helper()

	var writtenProblem problemBody
	require.NoError(t, json.Unmarshal(response.Body.Bytes(), &writtenProblem))

	return writtenProblem
}
