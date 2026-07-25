package smartlyq

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
)

func TestBearerAndDefaultHeaders(t *testing.T) {
	var gotAuth, gotUA, gotAccept string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotAuth = r.Header.Get("Authorization")
		gotUA = r.Header.Get("User-Agent")
		gotAccept = r.Header.Get("Accept")
		_, _ = w.Write([]byte(`{"success":true,"data":{}}`))
	}))
	defer srv.Close()

	c := NewClient("sqk_test_xxxxxxxxxxxx", WithBaseURL(srv.URL))
	if _, err := c.Account.GetMe(context.Background(), nil); err != nil {
		t.Fatalf("request failed: %v", err)
	}
	if gotAuth != "Bearer sqk_test_xxxxxxxxxxxx" {
		t.Errorf("Authorization = %q, want bearer key", gotAuth)
	}
	if gotUA != "smartlyq-go" {
		t.Errorf("User-Agent = %q, want %q", gotUA, "smartlyq-go")
	}
	if gotAccept != "application/json" {
		t.Errorf("Accept = %q, want application/json", gotAccept)
	}
}

func TestAPIErrorParsing(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusPaymentRequired)
		_, _ = w.Write([]byte(`{"success":false,"error":{"code":"insufficient_credits","message":"Not enough credits"},"meta":{"request_id":"req_abc123"}}`))
	}))
	defer srv.Close()

	c := NewClient("sqk_test_xxxxxxxxxxxx", WithBaseURL(srv.URL), WithMaxRetries(0))
	_, err := c.Images.Generate(context.Background(), map[string]any{"prompt": "x"}, nil)
	if err == nil {
		t.Fatal("expected an error")
	}
	var apiErr *APIError
	if !errors.As(err, &apiErr) {
		t.Fatalf("error is %T, want *APIError", err)
	}
	if apiErr.StatusCode != http.StatusPaymentRequired {
		t.Errorf("StatusCode = %d, want 402", apiErr.StatusCode)
	}
	if apiErr.Code != "insufficient_credits" {
		t.Errorf("Code = %q, want insufficient_credits", apiErr.Code)
	}
	if apiErr.Message != "Not enough credits" {
		t.Errorf("Message = %q, want Not enough credits", apiErr.Message)
	}
	if apiErr.RequestID != "req_abc123" {
		t.Errorf("RequestID = %q, want req_abc123", apiErr.RequestID)
	}
	if apiErr.Error() == "" {
		t.Error("Error() returned an empty string")
	}
}

func TestRetryOn429ThenSuccess(t *testing.T) {
	var calls atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if calls.Add(1) == 1 {
			w.Header().Set("Retry-After", "0")
			w.WriteHeader(http.StatusTooManyRequests)
			_, _ = w.Write([]byte(`{"success":false,"error":{"code":"rate_limited","message":"Too many requests"}}`))
			return
		}
		_, _ = w.Write([]byte(`{"success":true,"data":{"ok":true}}`))
	}))
	defer srv.Close()

	c := NewClient("sqk_test_xxxxxxxxxxxx", WithBaseURL(srv.URL), WithMaxRetries(2))
	resp, err := c.Account.GetMe(context.Background(), nil)
	if err != nil {
		t.Fatalf("expected success after retry, got %v", err)
	}
	if !resp.Success {
		t.Error("Success = false, want true")
	}
	if got := calls.Load(); got != 2 {
		t.Errorf("server calls = %d, want 2", got)
	}
}

func TestNoRetryOn400(t *testing.T) {
	var calls atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte(`{"success":false,"error":{"code":"invalid_request","message":"Bad input"}}`))
	}))
	defer srv.Close()

	c := NewClient("sqk_test_xxxxxxxxxxxx", WithBaseURL(srv.URL), WithMaxRetries(2))
	_, err := c.Account.GetMe(context.Background(), nil)
	var apiErr *APIError
	if !errors.As(err, &apiErr) || apiErr.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected *APIError with status 400, got %v", err)
	}
	if got := calls.Load(); got != 1 {
		t.Errorf("server calls = %d, want 1 (no retry on 400)", got)
	}
}

func TestPerRequestHeaders(t *testing.T) {
	var gotProfile, gotIdem string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotProfile = r.Header.Get("X-Profile-Id")
		gotIdem = r.Header.Get("Idempotency-Key")
		_, _ = w.Write([]byte(`{"success":true,"data":{}}`))
	}))
	defer srv.Close()

	c := NewClient("sqk_test_xxxxxxxxxxxx", WithBaseURL(srv.URL))
	opts := &RequestOptions{ProfileID: "prof_123", IdempotencyKey: "idem_456"}
	if _, err := c.Social.CreatePost(context.Background(), map[string]any{"text": "hi"}, opts); err != nil {
		t.Fatalf("request failed: %v", err)
	}
	if gotProfile != "prof_123" {
		t.Errorf("X-Profile-Id = %q, want prof_123", gotProfile)
	}
	if gotIdem != "idem_456" {
		t.Errorf("Idempotency-Key = %q, want idem_456", gotIdem)
	}
}
