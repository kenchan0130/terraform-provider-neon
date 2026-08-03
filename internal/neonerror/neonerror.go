package neonerror

import (
	"errors"
	"fmt"
	"net/http"

	"github.com/kenchan0130/terraform-provider-neon/internal/neon"
)

// IsNotFound returns true if the error represents a 404 Not Found response from the Neon API.
func IsNotFound(err error) bool {
	var apiErr *neon.GeneralErrorStatusCode
	if errors.As(err, &apiErr) {
		return apiErr.StatusCode == http.StatusNotFound
	}
	return false
}

// Detail returns a human-readable description of a Neon API error.
func Detail(err error) string {
	var apiErr *neon.GeneralErrorStatusCode
	if errors.As(err, &apiErr) {
		msg := fmt.Sprintf("HTTP %d (%s): %s", apiErr.StatusCode, apiErr.Response.Code, apiErr.Response.Message)
		if rid, ok := apiErr.Response.RequestID.Get(); ok {
			msg = fmt.Sprintf("%s (request ID: %s)", msg, rid)
		}
		return msg
	}
	return err.Error()
}
