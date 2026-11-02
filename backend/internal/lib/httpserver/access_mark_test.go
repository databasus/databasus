package httpserver

import (
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const markedContract = `
openapi: 3.0.3
info:
  title: Marked contract
  version: 1.0.0
servers:
  - url: /api/v1
paths:
  /sessions:
    post:
      operationId: signIn
      x-access: public
      responses:
        "204":
          description: Signed in
  /users:
    get:
      operationId: listUsers
      x-access: admin
      responses:
        "204":
          description: The users
  /workspaces:
    get:
      operationId: listWorkspaces
      responses:
        "204":
          description: The workspaces
`

func TestGetAccessMark(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name         string
		method       string
		path         string
		expectedMark AccessMark
	}{
		{"returns public when the operation is marked public", http.MethodPost, "/sessions", AccessMarkPublic},
		{"returns admin when the operation is marked admin", http.MethodGet, "/users", AccessMarkAdmin},
		{"returns signed in when the operation has no mark", http.MethodGet, "/workspaces", AccessMarkSignedIn},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()
			contractRouter, err := NewAccessCheckedContractRouter([]byte(markedContract))
			require.NoError(t, err)
			guard := RequireContractOperation(contractRouter, slog.New(slog.DiscardHandler))

			markByRequest := make(chan AccessMark, 1)
			engine := gin.New()
			engine.Group("/api/v1", guard).Handle(testCase.method, testCase.path, func(ginContext *gin.Context) {
				markByRequest <- GetAccessMark(ginContext.Request.Context())
				ginContext.Status(http.StatusNoContent)
			})

			response := httptest.NewRecorder()
			engine.ServeHTTP(response, httptest.NewRequestWithContext(
				t.Context(), testCase.method, "/api/v1"+testCase.path, nil,
			))

			require.Equal(t, http.StatusNoContent, response.Code)
			assert.Equal(t, testCase.expectedMark, <-markByRequest)
		})
	}

	t.Run("returns signed in when the guard did not see the request", func(t *testing.T) {
		t.Parallel()

		assert.Equal(t, AccessMarkSignedIn, GetAccessMark(t.Context()))
	})
}

func TestNewAccessCheckedContractRouter(t *testing.T) {
	t.Parallel()

	t.Run("returns an error when an operation carries an unknown mark", func(t *testing.T) {
		t.Parallel()
		contractWithUnknownMark := `
openapi: 3.0.3
info:
  title: Unknown mark
  version: 1.0.0
paths:
  /users:
    get:
      operationId: listUsers
      x-access: everyone
      responses:
        "204":
          description: The users
`

		_, err := NewAccessCheckedContractRouter([]byte(contractWithUnknownMark))

		require.ErrorIs(t, err, ErrUnknownAccessMark)
	})
}
