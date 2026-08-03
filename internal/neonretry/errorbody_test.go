package neonretry_test

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
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

func TestErrorBodyRoundTripper_StructuredSyntaxSuffixUntouched(t *testing.T) {
	t.Parallel()

	jsonBody := `{"code":"bad_request","message":"invalid input"}`
	resp, err := doRoundTrip(t, newResponse(http.StatusBadRequest, "application/problem+json", jsonBody), nil)
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
