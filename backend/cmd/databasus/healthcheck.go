package main

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"time"
)

const (
	healthcheckSubcommand = "healthcheck"
	healthcheckTimeout    = 4 * time.Second
)

// runHealthcheck only asks the running server for its version, so it needs no setting.
func runHealthcheck(ctx context.Context, standardOutput, errorOutput io.Writer) int {
	err := checkServerAlive(ctx, "http://localhost"+serverAddress)
	if err != nil {
		_, _ = fmt.Fprintf(errorOutput, "healthcheck failed: %v\n", err)

		return 1
	}

	_, _ = fmt.Fprintln(standardOutput, "healthcheck ok")

	return 0
}

func checkServerAlive(ctx context.Context, baseURL string) error {
	requestCtx, cancel := context.WithTimeout(ctx, healthcheckTimeout)
	defer cancel()

	request, err := http.NewRequestWithContext(requestCtx, http.MethodGet, baseURL+"/api/v1/system/version", nil)
	if err != nil {
		return err
	}

	response, err := http.DefaultClient.Do(request)
	if err != nil {
		return err
	}
	defer func() { _ = response.Body.Close() }()

	_, _ = io.Copy(io.Discard, response.Body)

	if response.StatusCode != http.StatusOK {
		return fmt.Errorf("%w: %d", ErrUnexpectedStatus, response.StatusCode)
	}

	return nil
}
