package neonretry_test

import (
	"bytes"
	"compress/gzip"
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/kenchan0130/terraform-provider-neon/internal/neon"
	"github.com/kenchan0130/terraform-provider-neon/internal/neonerror"
	"github.com/kenchan0130/terraform-provider-neon/internal/neonretry"
)

// staticRoundTripper returns a fixed response for every request, echoing
// transportErr instead if set.
type staticRoundTripper struct {
	resp *http.Response
	err  error
}

func (rt *staticRoundTripper) RoundTrip(_ *http.Request) (*http.Response, error) {
	return rt.resp, rt.err
}

func newResponse(statusCode int, contentType string, body string) *http.Response {
	header := http.Header{}
	if contentType != "" {
		header.Set("Content-Type", contentType)
	}
	return &http.Response{
		StatusCode: statusCode,
		Header:     header,
		Body:       io.NopCloser(strings.NewReader(body)),
	}
}

func doRoundTrip(t *testing.T, resp *http.Response, transportErr error) (*http.Response, error) {
	t.Helper()
	rt := neonretry.NewErrorBodyRoundTripper(&staticRoundTripper{resp: resp, err: transportErr})
	req := httptest.NewRequest(http.MethodGet, "https://neon.example.com/api/v2/projects/foo", nil)
	return rt.RoundTrip(req)
}

