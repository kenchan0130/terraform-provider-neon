package neonretry

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"mime"
	"net/http"
	"strconv"
	"strings"
)

// maxErrorBodySnippet caps how much of a non-JSON error body is captured
// into the synthesized GeneralError message, to avoid unbounded memory use
// on pathological (e.g. HTML) error pages.
const maxErrorBodySnippet = 1024

// errorBodyRoundTripper rewrites non-JSON error responses (status >= 400
// with a Content-Type that isn't JSON) into a minimal JSON payload shaped
// like the Neon API's GeneralError schema, so ogen's generated decoder
// (which requires application/json on error branches) can decode it instead
// of failing with validate.InvalidContentType and discarding the body.
type errorBodyRoundTripper struct {
	inner http.RoundTripper
}

// NewErrorBodyRoundTripper returns an http.RoundTripper that rewrites
// non-JSON error response bodies into valid GeneralError JSON so that
// ogen's generated decoder can surface the original content instead of
// failing content-type validation and losing the body. If inner is nil,
// http.DefaultTransport is used.
func NewErrorBodyRoundTripper(inner http.RoundTripper) http.RoundTripper {
	if inner == nil {
		inner = http.DefaultTransport
	}
	return &errorBodyRoundTripper{inner: inner}
}

func (rt *errorBodyRoundTripper) RoundTrip(req *http.Request) (*http.Response, error) {
	resp, err := rt.inner.RoundTrip(req)
	if err != nil || resp == nil {
		if err != nil {
			return resp, fmt.Errorf("neonretry: round trip: %w", err)
		}
		return resp, nil
	}

	if resp.StatusCode < http.StatusBadRequest || isJSONContentType(resp.Header.Get("Content-Type")) {
		return resp, nil
	}

	return rewriteAsGeneralError(resp)
}

// isJSONContentType reports whether ct identifies a JSON media type. It
// treats "application/json" and any "+json" structured syntax suffix
// (e.g. "application/problem+json") as JSON, matching the media types
// ogen's generated decoders accept. Content-Type parameters (e.g.
// "; charset=utf-8") are ignored. Malformed or empty Content-Type values
// are treated as non-JSON.
func isJSONContentType(ct string) bool {
	if ct == "" {
		return false
	}
	mediaType, _, err := mime.ParseMediaType(ct)
	if err != nil {
		return false
	}
	return mediaType == "application/json" || strings.HasSuffix(mediaType, "+json")
}

// rewriteAsGeneralError replaces resp's body with a synthesized
// GeneralError JSON payload carrying a snippet of the original body, and
// fixes up Content-Type/Content-Length accordingly. The original status
// code and other headers are preserved.
func rewriteAsGeneralError(resp *http.Response) (*http.Response, error) {
	origContentType := resp.Header.Get("Content-Type")
	if origContentType == "" {
		origContentType = "unknown"
	}

	var body []byte
	if resp.Body != nil {
		limited := io.LimitReader(resp.Body, maxErrorBodySnippet)
		b, readErr := io.ReadAll(limited)
		_ = resp.Body.Close()
		if readErr != nil {
			return nil, fmt.Errorf("neonretry: read error body: %w", readErr)
		}
		body = b
	}

	snippet := sanitizeSnippet(body)

	payload := struct {
		Code    string `json:"code"`
		Message string `json:"message"`
	}{
		Code:    "",
		Message: origContentType + " response: " + snippet,
	}

	encoded, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("neonretry: marshal synthesized error body: %w", err)
	}

	resp.Body = io.NopCloser(bytes.NewReader(encoded))
	resp.Header.Set("Content-Type", "application/json")
	resp.Header.Set("Content-Length", strconv.Itoa(len(encoded)))
	resp.ContentLength = int64(len(encoded))

	return resp, nil
}

// sanitizeSnippet converts raw (possibly binary/non-UTF8) response body
// bytes into a readable, quoted string suitable for embedding in a JSON
// error message. strconv.Quote escapes control characters and non-printable
// runes so the result stays on a single line and free of garbage bytes.
func sanitizeSnippet(body []byte) string {
	valid := strings.ToValidUTF8(string(body), "�")
	return strconv.Quote(strings.TrimSpace(valid))
}
