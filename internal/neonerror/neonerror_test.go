package neonerror_test

import (
	"errors"
	"testing"

	goFasterErrors "github.com/go-faster/errors"
	"github.com/kenchan0130/terraform-provider-neon/internal/neon"
	"github.com/kenchan0130/terraform-provider-neon/internal/neonerror"
)

func TestDetail(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		err  error
		want string
	}{
		"API error with request ID": {
			err: &neon.GeneralErrorStatusCode{
				StatusCode: 400,
				Response: neon.GeneralError{
					Code:      neon.ErrorCode("invalid_request"),
					Message:   "updating the maintenance window is not allowed",
					RequestID: neon.NewOptString("req-123"),
				},
			},
			want: "HTTP 400 (invalid_request): updating the maintenance window is not allowed (request ID: req-123)",
		},
		"API error without request ID": {
			err: &neon.GeneralErrorStatusCode{
				StatusCode: 404,
				Response: neon.GeneralError{
					Code:    neon.ErrorCode("not_found"),
					Message: "project not found",
				},
			},
			want: "HTTP 404 (not_found): project not found",
		},
		"API error wrapped with go-faster/errors": {
			err: goFasterErrors.Wrap(&neon.GeneralErrorStatusCode{
				StatusCode: 400,
				Response: neon.GeneralError{
					Code:      neon.ErrorCode(""),
					Message:   "updating the maintenance window is not allowed",
					RequestID: neon.NewOptString("req-123"),
				},
			}, "do request"),
			want: "HTTP 400 (): updating the maintenance window is not allowed (request ID: req-123)",
		},
		"plain error fallback": {
			err:  errors.New("boom"),
			want: "boom",
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			got := neonerror.Detail(tt.err)
			if got != tt.want {
				t.Errorf("Detail() = %q, want %q", got, tt.want)
			}
		})
	}
}
