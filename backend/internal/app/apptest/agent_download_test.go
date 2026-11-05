package apptest_test

import (
	"context"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"databasus-backend/internal/app/apptest"
	apiclient "databasus-backend/internal/lib/apiclient"
)

func TestSystemAgentDownload(t *testing.T) {
	t.Parallel()

	for _, architecture := range []apiclient.AgentArchitecture{apiclient.AgentArchitectureAmd64, apiclient.AgentArchitectureArm64} {
		t.Run("streams the binary with exact headers when architecture is "+string(architecture), func(t *testing.T) {
			t.Parallel()

			binaryContents := strings.Repeat("agent "+string(architecture)+"\n", 512)
			directory := t.TempDir()
			require.NoError(
				t,
				os.WriteFile(
					filepath.Join(directory, "databasus-verification-agent-linux-"+string(architecture)),
					[]byte(binaryContents),
					0o755,
				),
			)

			application := apptest.New(t, apptest.WithAgentBinariesDir(directory))
			client := application.NewClient(t)

			response, err := client.DownloadVerificationAgentWithResponse(
				t.Context(),
				&apiclient.DownloadVerificationAgentParams{Arch: architecture},
				func(_ context.Context, request *http.Request) error {
					request.Header.Set("Accept-Encoding", "gzip")
					return nil
				},
			)
			require.NoError(t, err)
			require.Equal(t, http.StatusOK, response.StatusCode(), string(response.Body))
			assert.False(t, response.HTTPResponse.Uncompressed)
			assert.Empty(t, response.HTTPResponse.Header.Get("Content-Encoding"))
			assert.Equal(t, strconv.Itoa(len(response.Body)), response.HTTPResponse.Header.Get("Content-Length"))
			assert.Equal(t, int64(len(response.Body)), response.HTTPResponse.ContentLength)
			assert.Equal(t, binaryContents, string(response.Body))
			assert.Equal(t, "application/octet-stream", response.HTTPResponse.Header.Get("Content-Type"))
			assert.Equal(
				t,
				"attachment; filename=databasus-verification-agent",
				response.HTTPResponse.Header.Get("Content-Disposition"),
			)

			version, err := client.GetVersionWithResponse(t.Context())
			require.NoError(t, err)
			require.NotNil(t, version.JSON200)
			assert.True(t, version.HTTPResponse.Uncompressed)
			assert.Equal(t, version.JSON200.Version, response.HTTPResponse.Header.Get("X-Databasus-Version"))
		})
	}
	t.Run("returns contract errors when architecture or binary is unavailable", func(t *testing.T) {
		t.Parallel()

		directory := t.TempDir()
		application := apptest.New(t, apptest.WithAgentBinariesDir(directory))
		client := application.NewClient(t)

		missing, err := client.DownloadVerificationAgentWithResponse(
			t.Context(),
			&apiclient.DownloadVerificationAgentParams{Arch: apiclient.AgentArchitectureAmd64},
		)
		require.NoError(t, err)
		assert.Equal(t, http.StatusNotFound, missing.StatusCode())

		invalid, err := client.DownloadVerificationAgentWithResponse(
			t.Context(),
			&apiclient.DownloadVerificationAgentParams{Arch: "riscv64"},
		)
		require.NoError(t, err)
		assert.Equal(t, http.StatusBadRequest, invalid.StatusCode())

		request, err := http.NewRequestWithContext(
			t.Context(),
			http.MethodGet,
			application.URL+"/api/v1/system/verification-agent",
			nil,
		)
		require.NoError(t, err)
		response, err := http.DefaultClient.Do(request)
		require.NoError(t, err)
		defer func() { require.NoError(t, response.Body.Close()) }()
		assert.Equal(t, http.StatusBadRequest, response.StatusCode)
	})
	t.Run("hides filesystem errors when the binary is a directory", func(t *testing.T) {
		t.Parallel()

		directory := t.TempDir()
		require.NoError(t, os.Mkdir(filepath.Join(directory, "databasus-verification-agent-linux-amd64"), 0o755))
		application := apptest.New(t, apptest.WithAgentBinariesDir(directory))
		response, err := application.NewClient(t).
			DownloadVerificationAgentWithResponse(t.Context(), &apiclient.DownloadVerificationAgentParams{Arch: apiclient.AgentArchitectureAmd64})
		require.NoError(t, err)
		assert.Equal(t, http.StatusInternalServerError, response.StatusCode())
		assert.NotContains(t, string(response.Body), directory)
	})
}

func TestAgentDiscoveryAfterHeartbeatRefusal(t *testing.T) {
	t.Parallel()

	t.Run("serves public updates when a gigabyte-only heartbeat is rejected", func(t *testing.T) {
		t.Parallel()

		directory := t.TempDir()
		require.NoError(
			t,
			os.WriteFile(
				filepath.Join(directory, "databasus-verification-agent-linux-amd64"),
				[]byte("updated agent"),
				0o755,
			),
		)

		application := apptest.New(t, apptest.WithAgentBinariesDir(directory))
		admin := application.CreateAdmin(t)
		agent := application.CreateVerificationAgent(t, admin)

		request, err := http.NewRequestWithContext(
			t.Context(),
			http.MethodPost,
			application.URL+"/api/v1/agent/verification/"+agent.Agent.ID.String()+"/heartbeat",
			strings.NewReader(
				`{"maxCpu":4,"maxRamGb":8,"maxDiskGb":20,"maxConcurrentJobs":1,"currentVerificationIds":[]}`,
			),
		)
		require.NoError(t, err)
		request.Header.Set("Authorization", "Bearer "+agent.Token)
		request.Header.Set("Content-Type", "application/json")
		refused, err := http.DefaultClient.Do(request)
		require.NoError(t, err)
		require.NoError(t, refused.Body.Close())
		require.Equal(t, http.StatusBadRequest, refused.StatusCode)

		client := application.NewClient(t)
		version, err := client.GetVersionWithResponse(t.Context())
		require.NoError(t, err)
		require.NotNil(t, version.JSON200)
		binary, err := client.DownloadVerificationAgentWithResponse(
			t.Context(),
			&apiclient.DownloadVerificationAgentParams{Arch: apiclient.AgentArchitectureAmd64},
		)
		require.NoError(t, err)

		assert.Equal(t, http.StatusOK, binary.StatusCode())
		assert.Equal(t, "updated agent", string(binary.Body))
		assert.Nil(t, application.ListVerificationAgents(t, admin)[0].LastSeenAt)
	})
}
