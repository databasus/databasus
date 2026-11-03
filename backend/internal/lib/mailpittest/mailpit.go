package mailpittest

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

const requestTimeout = 5 * time.Second

type Message struct {
	ID        string    `json:"ID"`
	From      Address   `json:"From"`
	To        []Address `json:"To"`
	Subject   string    `json:"Subject"`
	MessageID string    `json:"MessageID"`
	Snippet   string    `json:"Snippet"`
}

type Address struct {
	Name    string `json:"Name"`
	Address string `json:"Address"`
}

type messagesResponse struct {
	Messages []Message `json:"messages"`
}

// Client queries a Mailpit instance's HTTP API to assert what mail a test delivered.
type Client struct {
	baseURL string
}

// NewClient builds a Mailpit API client for httpEndpoint (host:port of the container's HTTP port).
func NewClient(httpEndpoint string) *Client {
	return &Client{baseURL: "http://" + httpEndpoint}
}

func (c *Client) FetchMessages(ctx context.Context) ([]Message, error) {
	ctx, cancel := context.WithTimeout(ctx, requestTimeout)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.baseURL+"/api/v1/messages", nil)
	if err != nil {
		return nil, err
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("mailpit fetch: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("mailpit fetch returned %s: %s", resp.Status, string(body))
	}

	var payload messagesResponse
	if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
		return nil, fmt.Errorf("mailpit decode: %w", err)
	}

	return payload.Messages, nil
}
