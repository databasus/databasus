package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
)

type apiClient struct {
	httpClient *http.Client
	baseURL    string
	token      string
}

func (c *apiClient) get(ctx context.Context, path string, response any) error {
	return c.send(ctx, http.MethodGet, path, nil, response)
}

func (c *apiClient) post(ctx context.Context, path string, body, response any) error {
	return c.send(ctx, http.MethodPost, path, body, response)
}

func (c *apiClient) put(ctx context.Context, path string, body, response any) error {
	return c.send(ctx, http.MethodPut, path, body, response)
}

func (c *apiClient) send(ctx context.Context, method, path string, body, response any) error {
	var requestBody io.Reader

	if body != nil {
		encodedBody, err := json.Marshal(body)
		if err != nil {
			return fmt.Errorf("encode request to %s: %w", path, err)
		}

		requestBody = bytes.NewReader(encodedBody)
	}

	request, err := http.NewRequestWithContext(ctx, method, c.baseURL+"/api/v1"+path, requestBody)
	if err != nil {
		return err
	}

	request.Header.Set("Content-Type", "application/json")

	if c.token != "" {
		request.Header.Set("Authorization", "Bearer "+c.token)
	}

	httpResponse, err := c.httpClient.Do(request)
	if err != nil {
		return err
	}
	defer func() { _ = httpResponse.Body.Close() }()

	responseBody, err := io.ReadAll(httpResponse.Body)
	if err != nil {
		return fmt.Errorf("read response from %s: %w", path, err)
	}

	if httpResponse.StatusCode >= http.StatusBadRequest {
		return &apiError{httpResponse.StatusCode, readRefusalReason(httpResponse.StatusCode, responseBody)}
	}

	if response == nil {
		return nil
	}

	if err := json.Unmarshal(responseBody, response); err != nil {
		return fmt.Errorf("decode response from %s: %w", path, err)
	}

	return nil
}

func readRefusalReason(statusCode int, responseBody []byte) string {
	var refusal struct {
		Detail string `json:"detail"`
	}
	if json.Unmarshal(responseBody, &refusal) == nil && refusal.Detail != "" {
		return refusal.Detail
	}

	if text := strings.TrimSpace(string(responseBody)); text != "" {
		return fmt.Sprintf("HTTP %d: %s", statusCode, text)
	}

	return fmt.Sprintf("HTTP %d", statusCode)
}

func isAPIRefusal(err error) bool {
	var refusal *apiError

	return errors.As(err, &refusal)
}