func TestErrorBodyRoundTripper_NonJSONErrorRewritten(t *testing.T) {
	t.Parallel()

	html := `<html><body><h1>502 Bad Gateway</h1></body></html>`
	resp, err := doRoundTrip(t, newResponse(http.StatusBadGateway, "text/html", html), nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.StatusCode != http.StatusBadGateway {
		t.Fatalf("got status %d, want %d", resp.StatusCode, http.StatusBadGateway)
	}
	if ct := resp.Header.Get("Content-Type"); ct != "application/json" {
		t.Fatalf("got Content-Type %q, want application/json", ct)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("failed to read body: %v", err)
	}

	var decoded struct {
		Code    string `json:"code"`
		Message string `json:"message"`
	}
	if err := json.Unmarshal(body, &decoded); err != nil {
		t.Fatalf("rewritten body is not valid JSON: %v (body=%s)", err, body)
	}
	if !strings.Contains(decoded.Message, "502 Bad Gateway") {
		t.Fatalf("message %q does not contain original body snippet", decoded.Message)
	}
	if !strings.Contains(decoded.Message, "text/html") {
		t.Fatalf("message %q does not contain original content type", decoded.Message)
	}

	if cl := resp.Header.Get("Content-Length"); cl != "" {
		n, err := strconv.Atoi(cl)
		if err != nil {
			t.Fatalf("Content-Length %q is not numeric: %v", cl, err)
		}
		if n != len(body) {
			t.Fatalf("Content-Length %d does not match actual body length %d", n, len(body))
		}
	}
}

func TestErrorBodyRoundTripper_JSONErrorUntouched(t *testing.T) {
	t.Parallel()

	jsonBody := `{"code":"bad_request","message":"invalid input"}`
	resp, err := doRoundTrip(t, newResponse(http.StatusBadRequest, "application/json", jsonBody), nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("failed to read body: %v", err)
	}
	if string(body) != jsonBody {
		t.Fatalf("got body %q, want byte-identical %q", body, jsonBody)
	}
}

func TestErrorBodyRoundTripper_JSONErrorWithCharsetUntouched(t *testing.T) {
	t.Parallel()

	jsonBody := `{"code":"bad_request","message":"invalid input"}`
	resp, err := doRoundTrip(t, newResponse(http.StatusBadRequest, "application/json; charset=utf-8", jsonBody), nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("failed to read body: %v", err)
	}
	if string(body) != jsonBody {
		t.Fatalf("got body %q, want byte-identical %q", body, jsonBody)
	}
	if ct := resp.Header.Get("Content-Type"); ct != "application/json; charset=utf-8" {
		t.Fatalf("got Content-Type %q, want untouched", ct)
	}
}

// TestErrorBodyRoundTripper_StructuredSyntaxSuffixRewritten verifies that
// "application/problem+json" (and similar "+json" structured-syntax
// suffixes) are REWRITTEN, not passed through. ogen's generated decoders
// only accept the exact media type "application/json" on error branches
// (see the `case ct == "application/json":` checks in
// internal/neon/oas_response_decoders_gen.go) - a "+json" suffix would
// still fail validate.InvalidContentType if left untouched.
func TestErrorBodyRoundTripper_StructuredSyntaxSuffixRewritten(t *testing.T) {
	t.Parallel()

	jsonBody := `{"type":"about:blank","title":"Bad Request","status":400}`
	resp, err := doRoundTrip(t, newResponse(http.StatusBadRequest, "application/problem+json", jsonBody), nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if ct := resp.Header.Get("Content-Type"); ct != "application/json" {
		t.Fatalf("got Content-Type %q, want application/json (rewritten)", ct)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("failed to read body: %v", err)
	}

	var decoded struct {
		Code    string `json:"code"`
		Message string `json:"message"`
	}
	if err := json.Unmarshal(body, &decoded); err != nil {
		t.Fatalf("rewritten body is not valid JSON: %v (body=%s)", err, body)
	}
	if !strings.Contains(decoded.Message, "about:blank") {
		t.Fatalf("message %q does not contain original body snippet", decoded.Message)
	}
	if !strings.Contains(decoded.Message, "application/problem+json") {
		t.Fatalf("message %q does not mention the original content type", decoded.Message)
	}
}

func TestErrorBodyRoundTripper_SuccessResponseUntouched(t *testing.T) {
	t.Parallel()

	htmlBody := `<html><body>ok</body></html>`
	resp, err := doRoundTrip(t, newResponse(http.StatusOK, "text/html", htmlBody), nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("failed to read body: %v", err)
	}
	if string(body) != htmlBody {
		t.Fatalf("got body %q, want byte-identical %q", body, htmlBody)
	}
	if ct := resp.Header.Get("Content-Type"); ct != "text/html" {
		t.Fatalf("got Content-Type %q, want untouched", ct)
	}
}

func TestErrorBodyRoundTripper_TransportErrorUntouched(t *testing.T) {
	t.Parallel()

	transportErr := errors.New("connection refused")
	resp, err := doRoundTrip(t, nil, transportErr)
	if !errors.Is(err, transportErr) {
		t.Fatalf("got error %v, want %v", err, transportErr)
	}
	if resp != nil {
		t.Fatalf("expected nil response on transport error, got %v", resp)
	}
}

func TestErrorBodyRoundTripper_BodyTruncatedAt1KB(t *testing.T) {
	t.Parallel()

	longBody := strings.Repeat("a", 10_000)
	resp, err := doRoundTrip(t, newResponse(http.StatusInternalServerError, "text/plain", longBody), nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("failed to read body: %v", err)
	}

	var decoded struct {
		Code    string `json:"code"`
		Message string `json:"message"`
	}
	if err := json.Unmarshal(body, &decoded); err != nil {
		t.Fatalf("rewritten body is not valid JSON: %v", err)
	}

	// The message wraps the up-to-1KB snippet with a prefix and quoting;
	// it must not balloon to anywhere near the original 10KB body.
	if len(decoded.Message) > 1200 {
		t.Fatalf("message length %d suggests body was not truncated", len(decoded.Message))
	}
	// "text/plain" itself contains one 'a' (from "plain"), so the snippet
	// must account for no more than 1024 + 1 occurrences.
	if strings.Count(decoded.Message, "a") > 1025 {
		t.Fatalf("message contains more than 1024 bytes of original body content")
	}
}

func TestErrorBodyRoundTripper_BinaryGarbageSanitized(t *testing.T) {
	t.Parallel()

	binary := string([]byte{0x00, 0x01, 0xff, 0xfe, 'o', 'k', 0x07})
	resp, err := doRoundTrip(t, newResponse(http.StatusBadGateway, "application/octet-stream", binary), nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("failed to read body: %v", err)
	}

	var decoded struct {
		Code    string `json:"code"`
		Message string `json:"message"`
	}
	if err := json.Unmarshal(body, &decoded); err != nil {
		t.Fatalf("rewritten body is not valid JSON (garbage not sanitized): %v (body=%s)", err, body)
	}
	if !strings.Contains(decoded.Message, "ok") {
		t.Fatalf("message %q lost the readable portion of the body", decoded.Message)
	}
}

func TestErrorBodyRoundTripper_EmptyContentTypeTreatedAsUnknown(t *testing.T) {
	t.Parallel()

	resp, err := doRoundTrip(t, newResponse(http.StatusBadGateway, "", "some error text"), nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("failed to read body: %v", err)
	}

	var decoded struct {
		Code    string `json:"code"`
		Message string `json:"message"`
	}
	if err := json.Unmarshal(body, &decoded); err != nil {
		t.Fatalf("rewritten body is not valid JSON: %v", err)
	}
	if !strings.Contains(decoded.Message, "unknown") {
		t.Fatalf("message %q does not mention unknown content type", decoded.Message)
	}
	if !strings.Contains(decoded.Message, "some error text") {
		t.Fatalf("message %q does not contain original body", decoded.Message)
	}
}

func gzipCompress(t *testing.T, s string) string {
	t.Helper()
	var buf bytes.Buffer
	gw := gzip.NewWriter(&buf)
	if _, err := gw.Write([]byte(s)); err != nil {
		t.Fatalf("failed to gzip-compress test body: %v", err)
	}
	if err := gw.Close(); err != nil {
		t.Fatalf("failed to close gzip writer: %v", err)
	}
	return buf.String()
}

func newResponseWithEncoding(statusCode int, contentType, contentEncoding, body string) *http.Response {
	resp := newResponse(statusCode, contentType, body)
	if contentEncoding != "" {
		resp.Header.Set("Content-Encoding", contentEncoding)
	}
	return resp
}

func TestErrorBodyRoundTripper_GzipErrorBodyDecompressedAndReadable(t *testing.T) {
	t.Parallel()

	const original = `<html><body><h1>502 Bad Gateway</h1><p>upstream unavailable</p></body></html>`
	resp, err := doRoundTrip(t, newResponseWithEncoding(http.StatusBadGateway, "text/html", "gzip", gzipCompress(t, original)), nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if ce := resp.Header.Get("Content-Encoding"); ce != "" {
		t.Fatalf("got Content-Encoding %q, want cleared", ce)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("failed to read body: %v", err)
	}

	var decoded struct {
		Code    string `json:"code"`
		Message string `json:"message"`
	}
	if err := json.Unmarshal(body, &decoded); err != nil {
		t.Fatalf("rewritten body is not valid JSON: %v (body=%s)", err, body)
	}
	if !strings.Contains(decoded.Message, "502 Bad Gateway") {
		t.Fatalf("message %q does not contain decompressed, readable text", decoded.Message)
	}
	if !strings.Contains(decoded.Message, "upstream unavailable") {
		t.Fatalf("message %q does not contain decompressed, readable text", decoded.Message)
	}
}

func TestErrorBodyRoundTripper_MalformedGzipFallsBackToRawSnippet(t *testing.T) {
	t.Parallel()

	corrupt := "this is not a valid gzip stream"
	resp, err := doRoundTrip(t, newResponseWithEncoding(http.StatusBadGateway, "text/html", "gzip", corrupt), nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("failed to read body: %v", err)
	}

	var decoded struct {
		Code    string `json:"code"`
		Message string `json:"message"`
	}
	if err := json.Unmarshal(body, &decoded); err != nil {
		t.Fatalf("rewritten body is not valid JSON: %v (body=%s)", err, body)
	}
	if !strings.Contains(decoded.Message, "gzip") {
		t.Fatalf("message %q does not note the encoding for the fallback", decoded.Message)
	}
}

func TestErrorBodyRoundTripper_UnknownEncodingBodyOmitted(t *testing.T) {
	t.Parallel()

	// Simulate binary br-encoded content that must not be embedded raw.
	binary := string([]byte{0x1b, 0x9c, 0xff, 0x00, 0x01})
	resp, err := doRoundTrip(t, newResponseWithEncoding(http.StatusBadGateway, "text/html", "br", binary), nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("failed to read body: %v", err)
	}

	var decoded struct {
		Code    string `json:"code"`
		Message string `json:"message"`
	}
	if err := json.Unmarshal(body, &decoded); err != nil {
		t.Fatalf("rewritten body is not valid JSON: %v (body=%s)", err, body)
	}
	if !strings.Contains(decoded.Message, "br") {
		t.Fatalf("message %q does not note the unsupported encoding", decoded.Message)
	}
	for _, b := range []byte{0x1b, 0x9c, 0x00, 0x01} {
		if strings.IndexByte(decoded.Message, b) >= 0 {
			t.Fatalf("message %q embeds raw undecoded byte 0x%02x", decoded.Message, b)
		}
	}
}

func TestErrorBodyRoundTripper_EmptyBody(t *testing.T) {
	t.Parallel()

	resp, err := doRoundTrip(t, newResponse(http.StatusBadGateway, "text/html", ""), nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("failed to read body: %v", err)
	}

	var decoded struct {
		Code    string `json:"code"`
		Message string `json:"message"`
	}
	if err := json.Unmarshal(body, &decoded); err != nil {
		t.Fatalf("rewritten body is not valid JSON: %v (body=%s)", err, body)
	}
}

func TestErrorBodyRoundTripper_NilBody(t *testing.T) {
	t.Parallel()

	resp := newResponse(http.StatusBadGateway, "text/html", "")
	resp.Body = nil

	rt := neonretry.NewErrorBodyRoundTripper(&staticRoundTripper{resp: resp})
	req, err := http.NewRequest(http.MethodHead, "https://neon.example.com/api/v2/projects/foo", nil)
	if err != nil {
		t.Fatalf("failed to build request: %v", err)
	}
	req.Method = http.MethodHead

	got, err := rt.RoundTrip(req)
	if err != nil {
		t.Fatalf("unexpected error for nil-body (HEAD-style) response: %v", err)
	}
	if got.Header.Get("Content-Type") != "application/json" {
		t.Fatalf("got Content-Type %q, want application/json", got.Header.Get("Content-Type"))
	}
}

// closeTrackingBody wraps an io.Reader and records whether Close was
// called, so we can verify the original response body is always closed
// even after we've replaced resp.Body with the synthesized payload.
type closeTrackingBody struct {
	io.Reader
	closed atomic.Bool
}

func (b *closeTrackingBody) Close() error {
	b.closed.Store(true)
	return nil
}

func TestErrorBodyRoundTripper_OriginalBodyClosed(t *testing.T) {
	t.Parallel()

	tracked := &closeTrackingBody{Reader: strings.NewReader("<html>oops</html>")}
	resp := &http.Response{
		StatusCode: http.StatusBadGateway,
		Header:     http.Header{"Content-Type": []string{"text/html"}},
		Body:       tracked,
	}

	rt := neonretry.NewErrorBodyRoundTripper(&staticRoundTripper{resp: resp})
	req := httptest.NewRequest(http.MethodGet, "https://neon.example.com/api/v2/projects/foo", nil)
	if _, err := rt.RoundTrip(req); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !tracked.closed.Load() {
		t.Fatalf("expected original response body to be closed")
	}
}

func TestErrorBodyRoundTripper_UnrelatedHeadersPreserved(t *testing.T) {
	t.Parallel()

	resp := newResponse(http.StatusBadGateway, "text/html", "<html>oops</html>")
	resp.Header.Set("X-Request-Id", "abc-123")
	resp.Header.Set("Retry-After", "30")

	got, err := doRoundTrip(t, resp, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got.Header.Get("X-Request-Id") != "abc-123" {
		t.Fatalf("X-Request-Id header was not preserved: %q", got.Header.Get("X-Request-Id"))
	}
	if got.Header.Get("Retry-After") != "30" {
		t.Fatalf("Retry-After header was not preserved: %q", got.Header.Get("Retry-After"))
	}
}

func TestNewHTTPClient_ShallowCopyDoesNotMutateCallerClient(t *testing.T) {
	t.Parallel()

	original := &http.Client{Transport: http.DefaultTransport}
	origTransport := original.Transport

	_ = neonretry.NewHTTPClient(original, fastTestConfig(), nil)

	if original.Transport != origTransport {
		t.Fatalf("NewHTTPClient mutated the caller's http.Client.Transport in place")
	}
}

func TestNewHTTPClient_RetryExhaustionPreservesRewrittenErrorAndAttemptCount(t *testing.T) {
	t.Parallel()

	var requestCount atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		requestCount.Add(1)
		w.Header().Set("Content-Type", "text/html")
		w.WriteHeader(http.StatusBadGateway)
		_, _ = w.Write([]byte(`<html><body>502 Bad Gateway</body></html>`))
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

	// RetryMax=2 means 3 total attempts (1 initial + 2 retries); 502 is
	// retried by retryablehttp's default policy on every attempt, so the
	// rewritten Content-Type/status must be preserved on the final,
	// exhausted response.
	if got := requestCount.Load(); got != 3 {
		t.Fatalf("got %d requests, want 3", got)
	}
	if resp.StatusCode != http.StatusBadGateway {
		t.Fatalf("got status %d, want %d", resp.StatusCode, http.StatusBadGateway)
	}
	if ct := resp.Header.Get("Content-Type"); ct != "application/json" {
		t.Fatalf("got Content-Type %q, want application/json (rewritten)", ct)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("failed to read body: %v", err)
	}

	var decoded struct {
		Code    string `json:"code"`
		Message string `json:"message"`
	}
	if err := json.Unmarshal(body, &decoded); err != nil {
		t.Fatalf("rewritten body is not valid JSON: %v (body=%s)", err, body)
	}
	if !strings.Contains(decoded.Message, "502 Bad Gateway") {
		t.Fatalf("message %q does not contain the original body snippet", decoded.Message)
	}
}

// Integration test: a full neon.Client talking to an httptest server that
// returns an HTML 502 must surface the HTML snippet through
// *neon.GeneralErrorStatusCode and neonerror.Detail, instead of failing
// content-type validation and losing the body.
func TestNewHTTPClient_NonJSONErrorSurfacedThroughTypedError(t *testing.T) {
	t.Parallel()

	const htmlBody = `<html><body><h1>502 Bad Gateway</h1><p>nginx</p></body></html>`
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		w.WriteHeader(http.StatusBadGateway)
		_, _ = w.Write([]byte(htmlBody))
	}))
	defer server.Close()

	cfg := fastTestConfig()
	cfg.RetryMax = 0
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
	if statusErr.StatusCode != http.StatusBadGateway {
		t.Fatalf("got status code %d, want %d", statusErr.StatusCode, http.StatusBadGateway)
	}
	if !strings.Contains(statusErr.Response.Message, "502 Bad Gateway") {
		t.Fatalf("got message %q, want it to contain the HTML snippet", statusErr.Response.Message)
	}

	detail := neonerror.Detail(err)
	if !strings.Contains(detail, "502 Bad Gateway") {
		t.Fatalf("neonerror.Detail(err) = %q, want it to contain the HTML snippet", detail)
	}
}
