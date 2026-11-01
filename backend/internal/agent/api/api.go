package api

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/hashicorp/go-retryablehttp"

	"databasus-backend/internal/lib/agentclient"
)

// reportRetryDelay returns the exponential backoff for report attempt n
// (1-based): 1s,2s,4s,8s,16s,32s, capped at maxBackoff. The exponent is
// clamped before the shift so a large attempt count cannot overflow.
func reportRetryDelay(attempt int) time.Duration {
	return expBackoff(attempt - 1)
}

func expBackoff(shift int) time.Duration {
	const maxShift = 5 // 2^5 s = 32s = maxBackoff

	if shift < 0 {
		shift = 0
	}

	if shift >= maxShift {
		return maxBackoff
	}

	d := time.Duration(1<<shift) * time.Second
	if d > maxBackoff {
		return maxBackoff
	}

	return d
}

type Client struct {
	timeAfterFn       func(time.Duration) <-chan time.Time
	json              *agentclient.ClientWithResponses
	noRetry           *agentclient.ClientWithResponses
	stream            *agentclient.Client
	retryHTTP         *http.Client
	streamHTTP        *http.Client
	host              string
	token             string
	agentID           string
	log               *slog.Logger
	reportRetryBudget time.Duration
}

type Deps struct {
	Host    string
	Token   string
	AgentID string
	Logger  *slog.Logger
}

func NewClient(deps Deps) *Client {
	retryClient := retryablehttp.NewClient()
	retryClient.RetryMax = maxRetryAttempts - 1
	retryClient.RetryWaitMin = retryBaseDelay
	retryClient.RetryWaitMax = 4 * retryBaseDelay
	retryClient.Logger = deps.Logger
	retryClient.HTTPClient.Timeout = apiCallTimeout
	retryClient.CheckRetry = func(ctx context.Context, response *http.Response, err error) (bool, error) {
		if ctx.Err() != nil {
			return false, ctx.Err()
		}
		return err != nil || response.StatusCode >= 500 || response.StatusCode == http.StatusTooManyRequests, nil
	}

	client := &Client{
		timeAfterFn:       time.After,
		host:              strings.TrimRight(deps.Host, "/"),
		token:             deps.Token,
		agentID:           deps.AgentID,
		log:               deps.Logger,
		retryHTTP:         retryClient.StandardClient(),
		streamHTTP:        &http.Client{},
		reportRetryBudget: time.Minute,
	}
	client.json = &agentclient.ClientWithResponses{ClientInterface: client.newProtocolClient(client.retryHTTP)}
	client.noRetry = &agentclient.ClientWithResponses{
		ClientInterface: client.newProtocolClient(&http.Client{Timeout: apiCallTimeout}),
	}
	client.stream = client.newProtocolClient(client.streamHTTP)
	return client
}

func (c *Client) FetchServerVersion(ctx context.Context) (string, error) {
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, c.buildURL(versionPath), nil)
	if err != nil {
		return "", fmt.Errorf("create version request: %w", err)
	}
	c.setStreamHeaders(request)

	response, err := c.retryHTTP.Do(request)
	if err != nil {
		return "", err
	}
	if response.StatusCode >= 400 {
		return "", responseError(response, "fetch server version")
	}
	defer func() { _ = response.Body.Close() }()

	var version struct {
		Version string `json:"version"`
	}
	if err := json.NewDecoder(response.Body).Decode(&version); err != nil {
		return "", fmt.Errorf("decode server version: %w", err)
	}
	return version.Version, nil
}

func (c *Client) DownloadVerificationAgentBinary(
	ctx context.Context,
	arch, destPath, fallbackVersion string,
) (string, error) {
	requestURL := c.buildURL(verificationAgentBinaryPath) + "?" + url.Values{"arch": {arch}}.Encode()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, requestURL, nil)
	if err != nil {
		return "", fmt.Errorf("create agent download request: %w", err)
	}

	c.setStreamHeaders(req)

	resp, err := c.streamHTTP.Do(req)
	if err != nil {
		return "", err
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("server returned %d for verification agent download", resp.StatusCode)
	}

	file, err := os.Create(destPath)
	if err != nil {
		return "", err
	}
	defer func() { _ = file.Close() }()

	if _, err := io.Copy(file, resp.Body); err != nil {
		return "", err
	}

	downloadedVersion := strings.TrimSpace(resp.Header.Get(agentVersionHeader))
	if downloadedVersion == "" {
		return fallbackVersion, nil
	}

	return downloadedVersion, nil
}

