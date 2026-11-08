package main

import (
	"bytes"
	"context"
	"log/slog"
	"net/http"
	"strings"
	"testing"

	"databasus-backend/internal/lib/logging"
)

type seedRunOutcome struct {
	exitCode int
	output   string
}

func runSeedForTest(t *testing.T, params seedParams) (int, string) {
	t.Helper()
	outcome := runSeedWithContext(t.Context(), params)
	return outcome.exitCode, outcome.output
}

func runSeedWithContext(ctx context.Context, params seedParams) seedRunOutcome {
	var output bytes.Buffer
	exitCode := runSeed(
		ctx,
		&http.Client{Timeout: requestTimeout},
		params,
		logging.NewWithSinks(slog.NewTextHandler(&output, nil)).GetLogger(),
	)

	return seedRunOutcome{exitCode: exitCode, output: strings.TrimSpace(output.String())}
}
