package logging

import (
	"encoding/base64"
	"errors"
	"fmt"
	"log/slog"
	"net/url"
	"os"
	"strings"

	"databasus-backend/internal/lib/config"
)

const (
	logFileMaxSizeMB  = 5
	logFileMaxBackups = 3

	serviceName = "databasus"
)

type logFileSettings struct {
	isEnabled bool
	path      string
}

type openTelemetrySettings struct {
	endpointURL string
	isGRPC      bool
	headers     map[string]string
}

type settings struct {
	level          slog.Level
	serviceVersion string
	file           logFileSettings
	openTelemetry  *openTelemetrySettings
}

func newSettings(loadedConfig config.Config) (settings, error) {
	headers, err := parseOpenTelemetryHeaders(loadedConfig.OpenTelemetryHeaders)
	if err != nil {
		return settings{}, err
	}

	openTelemetry, err := parseOpenTelemetryEndpoint(loadedConfig.OpenTelemetryURL, headers)
	if err != nil {
		return settings{}, err
	}

	return settings{
		level:          parseLevel(loadedConfig.LogLevel),
		serviceVersion: loadedConfig.AppVersion,
		file:           logFileSettings{isEnabled: loadedConfig.IsLogFileEnabled(), path: loadedConfig.LogFilePath},
		openTelemetry:  openTelemetry,
	}, nil
}

func parseLevel(rawLevel string) slog.Level {
	switch strings.ToLower(strings.TrimSpace(rawLevel)) {
	case "":
		return slog.LevelInfo
	case "debug":
		return slog.LevelDebug
	case "info":
		return slog.LevelInfo
	case "warn", "warning":
		return slog.LevelWarn
	case "error":
		return slog.LevelError
	}

	fmt.Fprintf(os.Stderr, "LOG_LEVEL %q is not one of debug, info, warn, error - falling back to info\n", rawLevel)

	return slog.LevelInfo
}

// parseOpenTelemetryHeaders deliberately matches the format of the standard
// OTEL_EXPORTER_OTLP_HEADERS variable, percent-decoding included: that is the only way to write a
// value containing the separator, and VictoriaLogs takes comma-separated lists in VL-Stream-Fields
// and VL-Extra-Fields. Errors name the position rather than the entry, since these carry tokens
// and stderr is collected.
func parseOpenTelemetryHeaders(rawHeaders string) (map[string]string, error) {
	headers := make(map[string]string)

	for index, pair := range strings.Split(rawHeaders, ",") {
		if strings.TrimSpace(pair) == "" {
			continue
		}

		key, encodedValue, isPair := strings.Cut(pair, "=")

		key = strings.TrimSpace(key)
		if !isPair || key == "" {
			return nil, fmt.Errorf("OPEN_TELEMETRY_HEADERS entry %d is not key=value", index+1)
		}

		value, err := url.PathUnescape(strings.TrimSpace(encodedValue))
		if err != nil {
			return nil, fmt.Errorf("OPEN_TELEMETRY_HEADERS entry %d has a broken percent escape", index+1)
		}

		headers[key] = value
	}

	return headers, nil
}

// parseOpenTelemetryEndpoint rejects a missing or unknown scheme rather than guessing: SigNoz and
// friends document bare "host:443" endpoints, and treating those as HTTP yields neither logs nor
// an error. grpc:// and grpcs:// are Databasus-level schemes - the gRPC exporter only understands
// http/https and derives TLS from the scheme, so they are translated here.
func parseOpenTelemetryEndpoint(rawURL string, headers map[string]string) (*openTelemetrySettings, error) {
	rawURL = strings.TrimSpace(rawURL)
	if rawURL == "" {
		return nil, nil
	}

	// Every error below reports the endpoint via reportableEndpoint or not at all: .env.example
	// tells operators to put credentials in this variable, and these messages go to stderr, which
	// is collected. A url.Error formats the whole raw URL, so it must not be wrapped either.
	endpoint, err := url.Parse(rawURL)
	if err != nil {
		return nil, errors.New("OPEN_TELEMETRY_URL is not a valid URL")
	}

	reportableEndpoint := getReportableEndpoint(endpoint)

	isGRPC := endpoint.Scheme == "grpc" || endpoint.Scheme == "grpcs"

	switch endpoint.Scheme {
	case "http", "grpc":
		endpoint.Scheme = "http"
	case "https", "grpcs":
		endpoint.Scheme = "https"
	default:
		return nil, fmt.Errorf(
			"OPEN_TELEMETRY_URL must start with http://, https://, grpc:// or grpcs://, got %q",
			reportableEndpoint,
		)
	}

	if endpoint.Host == "" {
		return nil, fmt.Errorf("OPEN_TELEMETRY_URL is missing a host: %q", reportableEndpoint)
	}

	// The exporter builds its request URL from host and path alone, so a query string would be
	// dropped without a word. Ingest options belong in OPEN_TELEMETRY_HEADERS: VictoriaLogs accepts
	// every one of its params as a VL-* header, and OTLP/gRPC has no query to carry them in.
	if endpoint.RawQuery != "" || endpoint.Fragment != "" {
		return nil, errors.New("OPEN_TELEMETRY_URL must not carry a query string or fragment, " +
			"pass ingest options as headers in OPEN_TELEMETRY_HEADERS instead")
	}

	if authorization, isPresent := takeBasicAuthorization(endpoint); isPresent && !hasAuthorization(headers) {
		headers["Authorization"] = authorization
	}

	return &openTelemetrySettings{
		endpointURL: endpoint.String(),
		isGRPC:      isGRPC,
		headers:     headers,
	}, nil
}

// getReportableEndpoint drops the query and the fragment as well as the password, since a
// mistyped endpoint is often one where an ingestion key was pasted as a query param.
func getReportableEndpoint(endpoint *url.URL) string {
	withoutIngestOptions := *endpoint
	withoutIngestOptions.RawQuery = ""
	withoutIngestOptions.Fragment = ""

	return withoutIngestOptions.Redacted()
}

func takeBasicAuthorization(endpoint *url.URL) (string, bool) {
	if endpoint.User == nil {
		return "", false
	}

	password, _ := endpoint.User.Password()
	credentials := endpoint.User.Username() + ":" + password
	endpoint.User = nil

	return "Basic " + base64.StdEncoding.EncodeToString([]byte(credentials)), true
}

func hasAuthorization(headers map[string]string) bool {
	for key := range headers {
		if strings.EqualFold(key, "Authorization") {
			return true
		}
	}

	return false
}
