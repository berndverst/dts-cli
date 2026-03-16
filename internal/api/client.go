package api

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"strings"
	"time"

	"github.com/Azure/durabletask-cli/internal/auth"
)

// Client is the DTS Backend API HTTP client.
type Client struct {
	baseURL string
	taskHub string
	token   *auth.TokenProvider
	http    *http.Client
}

// NewClient creates a new DTS API client.
func NewClient(baseURL, taskHub string, tokenProvider *auth.TokenProvider) *Client {
	baseURL = strings.TrimRight(baseURL, "/")
	transport := &http.Transport{
		MaxIdleConns:        100,
		MaxIdleConnsPerHost: 10,
		IdleConnTimeout:     90 * time.Second,
		DialContext: (&net.Dialer{
			Timeout:   10 * time.Second,
			KeepAlive: 30 * time.Second,
		}).DialContext,
		ForceAttemptHTTP2:  true,
		DisableCompression: false,
	}
	return &Client{
		baseURL: baseURL,
		taskHub: taskHub,
		token:   tokenProvider,
		http: &http.Client{
			Timeout:   30 * time.Second,
			Transport: transport,
		},
	}
}

// SetTaskHub changes the active task hub.
func (c *Client) SetTaskHub(taskHub string) {
	c.taskHub = taskHub
}

// BaseURL returns the configured base URL.
func (c *Client) BaseURL() string {
	return c.baseURL
}

// TaskHub returns the configured task hub name.
func (c *Client) TaskHub() string {
	return c.taskHub
}

// doRequest executes an authenticated HTTP request to the DTS API.
func (c *Client) doRequest(ctx context.Context, method, path string, body interface{}) (*http.Response, error) {
	url := c.baseURL + path

	var bodyReader io.Reader
	if body != nil {
		data, err := json.Marshal(body)
		if err != nil {
			return nil, fmt.Errorf("marshaling request body: %w", err)
		}
		bodyReader = bytes.NewReader(data)
	}

	req, err := http.NewRequestWithContext(ctx, method, url, bodyReader)
	if err != nil {
		return nil, fmt.Errorf("creating request: %w", err)
	}

	// Set required headers
	req.Header.Set("x-taskhub", strings.ToLower(c.taskHub))
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	// Acquire and set auth token
	if c.token != nil {
		token, err := c.token.GetToken(ctx)
		if err != nil {
			return nil, fmt.Errorf("acquiring auth token: %w", err)
		}
		req.Header.Set("Authorization", "Bearer "+token)
	}

	resp, err := c.http.Do(req)
	if err != nil {
		return nil, fmt.Errorf("executing request: %w", err)
	}

	return resp, nil
}

// doJSON performs a request and decodes the JSON response into result.
func (c *Client) doJSON(ctx context.Context, method, path string, body interface{}, result interface{}) error {
	resp, err := c.doRequest(ctx, method, path, body)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if err := checkResponse(resp); err != nil {
		return err
	}

	if result != nil && resp.StatusCode != http.StatusNoContent {
		if err := json.NewDecoder(resp.Body).Decode(result); err != nil {
			return fmt.Errorf("decoding response: %w", err)
		}
	}
	return nil
}

// doNoContent performs a request that expects no response body.
func (c *Client) doNoContent(ctx context.Context, method, path string, body interface{}) error {
	resp, err := c.doRequest(ctx, method, path, body)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	return checkResponse(resp)
}

// doRaw performs a request and returns the raw response body as a string.
func (c *Client) doRaw(ctx context.Context, method, path string, body interface{}) (string, int, error) {
	resp, err := c.doRequest(ctx, method, path, body)
	if err != nil {
		return "", 0, err
	}
	defer resp.Body.Close()

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", resp.StatusCode, fmt.Errorf("reading response: %w", err)
	}
	return string(data), resp.StatusCode, nil
}

// APIError represents an error from the DTS API.
type APIError struct {
	StatusCode int
	Status     string
	Body       string
}

func (e *APIError) Error() string {
	switch e.StatusCode {
	case http.StatusForbidden:
		return "Permission denied. Check your RBAC role assignment."
	case http.StatusNotFound:
		return "Resource not found. It may have been purged."
	case http.StatusConflict:
		return fmt.Sprintf("Conflict: %s", e.Body)
	default:
		return fmt.Sprintf("API error %d: %s", e.StatusCode, e.Body)
	}
}

func checkResponse(resp *http.Response) error {
	if resp.StatusCode >= 200 && resp.StatusCode < 300 {
		return nil
	}
	body, _ := io.ReadAll(resp.Body)
	return &APIError{
		StatusCode: resp.StatusCode,
		Status:     resp.Status,
		Body:       string(body),
	}
}

// Ping checks connectivity to the DTS backend.
func (c *Client) Ping(ctx context.Context) error {
	return c.doNoContent(ctx, http.MethodGet, "/v1/taskhubs/ping", nil)
}