func (c *Client) Heartbeat(
	ctx context.Context,
	request agentclient.HeartbeatRequest,
) (*agentclient.HeartbeatResponse, error) {
	agentID, err := uuid.Parse(c.agentID)
	if err != nil {
		return nil, fmt.Errorf("parse agent ID: %w", err)
	}
	response, err := c.json.RecordHeartbeatWithResponse(ctx, agentID, request)
	if err != nil {
		return nil, err
	}
	if response.StatusCode() != http.StatusOK {
		return nil, &ResponseError{Op: "heartbeat", StatusCode: response.StatusCode(), Body: string(response.Body)}
	}
	if response.JSON200 == nil {
		return nil, fmt.Errorf("heartbeat response has no JSON body")
	}
	return response.JSON200, nil
}

func (c *Client) ClaimVerification(
	ctx context.Context,
	capacity agentclient.AgentCapacity,
) (*agentclient.VerificationAssignment, error) {
	agentID, err := uuid.Parse(c.agentID)
	if err != nil {
		return nil, fmt.Errorf("parse agent ID: %w", err)
	}
	response, err := c.noRetry.ClaimVerificationWithResponse(ctx, agentID, agentclient.ClaimRequest{Capacity: capacity})
	if err != nil {
		return nil, err
	}
	if response.StatusCode() == http.StatusNoContent {
		return nil, nil
	}
	if response.StatusCode() != http.StatusOK {
		return nil, &ResponseError{Op: "claim", StatusCode: response.StatusCode(), Body: string(response.Body)}
	}
	if response.JSON200 == nil {
		return nil, fmt.Errorf("claim response has no JSON body")
	}
	return response.JSON200, nil
}

// DownloadBackup opens the plaintext backup stream. On 200 it returns the
// response body for the caller to stream into the container. A non-2xx status
// is returned as *ResponseError so the caller can tell 410 (drop, no report)
// from other 4xx (agent-setup failure) from 5xx (retryable). A transport error
// is returned as-is (retryable).
func (c *Client) DownloadBackup(
	ctx context.Context,
	verificationID uuid.UUID,
) (io.ReadCloser, error) {
	agentID, err := uuid.Parse(c.agentID)
	if err != nil {
		return nil, fmt.Errorf("parse agent ID: %w", err)
	}
	resp, err := c.stream.StreamVerificationBackup(ctx, agentID, verificationID)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
		_ = resp.Body.Close()

		return nil, &ResponseError{
			Op:         "backup stream",
			StatusCode: resp.StatusCode,
			Body:       string(body),
		}
	}

	return resp.Body, nil
}

// Report submits the terminal result with bounded retry-with-deadline. Transport
// errors and 5xx are retried with exponential backoff under reportRetryBudget; a
// 410 returns ErrReportGone immediately (zero retries); any other 4xx is a
// non-retryable error; budget exhaustion returns ErrReportBudgetExhausted so the
// run is reclaimed by the backend on the agent's next heartbeat.
func (c *Client) Report(
	ctx context.Context,
	verificationID uuid.UUID,
	req agentclient.VerificationReport,
) error {
	ctx, cancel := context.WithTimeoutCause(ctx, c.reportRetryBudget, ErrReportBudgetExhausted)
	defer cancel()

	agentID, err := uuid.Parse(c.agentID)
	if err != nil {
		return fmt.Errorf("parse agent ID: %w", err)
	}
	for attempt := 1; ; attempt++ {
		resp, err := c.noRetry.ReportVerification(ctx, agentID, verificationID, req)

		if err == nil {
			switch {
			case resp.StatusCode == http.StatusNoContent:
				_ = resp.Body.Close()
				return nil
			case resp.StatusCode == http.StatusGone:
				_ = resp.Body.Close()
				return ErrReportGone
			case resp.StatusCode < 500:
				return responseError(resp, "report")
			default:
				_ = resp.Body.Close()
			}
		}

		select {
		case <-ctx.Done():
			return context.Cause(ctx)
		case <-c.timeAfterFn(reportRetryDelay(attempt)):
		}
	}
}

func (c *Client) buildURL(path string) string {
	return c.host + path
}

func (c *Client) newProtocolClient(doer agentclient.HttpRequestDoer) *agentclient.Client {
	return &agentclient.Client{Server: c.host + "/api/v1/", Client: doer, RequestEditors: []agentclient.RequestEditorFn{
		func(_ context.Context, request *http.Request) error { c.setStreamHeaders(request); return nil },
	}}
}

func (c *Client) setStreamHeaders(req *http.Request) {
	if h := bearerHeader(c.token); h != "" {
		req.Header.Set("Authorization", h)
	}

	req.Header.Set("Cache-Control", "no-cache")
	req.Header.Set("Pragma", "no-cache")
}

func bearerHeader(token string) string {
	if token == "" {
		return ""
	}

	return "Bearer " + token
}
