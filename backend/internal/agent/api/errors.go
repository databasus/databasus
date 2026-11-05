package api

import (
	"errors"
	"fmt"
	"io"
	"net/http"
)

// ErrReportGone signals the report route returned 410 — the row is no longer
// this agent's. The caller logs at info and drops; it is never retried.
var ErrReportGone = errors.New("verification no longer owned by this agent")

// ErrReportBudgetExhausted is the cause attached when report retries exceed
// reportRetryBudget; the run is then reclaimed by the backend on the agent's
// next heartbeat.
var ErrReportBudgetExhausted = errors.New("report retry budget exhausted")

type ResponseError struct {
	Op         string
	StatusCode int
	Body       string
}

func (e *ResponseError) Error() string {
	return fmt.Sprintf("%s: server returned status %d: %s", e.Op, e.StatusCode, e.Body)
}

func (e *ResponseError) IsRetryable() bool { return e.StatusCode >= 500 }

func (e *ResponseError) IsGone() bool { return e.StatusCode == http.StatusGone }

func responseError(response *http.Response, operation string) error {
	body, _ := io.ReadAll(io.LimitReader(response.Body, 4096))
	_ = response.Body.Close()
	return &ResponseError{Op: operation, StatusCode: response.StatusCode, Body: string(body)}
}
