package logging

import (
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel/exporters/otlp/otlplog/otlploggrpc"
	"go.opentelemetry.io/otel/exporters/otlp/otlplog/otlploghttp"
	collectorpb "go.opentelemetry.io/proto/otlp/collector/logs/v1"
	"google.golang.org/protobuf/proto"
)

type receivedExport struct {
	headers http.Header
	request *collectorpb.ExportLogsServiceRequest
}

func startOTLPHTTPReceiver(t *testing.T) (string, <-chan receivedExport) {
	t.Helper()

	exports := make(chan receivedExport, 1)

	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		body, err := io.ReadAll(request.Body)
		if !assert.NoError(t, err) {
			return
		}

		exportRequest := &collectorpb.ExportLogsServiceRequest{}
		if !assert.NoError(t, proto.Unmarshal(body, exportRequest)) {
			return
		}

		select {
		case exports <- receivedExport{headers: request.Header.Clone(), request: exportRequest}:
		default:
		}

		response, err := proto.Marshal(&collectorpb.ExportLogsServiceResponse{})
		if !assert.NoError(t, err) {
			return
		}

		writer.Header().Set("Content-Type", "application/x-protobuf")
		_, _ = writer.Write(response)
	}))
	t.Cleanup(server.Close)

	return server.URL + "/v1/logs", exports
}

func Test_NewOpenTelemetryHandler_OverHttp_DeliversRecordToEndpoint(t *testing.T) {
	endpointURL, exports := startOTLPHTTPReceiver(t)

	openTelemetry, err := parseOpenTelemetryEndpoint(endpointURL, map[string]string{"x-tenant": "acme"})
	require.NoError(t, err)
	require.NotNil(t, openTelemetry)

	handler, shutdown, err := newOpenTelemetryHandler(t.Context(), *openTelemetry, "3.26.0")
	require.NoError(t, err)

	slog.New(handler).InfoContext(t.Context(), "backup finished", slog.String("database_id", "db-42"))
	require.NoError(t, shutdown(t.Context()))

	var export receivedExport
	select {
	case export = <-exports:
	case <-time.After(10 * time.Second):
		t.Fatal("no OTLP export reached the endpoint")
	}

	assert.Equal(t, "acme", export.headers.Get("x-tenant"))

	logRecords := export.request.GetResourceLogs()[0].GetScopeLogs()[0].GetLogRecords()
	require.Len(t, logRecords, 1)
	assert.Equal(t, "backup finished", logRecords[0].GetBody().GetStringValue())

	attributesByKey := map[string]string{}
	for _, attribute := range logRecords[0].GetAttributes() {
		attributesByKey[attribute.GetKey()] = attribute.GetValue().GetStringValue()
	}
	assert.Equal(t, "db-42", attributesByKey["database_id"])
}

func Test_NewOpenTelemetryHandler_WithEncodedHeaderValue_SendsItDecodedOnTheWire(t *testing.T) {
	endpointURL, exports := startOTLPHTTPReceiver(t)

	headers, err := parseOpenTelemetryHeaders("VL-Extra-Fields=host%3Dweb1%2Cdc%3Deu")
	require.NoError(t, err)

	openTelemetry, err := parseOpenTelemetryEndpoint(endpointURL, headers)
	require.NoError(t, err)
	require.NotNil(t, openTelemetry)

	handler, shutdown, err := newOpenTelemetryHandler(t.Context(), *openTelemetry, "")
	require.NoError(t, err)

	slog.New(handler).InfoContext(t.Context(), "storage created")
	require.NoError(t, shutdown(t.Context()))

	var export receivedExport
	select {
	case export = <-exports:
	case <-time.After(10 * time.Second):
		t.Fatal("no OTLP export reached the endpoint")
	}

	assert.Equal(t, "host=web1,dc=eu", export.headers.Get("VL-Extra-Fields"))
}

func Test_NewOpenTelemetryHandler_WithScopedAttrs_ShipsThemWithTheRecord(t *testing.T) {
	endpointURL, exports := startOTLPHTTPReceiver(t)

	openTelemetry, err := parseOpenTelemetryEndpoint(endpointURL, map[string]string{})
	require.NoError(t, err)

	handler, shutdown, err := newOpenTelemetryHandler(t.Context(), *openTelemetry, "")
	require.NoError(t, err)

	slog.New(handler).With(slog.String("workspace_id", "ws-7")).InfoContext(t.Context(), "storage created")
	require.NoError(t, shutdown(t.Context()))

	var export receivedExport
	select {
	case export = <-exports:
	case <-time.After(10 * time.Second):
		t.Fatal("no OTLP export reached the endpoint")
	}

	logRecords := export.request.GetResourceLogs()[0].GetScopeLogs()[0].GetLogRecords()
	require.Len(t, logRecords, 1)

	attributesByKey := map[string]string{}
	for _, attribute := range logRecords[0].GetAttributes() {
		attributesByKey[attribute.GetKey()] = attribute.GetValue().GetStringValue()
	}
	assert.Equal(t, "ws-7", attributesByKey["workspace_id"])
}

// The gRPC transport is covered by branch selection only; an end-to-end gRPC round trip would need
// a registered LogsServiceServer for one assertion the manual Collector check already makes.
// otlploggrpc.New dials lazily and never errors here, so the exporter type is the real assertion -
// without it the HTTP branch would satisfy this test just as well.
func Test_NewOpenTelemetryExporter_WithGrpcScheme_SelectsTheGrpcExporter(t *testing.T) {
	openTelemetry, err := parseOpenTelemetryEndpoint("grpc://localhost:4317", map[string]string{})
	require.NoError(t, err)

	exporter, err := newOpenTelemetryExporter(t.Context(), *openTelemetry)
	require.NoError(t, err)

	require.IsType(t, &otlploggrpc.Exporter{}, exporter)
	require.NoError(t, exporter.Shutdown(t.Context()))
}

func Test_NewOpenTelemetryExporter_WithHttpScheme_SelectsTheHttpExporter(t *testing.T) {
	openTelemetry, err := parseOpenTelemetryEndpoint("http://localhost:4318/v1/logs", map[string]string{})
	require.NoError(t, err)

	exporter, err := newOpenTelemetryExporter(t.Context(), *openTelemetry)
	require.NoError(t, err)

	require.IsType(t, &otlploghttp.Exporter{}, exporter)
	require.NoError(t, exporter.Shutdown(t.Context()))
}
