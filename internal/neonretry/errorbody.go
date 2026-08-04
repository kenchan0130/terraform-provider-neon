package neonretry

import (
	"bytes"
	"compress/gzip"
	"encoding/json"
	"fmt"
	"io"
	"mime"
	"net/http"
	"strconv"
	"strings"
)

// maxErrorBodySnippet caps how much of a non-JSON error body (after
// decompression, if applicable) is captured into the synthesized
// GeneralError message, to avoid unbounded memory/CPU use on pathological
// (e.g. HTML, or maliciously compressed) error pages.
const maxErrorBodySnippet = 1024

// maxRawErrorBody caps how many raw (possibly compressed) bytes are read
// from an error response body before giving up. This bounds memory use
// independently of maxErrorBodySnippet, since compressed bytes can be much
// smaller than their decompressed form.
const maxRawErrorBody = 1 << 20 // 1MiB

// errorBodyRoundTripper rewrites non-JSON error responses (status >= 400
// with a Content-Type ogen's generated decoder would not accept) into a
// minimal JSON payload shaped like the Neon API's GeneralError schema, so
// ogen's generated decoder (which requires an exact "application/json"
// Content-Type on error branches) can decode it instead of failing with
// validate.InvalidContentType and discarding the body.
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
	if err != nil {
		return resp, fmt.Errorf("neonretry: round trip: %w", err)
	}
	if resp == nil {
		return resp, nil
	}

	if resp.StatusCode < http.StatusBadRequest || isOgenAcceptedJSON(resp.Header.Get("Content-Type")) {
		return resp, nil
	}

	return rewriteAsGeneralError(resp)
}

// CloseIdleConnections forwards to the inner transport's
// CloseIdleConnections method, if it implements one. http.Client and
// retryablehttp's client both discover this optional method via the same
// interface{ CloseIdleConnections() } type assertion, so without this the
// wrapper would silently break connection-pool cleanup.
func (rt *errorBodyRoundTripper) CloseIdleConnections() {
	type closeIdler interface {
		CloseIdleConnections()
	}
	if ci, ok := rt.inner.(closeIdler); ok {
		ci.CloseIdleConnections()
	}
}

// isOgenAcceptedJSON reports whether ct is the exact Content-Type ogen's
// generated decoders accept on error branches: media type
// "application/json" (parameters such as "; charset=utf-8" are ignored,
// matching mime.ParseMediaType's behavior, but the media type itself must
// match exactly). Notably, structured-syntax suffixes like
// "application/problem+json" are NOT accepted by the generated decoders
// and so must NOT be treated as JSON here. Malformed or empty Content-Type
// values are treated as non-JSON.
func isOgenAcceptedJSON(ct string) bool {
	if ct == "" {
		return false
	}
	mediaType, _, err := mime.ParseMediaType(ct)
	if err != nil {
		return false
	}
	return mediaType == "application/json"
}

// rewriteAsGeneralError replaces resp's body with a synthesized
// GeneralError JSON payload carrying a snippet of the original body, and
// fixes up Content-Type/Content-Length/Content-Encoding/TransferEncoding
// accordingly. The original status code and other headers are preserved.
func rewriteAsGeneralError(resp *http.Response) (*http.Response, error) {
	origContentType := resp.Header.Get("Content-Type")
	if origContentType == "" {
		origContentType = "unknown"
	}

	message, err := buildErrorMessage(resp, origContentType)
	if err != nil {
		return nil, err
	}

	payload := struct {
		Code    string `json:"code"`
		Message string `json:"message"`
	}{
		Code:    "",
		Message: message,
	}

	encoded, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("neonretry: marshal synthesized error body: %w", err)
	}

	resp.Body = io.NopCloser(bytes.NewReader(encoded))
	resp.Header.Set("Content-Type", "application/json")
	resp.Header.Set("Content-Length", strconv.Itoa(len(encoded)))
	resp.Header.Del("Content-Encoding")
	resp.ContentLength = int64(len(encoded))
	resp.Uncompressed = false
	resp.TransferEncoding = nil

	return resp, nil
}

