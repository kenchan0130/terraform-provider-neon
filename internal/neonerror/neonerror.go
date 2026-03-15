package neonerror

import (
	"errors"
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
