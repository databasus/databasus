package logging

import (
	"encoding/base64"
	"log/slog"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func Test_ParseLevel_WithEachSupportedName_ReturnsMatchingLevel(t *testing.T) {
	levelsByName := map[string]slog.Level{
		"debug":   slog.LevelDebug,
		"DEBUG":   slog.LevelDebug,
		" info ":  slog.LevelInfo, //nolint:gocritic // surrounding whitespace is the case under test
		"warn":    slog.LevelWarn,
		"warning": slog.LevelWarn,
		"error":   slog.LevelError,
		"":        slog.LevelInfo,
		"loudest": slog.LevelInfo,
	}

	for name, level := range levelsByName {
		assert.Equal(t, level, parseLevel(name), "level name %q", name)
	}
}

func Test_ParseOpenTelemetryEndpoint_WhenUrlIsEmpty_DisablesSink(t *testing.T) {
	openTelemetry, err := parseOpenTelemetryEndpoint("  ", map[string]string{})

	require.NoError(t, err)
	assert.Nil(t, openTelemetry)
}

func Test_ParseOpenTelemetryEndpoint_WithHttpScheme_KeepsPathVerbatim(t *testing.T) {
	openTelemetry, err := parseOpenTelemetryEndpoint(
		"http://victorialogs:9428/insert/opentelemetry/v1/logs",
		map[string]string{},
	)

	require.NoError(t, err)
	require.NotNil(t, openTelemetry)
	assert.False(t, openTelemetry.isGRPC)
	assert.Equal(t, "http://victorialogs:9428/insert/opentelemetry/v1/logs", openTelemetry.endpointURL)
}

func Test_ParseOpenTelemetryEndpoint_WithGrpcSchemes_TranslatesToHttpForExporter(t *testing.T) {
	insecureEndpoint, err := parseOpenTelemetryEndpoint("grpc://graylog:4317", map[string]string{})
	require.NoError(t, err)
	require.NotNil(t, insecureEndpoint)
	assert.True(t, insecureEndpoint.isGRPC)
	assert.Equal(t, "http://graylog:4317", insecureEndpoint.endpointURL)

	secureEndpoint, err := parseOpenTelemetryEndpoint("grpcs://ingest.signoz.cloud:443", map[string]string{})
	require.NoError(t, err)
	require.NotNil(t, secureEndpoint)
	assert.True(t, secureEndpoint.isGRPC)
	assert.Equal(t, "https://ingest.signoz.cloud:443", secureEndpoint.endpointURL)
}

func Test_ParseOpenTelemetryEndpoint_WithMissingOrUnknownScheme_ReturnsError(t *testing.T) {
	rejectedURLs := []string{"ingest.eu.signoz.cloud:443", "ftp://logs:9428", "//logs:9428/v1/logs"}

	for _, rejectedURL := range rejectedURLs {
		openTelemetry, err := parseOpenTelemetryEndpoint(rejectedURL, map[string]string{})

		require.Error(t, err, "URL %q", rejectedURL)
		assert.Nil(t, openTelemetry)
		assert.Contains(t, err.Error(), "OPEN_TELEMETRY_URL")
	}
}

func Test_ParseOpenTelemetryEndpoint_WithQueryOrFragment_ReturnsErrorPointingAtHeaders(t *testing.T) {
	rejectedURLs := []string{
		"http://victorialogs:9428/insert/opentelemetry/v1/logs?_stream_fields=host",
		"http://victorialogs:9428/insert/opentelemetry/v1/logs#logs",
	}

	for _, rejectedURL := range rejectedURLs {
		openTelemetry, err := parseOpenTelemetryEndpoint(rejectedURL, map[string]string{})

		require.Error(t, err, "URL %q", rejectedURL)
		assert.Nil(t, openTelemetry)
		assert.Contains(t, err.Error(), "OPEN_TELEMETRY_HEADERS")
	}
}

func Test_ParseOpenTelemetryEndpoint_WhenRejectedUrlCarriesCredentials_KeepsThemOutOfTheError(t *testing.T) {
	rejectedURLs := []string{
		"ingest.eu.signoz.cloud:443?api-key=secret",
		"http://victorialogs:9428/insert/opentelemetry/v1/logs?api-key=secret",
		"ftp://viewer:secret@victorialogs:9428/insert/opentelemetry/v1/logs",
		"https://viewer:secret@/insert/opentelemetry/v1/logs",
	}

	for _, rejectedURL := range rejectedURLs {
		_, err := parseOpenTelemetryEndpoint(rejectedURL, map[string]string{})

		require.Error(t, err, "URL %q", rejectedURL)
		assert.NotContains(t, err.Error(), "secret")
	}
}

func Test_ParseOpenTelemetryEndpoint_WithUserInfo_MovesCredentialsIntoHeaderAndStripsThem(t *testing.T) {
	openTelemetry, err := parseOpenTelemetryEndpoint("http://viewer:hunter2@logs:9428/v1/logs", map[string]string{})

	require.NoError(t, err)
	require.NotNil(t, openTelemetry)
	assert.Equal(t, "http://logs:9428/v1/logs", openTelemetry.endpointURL)
	assert.Equal(t,
		"Basic "+base64.StdEncoding.EncodeToString([]byte("viewer:hunter2")),
		openTelemetry.headers["Authorization"],
	)
}

func Test_ParseOpenTelemetryEndpoint_WhenHeaderAndUserInfoBothSetAuthorization_HeaderWins(t *testing.T) {
	openTelemetry, err := parseOpenTelemetryEndpoint(
		"http://viewer:hunter2@logs:9428/v1/logs",
		map[string]string{"authorization": "Bearer from-header"},
	)

	require.NoError(t, err)
	require.NotNil(t, openTelemetry)
	assert.Equal(t, "Bearer from-header", openTelemetry.headers["authorization"])
	assert.NotContains(t, openTelemetry.headers, "Authorization")
	assert.Equal(t, "http://logs:9428/v1/logs", openTelemetry.endpointURL)
}

func Test_ParseOpenTelemetryHeaders_WithPaddedPairsAndEmptyEntries_KeepsEveryPair(t *testing.T) {
	headers, err := parseOpenTelemetryHeaders(" signoz-ingestion-key=abc123 , x-scope-org-id=tenant-1 ,, ")

	require.NoError(t, err)
	assert.Equal(t, map[string]string{
		"signoz-ingestion-key": "abc123",
		"x-scope-org-id":       "tenant-1",
	}, headers)
}

func Test_ParseOpenTelemetryHeaders_WithEncodedValue_DecodesPercentEscapes(t *testing.T) {
	headers, err := parseOpenTelemetryHeaders("VL-Extra-Fields=host%3Dweb1%2Cdc%3Deu,Authorization=Basic%20abc")

	require.NoError(t, err)
	assert.Equal(t, map[string]string{
		"VL-Extra-Fields": "host=web1,dc=eu",
		"Authorization":   "Basic abc",
	}, headers)
}

func Test_ParseOpenTelemetryHeaders_WithMalformedEntry_FailsWithoutEchoingValue(t *testing.T) {
	headers, err := parseOpenTelemetryHeaders("x-tenant=acme,signoz-ingestion-key-and-secret-token")

	require.Error(t, err)
	assert.Nil(t, headers)
	assert.Contains(t, err.Error(), "entry 2")
	assert.NotContains(t, err.Error(), "signoz-ingestion-key-and-secret-token")
}

func Test_ParseOpenTelemetryHeaders_WithBrokenPercentEscape_FailsWithoutEchoingValue(t *testing.T) {
	headers, err := parseOpenTelemetryHeaders("Authorization=Bearer%2secret")

	require.Error(t, err)
	assert.Nil(t, headers)
	assert.Contains(t, err.Error(), "entry 1")
	assert.NotContains(t, err.Error(), "secret")
}