// buildErrorMessage reads (and closes) resp.Body and produces the
// human-readable message embedded in the synthesized GeneralError.
//
// Content-Encoding is handled as follows:
//   - empty/"identity": the raw body is used directly.
//   - "gzip": the body is decompressed (bounded by maxRawErrorBody
//     compressed bytes and maxErrorBodySnippet decompressed bytes). If
//     decompression fails partway (corrupt/truncated gzip), the raw
//     (still-compressed) bytes already read are used as the snippet
//     instead, sanitized like any other binary content.
//   - anything else (br, deflate, zstd, ...): decoding is not attempted;
//     the body is omitted from the message and the encoding is noted
//     instead, so binary/garbled content is never embedded.
func buildErrorMessage(resp *http.Response, origContentType string) (string, error) {
	defer func() {
		if resp.Body != nil {
			_ = resp.Body.Close()
		}
	}()

	if resp.Body == nil || resp.Body == http.NoBody {
		return fmt.Sprintf("%s response: %s", origContentType, sanitizeSnippet(nil)), nil
	}

	encoding := strings.ToLower(strings.TrimSpace(resp.Header.Get("Content-Encoding")))

	switch encoding {
	case "", "identity":
		raw, err := io.ReadAll(io.LimitReader(resp.Body, maxErrorBodySnippet))
		if err != nil {
			return "", fmt.Errorf("neonretry: read error body: %w", err)
		}
		return fmt.Sprintf("%s response: %s", origContentType, sanitizeSnippet(raw)), nil

	case "gzip":
		rawCompressed, err := io.ReadAll(io.LimitReader(resp.Body, maxRawErrorBody))
		if err != nil {
			return "", fmt.Errorf("neonretry: read compressed error body: %w", err)
		}

		decompressed, decodeErr := decodeGzipSnippet(rawCompressed)
		if decodeErr != nil {
			// Malformed/truncated gzip: fall back to the raw bytes we
			// already have, sanitized like any other binary content.
			return fmt.Sprintf("%s response (encoding: gzip, undecodable): %s", origContentType, sanitizeSnippet(rawCompressed)), nil
		}
		return fmt.Sprintf("%s response: %s", origContentType, sanitizeSnippet(decompressed)), nil

	default:
		// Unknown/unsupported encoding: don't risk embedding binary
		// garbage in the message, just note the encoding.
		return fmt.Sprintf("%s response (encoding: %s, body omitted)", origContentType, encoding), nil
	}
}

// decodeGzipSnippet decompresses raw (assumed gzip-encoded) bytes, capped
// at maxErrorBodySnippet decompressed bytes. Because gzip.Reader decodes
// incrementally, capping the read via io.LimitReader bounds decompression
// work even if raw would expand to something much larger.
func decodeGzipSnippet(raw []byte) ([]byte, error) {
	gz, err := gzip.NewReader(bytes.NewReader(raw))
	if err != nil {
		return nil, fmt.Errorf("neonretry: create gzip reader: %w", err)
	}
	defer func() { _ = gz.Close() }()

	decompressed, err := io.ReadAll(io.LimitReader(gz, maxErrorBodySnippet))
	if err != nil {
		return nil, fmt.Errorf("neonretry: decompress gzip body: %w", err)
	}
	return decompressed, nil
}

// sanitizeSnippet converts raw (possibly binary/non-UTF8) response body
// bytes into a readable, quoted string suitable for embedding in a JSON
// error message. strconv.Quote escapes control characters and non-printable
// runes so the result stays on a single line and free of garbage bytes.
func sanitizeSnippet(body []byte) string {
	valid := strings.ToValidUTF8(string(body), "�")
	return strconv.Quote(strings.TrimSpace(valid))
}
