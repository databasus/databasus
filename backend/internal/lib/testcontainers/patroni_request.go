package containers

import (
	"bytes"
	"fmt"
	"io"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

type patroniRequest struct {
	Endpoint Endpoint
	Method   string
	Path     string
	Body     []byte
}

func (fixture *Patroni) request(t *testing.T, spec patroniRequest) []byte {
	t.Helper()

	request, err := http.NewRequestWithContext(
		t.Context(),
		spec.Method,
		fmt.Sprintf("http://%s:%d%s", spec.Endpoint.Host, spec.Endpoint.Port, spec.Path),
		bytes.NewReader(spec.Body),
	)
	require.NoError(t, err)
	request.Header.Set("Content-Type", "application/json")
	httpClient := &http.Client{Timeout: 30 * time.Second}
	response, err := httpClient.Do(request)
	require.NoError(t, err)
	defer func() { _ = response.Body.Close() }()

	output, err := io.ReadAll(io.LimitReader(response.Body, 16384))
	require.NoError(t, err)

	require.True(
		t,
		response.StatusCode >= 200 && response.StatusCode < 300,
		"%s %s: %d %s",
		spec.Method,
		spec.Path,
		response.StatusCode,
		strings.ReplaceAll(string(output), PostgresPassword, "[REDACTED]"),
	)

	return output
}
