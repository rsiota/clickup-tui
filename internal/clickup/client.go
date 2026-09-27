package clickup

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

const defaultBaseURL = "https://api.clickup.com/api"

type Client struct {
	baseURL    string
	token      string
	httpClient *http.Client
}

type APIError struct {
	Status  int
	Code    string
	Message string
}

func (e *APIError) Error() string {
	if e.Message != "" {
		return fmt.Sprintf("clickup: %s (HTTP %d)", e.Message, e.Status)
	}
	return fmt.Sprintf("clickup: HTTP %d", e.Status)
}

func New(token string) *Client {
	return &Client{
		baseURL: defaultBaseURL,
		token:   token,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

func (c *Client) do(ctx context.Context, method, path string, body, result any) error {
	var payload []byte
	if body != nil {
		var err error
		payload, err = json.Marshal(body)
		if err != nil {
			return fmt.Errorf("clickup: marshal: %w", err)
		}
	}

	var last error
	for attempt := 0; attempt < 3; attempt++ {
		if attempt > 0 {
			wait := time.Duration(attempt) * time.Second
			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-time.After(wait):
			}
		}
		err := c.doOnce(ctx, method, path, payload, result)
		if err == nil {
			return nil
		}
		last = err
		if apiErr, ok := err.(*APIError); ok && (apiErr.Status == 429 || apiErr.Status >= 500) {
			continue
		}
		return err
	}
	return last
}

func (c *Client) doOnce(ctx context.Context, method, path string, payload []byte, result any) error {
	var rdr io.Reader
	if payload != nil {
		rdr = bytes.NewReader(payload)
	}
	req, err := http.NewRequestWithContext(ctx, method, c.baseURL+path, rdr)
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", c.token)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", "clickup-tui")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("clickup: read body: %w", err)
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return parseAPIError(resp.StatusCode, data)
	}
	if result != nil && len(data) > 0 {
		if err := json.Unmarshal(data, result); err != nil {
			return fmt.Errorf("clickup: decode: %w", err)
		}
	}
	return nil
}

func parseAPIError(status int, body []byte) error {
	var raw struct {
		Err     string `json:"err"`
		ECODE   string `json:"ECODE"`
		Message string `json:"message"`
	}
	_ = json.Unmarshal(body, &raw)
	msg := raw.Err
	if msg == "" {
		msg = raw.Message
	}
	if msg == "" {
		msg = strings.TrimSpace(string(body))
	}
	return &APIError{Status: status, Code: raw.ECODE, Message: msg}
}

func (c *Client) GetUser(ctx context.Context) (*User, error) {
	var resp UserResponse
	if err := c.do(ctx, http.MethodGet, "/v2/user", nil, &resp); err != nil {
		return nil, err
	}
	return &resp.User, nil
}

func (c *Client) ListWorkspaces(ctx context.Context) ([]Workspace, error) {
	var resp WorkspacesResponse
	if err := c.do(ctx, http.MethodGet, "/v2/team", nil, &resp); err != nil {
		return nil, err
	}
	return resp.Teams, nil
}

func LooksCustomID(id string) bool {
	return strings.Contains(id, "-")
}

func taskScopeQuery(id, workspace string) string {
	if !LooksCustomID(id) || workspace == "" {
		return ""
	}
	q := url.Values{}
	q.Set("custom_task_ids", "true")
	q.Set("team_id", workspace)
	return "?" + q.Encode()
}
