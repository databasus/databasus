package upgrade

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
)

// apiClient sends JSON to one Databasus instance. The requests and the response shapes belong to
// previous_release_client_test.go and current_release_checks_test.go, because the two releases may
// speak different APIs.
type apiClient struct {
	httpClient *http.Client
	baseURL    string
	token      string
}

func newAPIClient(hostPort int) *apiClient {
	return &apiClient{
		httpClient: &http.Client{},
		baseURL:    fmt.Sprintf("http://127.0.0.1:%d/api/v1", hostPort),
		token:      "",
	}
}

func (c *apiClient) get(ctx context.Context, path string, response any) error {
	return c.send(ctx, http.MethodGet, path, nil, response)
}

func (c *apiClient) post(ctx context.Context, path string, body, response any) error {
	return c.send(ctx, http.MethodPost, path, body, response)
}

func (c *apiClient) send(ctx context.Context, method, path string, body, response any) error {
	var requestBody io.Reader

	if body != nil {
		encodedBody, err := json.Marshal(body)
		if err != nil {
			return fmt.Errorf("encode the request to %s %s: %w", method, path, err)
		}

		requestBody = bytes.NewReader(encodedBody)
	}

	request, err := http.NewRequestWithContext(ctx, method, c.baseURL+path, requestBody)
	if err != nil {
		return fmt.Errorf("build the request to %s %s: %w", method, path, err)
	}

	request.Header.Set("Content-Type", "application/json")

	if c.token != "" {
		request.Header.Set("Authorization", "Bearer "+c.token)
	}

	httpResponse, err := c.httpClient.Do(request)
	if err != nil {
		return fmt.Errorf("%s %s: %w", method, path, err)
	}
	defer func() { _ = httpResponse.Body.Close() }()

	responseBody, err := io.ReadAll(httpResponse.Body)
	if err != nil {
		return fmt.Errorf("read the response to %s %s: %w", method, path, err)
	}

	if httpResponse.StatusCode >= http.StatusBadRequest {
		return fmt.Errorf("%s %s answered %d: %s",
			method, path, httpResponse.StatusCode, strings.TrimSpace(string(responseBody)))
	}

	if response == nil {
		return nil
	}

	err = json.Unmarshal(responseBody, response)
	if err != nil {
		return fmt.Errorf("decode the response to %s %s: %w", method, path, err)
	}

	return nil
}
