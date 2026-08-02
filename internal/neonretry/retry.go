// Package neonretry configures the retryablehttp client used by the Neon
// provider so that requests rejected with 423 Locked (Neon returns this
// while a project operation, e.g. a branch create, is still in progress)
// are retried like transport errors and 429/5xx responses.
package neonretry

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/hashicorp/go-retryablehttp"
)

// Config controls the retry behavior of the HTTP client used to talk to the
// Neon API.
type Config struct {
	RetryMax     int
	RetryWaitMin time.Duration
	RetryWaitMax time.Duration
}

// DefaultConfig returns the retry configuration used in production.
//
// RetryMax 6 gives ~61s budget (1+2+4+8+16+30) to outlast in-progress Neon
// operations behind 423; tradeoff: persistent 5xx surfaces after ~61s
// instead of ~15s.
func DefaultConfig() Config {
	return Config{
		RetryMax:     6,
		RetryWaitMin: 1 * time.Second,
		RetryWaitMax: 30 * time.Second,
	}
}

// CheckRetry decides whether a request should be retried. It delegates to
// retryablehttp.DefaultRetryPolicy first, which handles context
// cancellation, nil responses, and the standard 429/5xx retry rules. Only
// when the default policy declines to retry do we additionally retry on
// 423 Locked, which Neon returns while a project operation is in progress.
//
// Delegating first is required: it handles ctx cancellation, nil resp,
// 429/5xx. A 423-first branch would bypass cancellation and risk nil deref.
func CheckRetry(ctx context.Context, resp *http.Response, err error) (bool, error) {
	retry, cerr := retryablehttp.DefaultRetryPolicy(ctx, resp, err)
	if cerr != nil {
		return retry, fmt.Errorf("neonretry: default retry policy: %w", cerr)
	}
	if retry {
		return retry, nil
	}

	return resp != nil && resp.StatusCode == http.StatusLocked, nil
}

// NewHTTPClient builds a *http.Client that retries requests according to
// cfg, using CheckRetry to additionally retry on 423 Locked. If inner is
// non-nil, it is used as the underlying HTTPClient (e.g. for tests that
// inject a mock transport); otherwise retryablehttp's default HTTPClient is
// used.
func NewHTTPClient(inner *http.Client, cfg Config, logHook retryablehttp.RequestLogHook) *http.Client {
	c := retryablehttp.NewClient()
	c.Logger = nil
	c.RequestLogHook = logHook

	if inner != nil {
		c.HTTPClient = inner
	}

	c.RetryMax = cfg.RetryMax
	c.RetryWaitMin = cfg.RetryWaitMin
	c.RetryWaitMax = cfg.RetryWaitMax
	c.CheckRetry = CheckRetry

	// The default ErrorHandler discards the final response on exhaustion
	// and returns an opaque error, which would break ogen's typed
	// GeneralErrorStatusCode decoding (neonerror.IsNotFound etc.).
	// PassthroughErrorHandler returns the final response so behavior on
	// exhaustion matches the no-retry behavior.
	//
	// Note: retryablehttp.DefaultBackoff honors Retry-After only for
	// 429/503; Neon doesn't document Retry-After on 423.
	c.ErrorHandler = retryablehttp.PassthroughErrorHandler

	return c.StandardClient()
}
