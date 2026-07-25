// Package smartlyq is the official Go SDK for the SmartlyQ API.
//
// The HTTP core in this file is hand-written; the resource surface in
// resources_gen.go is generated from openapi.json by scripts/generate.
package smartlyq

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"math/rand"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"
)

const (
	defaultBaseURL    = "https://api.smartlyq.com/v1"
	defaultTimeout    = 60 * time.Second
	defaultMaxRetries = 2
	userAgent         = "smartlyq-go"
)

// Client is the SmartlyQ API client. Construct it with NewClient and access
// endpoints through its resource fields, e.g. client.Social.CreatePost(...).
type Client struct {
	apiKey     string
	baseURL    string
	httpClient *http.Client
	timeout    time.Duration
	maxRetries int

	resources
}

// Option configures a Client.
type Option func(*Client)

// WithBaseURL overrides the API base URL (default https://api.smartlyq.com/v1).
func WithBaseURL(baseURL string) Option {
	return func(c *Client) { c.baseURL = strings.TrimRight(baseURL, "/") }
}

// WithHTTPClient sets a custom *http.Client for all requests.
func WithHTTPClient(httpClient *http.Client) Option {
	return func(c *Client) { c.httpClient = httpClient }
}

// WithTimeout sets the per-request timeout (default 60s). Zero disables it.
func WithTimeout(timeout time.Duration) Option {
	return func(c *Client) { c.timeout = timeout }
}

// WithMaxRetries sets the number of automatic retries on 429/5xx responses
// (default 2). Zero disables retries.
func WithMaxRetries(maxRetries int) Option {
	return func(c *Client) { c.maxRetries = maxRetries }
}

// NewClient returns a SmartlyQ API client. Pass your API key
// ("sqk_live_..." or "sqk_test_..."); if apiKey is empty the
// SMARTLYQ_API_KEY environment variable is used.
func NewClient(apiKey string, opts ...Option) *Client {
	if apiKey == "" {
		apiKey = os.Getenv("SMARTLYQ_API_KEY")
	}
	c := &Client{
		apiKey:     apiKey,
		baseURL:    defaultBaseURL,
		httpClient: http.DefaultClient,
		timeout:    defaultTimeout,
		maxRetries: defaultMaxRetries,
	}
	for _, opt := range opts {
		opt(c)
	}
	c.initResources()
	return c
}

// RequestOptions carries per-request options, passed as the last argument of
// every method. A nil *RequestOptions is valid and means defaults.
type RequestOptions struct {
	// ProfileID acts on behalf of a managed Profile (sent as X-Profile-Id).
	ProfileID string
	// IdempotencyKey makes write requests safe to retry (sent as Idempotency-Key).
	IdempotencyKey string
	// Headers are extra headers merged into the request.
	Headers map[string]string
}

// Envelope is the standard SmartlyQ response envelope. Data is left raw so
// callers can unmarshal it into their own types with UnmarshalData.
type Envelope struct {
	Success    bool            `json:"success"`
	Data       json.RawMessage `json:"data,omitempty"`
	Usage      *Usage          `json:"usage,omitempty"`
	Meta       *Meta           `json:"meta,omitempty"`
	Pagination *Pagination     `json:"pagination,omitempty"`
}

// Response is an alias for Envelope.
type Response = Envelope

// UnmarshalData unmarshals the envelope's data payload into v.
func (e *Envelope) UnmarshalData(v any) error {
	if len(e.Data) == 0 {
		return fmt.Errorf("smartlyq: envelope has no data")
	}
	return json.Unmarshal(e.Data, v)
}

// Usage reports metered usage charged for a request.
type Usage struct {
	Units            int    `json:"units,omitempty"`
	Cost             string `json:"cost,omitempty"`
	BalanceRemaining string `json:"balance_remaining,omitempty"`
}

// Meta carries response metadata.
type Meta struct {
	RequestID string `json:"request_id,omitempty"`
	Timestamp string `json:"timestamp,omitempty"`
}

// Pagination describes list pagination.
type Pagination struct {
	Page    int `json:"page,omitempty"`
	PerPage int `json:"per_page,omitempty"`
	Total   int `json:"total,omitempty"`
	Pages   int `json:"pages,omitempty"`
}

// APIError is returned for any non-2xx API response.
type APIError struct {
	StatusCode int
	Code       string
	Message    string
	RequestID  string
}

// Error implements the error interface.
func (e *APIError) Error() string {
	msg := fmt.Sprintf("smartlyq: HTTP %d", e.StatusCode)
	if e.Code != "" {
		msg += " " + e.Code
	}
	if e.Message != "" {
		msg += ": " + e.Message
	}
	if e.RequestID != "" {
		msg += " (request_id: " + e.RequestID + ")"
	}
	return msg
}

// errorEnvelope mirrors the API error shape:
// {"success":false,"error":{"code","message"},"meta":{"request_id"}}
type errorEnvelope struct {
	Error struct {
		Code    string `json:"code"`
		Message string `json:"message"`
	} `json:"error"`
	Meta struct {
		RequestID string `json:"request_id"`
	} `json:"meta"`
}

