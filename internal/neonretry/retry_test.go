package neonretry_test

import (
	"context"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/kenchan0130/terraform-provider-neon/internal/neon"
	"github.com/kenchan0130/terraform-provider-neon/internal/neonretry"
)

func TestCheckRetry(t *testing.T) {
	tests := []struct {
		name       string
		statusCode int
		wantRetry  bool
	}{
		{"423 locked retries", http.StatusLocked, true},
		{"429 too many requests retries", http.StatusTooManyRequests, true},
		{"500 internal server error retries", http.StatusInternalServerError, true},
		{"502 bad gateway retries", http.StatusBadGateway, true},
		{"404 not found does not retry", http.StatusNotFound, false},
		{"200 ok does not retry", http.StatusOK, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resp := &http.Response{StatusCode: tt.statusCode}
			retry, err := neonretry.CheckRetry(context.Background(), resp, nil)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if retry != tt.wantRetry {
				t.Fatalf("CheckRetry(%d) = %v, want %v", tt.statusCode, retry, tt.wantRetry)
			}
		})
	}

	t.Run("nil response with transport error delegates", func(t *testing.T) {
		transportErr := errors.New("connection refused")
		retry, err := neonretry.CheckRetry(context.Background(), nil, transportErr)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !retry {
			t.Fatalf("expected retry=true for nil response with transport error")
		}
	})

	t.Run("cancelled context does not retry and returns context error", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		cancel()

		resp := &http.Response{StatusCode: http.StatusLocked}
		retry, err := neonretry.CheckRetry(ctx, resp, nil)
		if retry {
			t.Fatalf("expected retry=false for cancelled context")
		}
		if !errors.Is(err, context.Canceled) {
			t.Fatalf("expected returned error to be context.Canceled, got: %v", err)
		}
	})
}

func fastTestConfig() neonretry.Config {
	return neonretry.Config{
		RetryMax:     3,
		RetryWaitMin: time.Millisecond,
		RetryWaitMax: 5 * time.Millisecond,
	}
}

func TestNewHTTPClient_RetriesLocked(t *testing.T) {
	const wantBody = `{"name":"test-endpoint","branch_id":"br-test-001","type":"read_write"}`

	requestCount := 0
	var seenBodies []string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestCount++

		body, err := io.ReadAll(r.Body)
		if err != nil {
			t.Errorf("failed to read request body on attempt %d: %v", requestCount, err)
		}
		seenBodies = append(seenBodies, string(body))

		if requestCount <= 2 {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusLocked)
			_, _ = w.Write([]byte(`{"code":"locked","message":"project is locked"}`))
			return
		}
		w.WriteHeader(http.StatusCreated)
	}))
	defer server.Close()

	client := neonretry.NewHTTPClient(nil, fastTestConfig(), nil)

	resp, err := client.Post(server.URL, "application/json", strings.NewReader(wantBody))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("got status %d, want %d", resp.StatusCode, http.StatusCreated)
	}
	if requestCount != 3 {
		t.Fatalf("got %d requests, want 3", requestCount)
	}

	if len(seenBodies) != 3 {
		t.Fatalf("got %d recorded bodies, want 3", len(seenBodies))
	}
	for i, body := range seenBodies {
		if body != wantBody {
			t.Fatalf("attempt %d: got body %q, want %q", i+1, body, wantBody)
		}
	}
}

func TestNewHTTPClient_PassthroughOnExhaustion(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusLocked)
		_, _ = w.Write([]byte(`{"code":"locked","message":"project is locked"}`))
	}))
	defer server.Close()

	cfg := fastTestConfig()
	cfg.RetryMax = 2
	client := neonretry.NewHTTPClient(nil, cfg, nil)

	resp, err := client.Get(server.URL)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusLocked {
		t.Fatalf("got status %d, want %d", resp.StatusCode, http.StatusLocked)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("failed to read body: %v", err)
	}
	if len(body) == 0 {
		t.Fatalf("expected non-empty body")
	}
}

type testSecuritySource struct{}

func (testSecuritySource) BearerAuth(_ context.Context, _ neon.OperationName) (neon.BearerAuth, error) {
	return neon.BearerAuth{Token: "test-api-key"}, nil
}

func (testSecuritySource) CookieAuth(_ context.Context, _ neon.OperationName) (neon.CookieAuth, error) {
	return neon.CookieAuth{}, nil
}

func (testSecuritySource) TokenCookieAuth(_ context.Context, _ neon.OperationName) (neon.TokenCookieAuth, error) {
	return neon.TokenCookieAuth{}, nil
}

func TestNewHTTPClient_TypedErrorDecodeOnExhaustion(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusLocked)
		_, _ = w.Write([]byte(`{"code":"locked","message":"project is locked","request_id":"req-123"}`))
	}))
	defer server.Close()

	cfg := fastTestConfig()
	cfg.RetryMax = 1
	httpClient := neonretry.NewHTTPClient(nil, cfg, nil)

	client, err := neon.NewClient(server.URL, testSecuritySource{}, neon.WithClient(httpClient))
	if err != nil {
		t.Fatalf("failed to create neon client: %v", err)
	}

	_, err = client.GetProject(context.Background(), neon.GetProjectParams{ProjectID: "test-project"})
	if err == nil {
		t.Fatal("expected an error, got nil")
	}

	var statusErr *neon.GeneralErrorStatusCode
	if !errors.As(err, &statusErr) {
		t.Fatalf("expected error to be *neon.GeneralErrorStatusCode, got: %T (%v)", err, err)
	}

	if statusErr.StatusCode != http.StatusLocked {
		t.Fatalf("got status code %d, want %d", statusErr.StatusCode, http.StatusLocked)
	}
	if statusErr.Response.Message != "project is locked" {
		t.Fatalf("got message %q, want %q", statusErr.Response.Message, "project is locked")
	}
	requestID, ok := statusErr.Response.RequestID.Get()
	if !ok || requestID != "req-123" {
		t.Fatalf("got request_id %q (ok=%v), want %q", requestID, ok, "req-123")
	}
}
