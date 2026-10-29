package system_agent

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	system_version "databasus-backend/internal/features/system/version"
)

func TestDownloadVerificationAgent(t *testing.T) {
	for _, scenario := range []struct {
		name      string
		arch      string
		isPresent bool
		status    int
	}{
		{"sends the binary and version when the binary exists", "amd64", true, http.StatusOK},
		{"returns bad request when the architecture is unknown", "riscv64", false, http.StatusBadRequest},
		{"returns not found when the binary is missing", "amd64", false, http.StatusNotFound},
	} {
		t.Run(scenario.name, func(t *testing.T) {
			directory := t.TempDir()
			if scenario.isPresent {
				require.NoError(
					t,
					os.WriteFile(
						filepath.Join(directory, "databasus-verification-agent-linux-amd64"),
						[]byte("agent bytes"),
						0o755,
					),
				)
			}
			router := gin.New()
			controller := AgentController{BinariesDir: directory}
			controller.RegisterRoutes(router.Group("/api/v1"))
			response := httptest.NewRecorder()
			router.ServeHTTP(
				response,
				httptest.NewRequest(http.MethodGet, "/api/v1/system/verification-agent?arch="+scenario.arch, nil),
			)
			require.Equal(t, scenario.status, response.Code)
			if scenario.isPresent {
				assert.Equal(t, "agent bytes", response.Body.String())
				assert.Equal(t, system_version.GetAppVersion(), response.Header().Get(AgentVersionHeader))
				assert.Equal(t, "application/octet-stream", response.Header().Get("Content-Type"))
			} else {
				assert.Empty(t, response.Header().Get(AgentVersionHeader))
			}
		})
	}
}
