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

	"github.com/go-resty/resty/v2"
	"github.com/google/uuid"
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
	timeAfterFn func(time.Duration) <-chan time.Time
	json        *resty.Client
	noRetry     *resty.Client
	streamHTTP  *http.Client
	host        string
	token       string
	agentID     string
	log         *slog.Logger
}

type Deps struct {
	Host    string
	Token   string
	AgentID string
	Logger  *slog.Logger
}

func NewClient(deps Deps) *Client {
	setAuth := func(_ *resty.Client, req *resty.Request) error {
		if h := bearerHeader(deps.Token); h != "" {
			req.SetHeader("Authorization", h)
		}

		return nil
	}

	jsonClient := resty.New().
		SetTimeout(apiCallTimeout).
		SetRetryCount(maxRetryAttempts - 1).
		SetRetryWaitTime(retryBaseDelay).
		SetRetryMaxWaitTime(4 * retryBaseDelay).
		AddRetryCondition(func(resp *resty.Response, err error) bool {
			return err != nil || resp.StatusCode() >= 500
		}).
		SetHeaders(noCacheHeaders).
		OnBeforeRequest(setAuth)

	// Claim and report own their retry policy explicitly (claim: loop retries
	// forever; report: bounded retry-with-deadline). A retrying client here
	// would make "exactly N retries / give up at the budget" untestable.
	noRetryClient := resty.New().
		SetTimeout(apiCallTimeout).
		SetHeaders(noCacheHeaders).
		OnBeforeRequest(setAuth)

	return &Client{
		timeAfterFn: time.After,
		json:        jsonClient,
		noRetry:     noRetryClient,
		streamHTTP:  &http.Client{},
		host:        deps.Host,
		token:       deps.Token,
		agentID:     deps.AgentID,
		log:         deps.Logger,
	}
}

func (c *Client) FetchServerVersion(ctx context.Context) (string, error) {
	var ver versionResponse

	httpResp, err := c.json.R().
		SetContext(ctx).
		SetResult(&ver).
		Get(c.buildURL(versionPath))
	if err != nil {
		return "", err
	}

	if err := c.checkResponse(httpResp, "fetch server version"); err != nil {
		return "", err
	}

	return ver.Version, nil
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
	request HeartbeatRequest,
) (*HeartbeatResponse, error) {
	var response HeartbeatResponse

	httpResp, err := c.json.R().
		SetContext(ctx).
		SetBody(request).
		SetResult(&response).
		Post(c.buildURL(fmt.Sprintf(heartbeatPathFmt, c.agentID)))
	if err != nil {
		return nil, err
	}

	if err := c.checkResponse(httpResp, "heartbeat"); err != nil {
		return nil, err
	}

	return &response, nil
}

// ClaimVerification performs a single claim attempt. 200 returns the assignment;
// 204 returns (nil, nil) meaning nothing fits; any other status is an error.
// The runner loop owns retry/backoff — this never loops.
func (c *Client) ClaimVerification(
	ctx context.Context,
	capacity AgentCapacity,
) (*JobAssignment, error) {
	resp, err := c.noRetry.R().
		SetContext(ctx).
		SetBody(ClaimRequest{Capacity: capacity}).
		Post(c.buildURL(fmt.Sprintf(claimPathFmt, c.agentID)))
	if err != nil {
		return nil, err
	}

	if resp.StatusCode() == http.StatusNoContent {
		return nil, nil
	}

	if resp.StatusCode() != http.StatusOK {
		return nil, &ResponseError{Op: "claim", StatusCode: resp.StatusCode(), Body: resp.String()}
	}

	var assignment JobAssignment
	if err := json.Unmarshal(resp.Body(), &assignment); err != nil {
		return nil, fmt.Errorf("decode job assignment: %w", err)
	}

	return &assignment, nil
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
	requestURL := c.buildURL(fmt.Sprintf(backupStreamPathFmt, c.agentID, verificationID))

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, requestURL, nil)
	if err != nil {
		return nil, fmt.Errorf("create backup stream request: %w", err)
	}

	c.setStreamHeaders(req)

	resp, err := c.streamHTTP.Do(req)
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
	req ReportRequest,
) error {
	ctx, cancel := context.WithTimeoutCause(ctx, reportRetryBudget, ErrReportBudgetExhausted)
	defer cancel()

	requestURL := c.buildURL(fmt.Sprintf(reportPathFmt, c.agentID, verificationID))

	for attempt := 1; ; attempt++ {
		resp, err := c.noRetry.R().
			SetContext(ctx).
			SetBody(req).
			Post(requestURL)

		if err == nil {
			switch {
			case resp.StatusCode() == http.StatusNoContent:
				return nil
			case resp.StatusCode() == http.StatusGone:
				return ErrReportGone
			case resp.StatusCode() < 500:
				return &ResponseError{
					Op:         "report",
					StatusCode: resp.StatusCode(),
					Body:       resp.String(),
				}
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

func (c *Client) checkResponse(resp *resty.Response, method string) error {
	if resp.StatusCode() >= 400 {
		return fmt.Errorf("%s: server returned status %d: %s", method, resp.StatusCode(), resp.String())
	}

	return nil
}

func (c *Client) setStreamHeaders(req *http.Request) {
	if h := bearerHeader(c.token); h != "" {
		req.Header.Set("Authorization", h)
	}

	for name, value := range noCacheHeaders {
		req.Header.Set(name, value)
	}
}

func bearerHeader(token string) string {
	if token == "" {
		return ""
	}

	return "Bearer " + token
}