func isRetryable(status int) bool {
	switch status {
	case http.StatusTooManyRequests,
		http.StatusInternalServerError,
		http.StatusBadGateway,
		http.StatusServiceUnavailable,
		http.StatusGatewayTimeout:
		return true
	}
	return false
}

// do executes one API call with auth, retries, and envelope decoding. All
// generated resource methods funnel through here.
func (c *Client) do(ctx context.Context, method, path string, query map[string]string, body map[string]any, opts *RequestOptions) (*Envelope, error) {
	endpoint := c.baseURL + path
	if len(query) > 0 {
		values := url.Values{}
		for key, value := range query {
			values.Set(key, value)
		}
		endpoint += "?" + values.Encode()
	}

	var payload []byte
	if body != nil {
		var err error
		payload, err = json.Marshal(body)
		if err != nil {
			return nil, fmt.Errorf("smartlyq: encoding request body: %w", err)
		}
	}

	if c.timeout > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, c.timeout)
		defer cancel()
	}

	var lastErr error
	for attempt := 0; attempt <= c.maxRetries; attempt++ {
		var reader io.Reader
		if payload != nil {
			reader = bytes.NewReader(payload)
		}
		req, err := http.NewRequestWithContext(ctx, method, endpoint, reader)
		if err != nil {
			return nil, fmt.Errorf("smartlyq: building request: %w", err)
		}
		req.Header.Set("Authorization", "Bearer "+c.apiKey)
		req.Header.Set("Accept", "application/json")
		req.Header.Set("User-Agent", userAgent)
		if payload != nil {
			req.Header.Set("Content-Type", "application/json")
		}
		if opts != nil {
			for key, value := range opts.Headers {
				req.Header.Set(key, value)
			}
			if opts.ProfileID != "" {
				req.Header.Set("X-Profile-Id", opts.ProfileID)
			}
			if opts.IdempotencyKey != "" {
				req.Header.Set("Idempotency-Key", opts.IdempotencyKey)
			}
		}

		resp, err := c.httpClient.Do(req)
		if err != nil {
			lastErr = err
			if ctx.Err() != nil || attempt >= c.maxRetries {
				return nil, fmt.Errorf("smartlyq: request failed: %w", err)
			}
			if err := sleepContext(ctx, backoff(attempt)); err != nil {
				return nil, fmt.Errorf("smartlyq: request failed: %w", lastErr)
			}
			continue
		}

		respBody, readErr := io.ReadAll(resp.Body)
		resp.Body.Close()
		if readErr != nil {
			return nil, fmt.Errorf("smartlyq: reading response: %w", readErr)
		}

		if resp.StatusCode >= 200 && resp.StatusCode < 300 {
			if resp.StatusCode == http.StatusNoContent || len(respBody) == 0 {
				return &Envelope{Success: true}, nil
			}
			var envelope Envelope
			if err := json.Unmarshal(respBody, &envelope); err != nil {
				return nil, fmt.Errorf("smartlyq: decoding response: %w", err)
			}
			return &envelope, nil
		}

		if isRetryable(resp.StatusCode) && attempt < c.maxRetries {
			delay := backoff(attempt)
			if retryAfter := resp.Header.Get("Retry-After"); retryAfter != "" {
				if seconds, err := strconv.Atoi(retryAfter); err == nil && seconds >= 0 {
					delay = time.Duration(seconds) * time.Second
				}
			}
			if err := sleepContext(ctx, delay); err != nil {
				return nil, apiErrorFrom(resp.StatusCode, respBody)
			}
			continue
		}

		return nil, apiErrorFrom(resp.StatusCode, respBody)
	}

	return nil, fmt.Errorf("smartlyq: request failed: %w", lastErr)
}

func apiErrorFrom(status int, body []byte) *APIError {
	apiErr := &APIError{StatusCode: status, Message: fmt.Sprintf("HTTP %d", status)}
	var envelope errorEnvelope
	if err := json.Unmarshal(body, &envelope); err == nil {
		if envelope.Error.Message != "" {
			apiErr.Message = envelope.Error.Message
		}
		apiErr.Code = envelope.Error.Code
		apiErr.RequestID = envelope.Meta.RequestID
	}
	return apiErr
}

// backoff returns the exponential backoff delay for a retry attempt,
// with up to 25% jitter.
func backoff(attempt int) time.Duration {
	base := 500 * time.Millisecond * time.Duration(1<<attempt)
	jitter := time.Duration(rand.Int63n(int64(base)/4 + 1))
	return base + jitter
}

// sleepContext sleeps for d unless ctx is done first.
func sleepContext(ctx context.Context, d time.Duration) error {
	if d <= 0 {
		return nil
	}
	timer := time.NewTimer(d)
	defer timer.Stop()
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-timer.C:
		return nil
	}
}
