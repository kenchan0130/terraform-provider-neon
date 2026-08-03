// Package neonwait provides helpers to poll Neon project operations until
// they reach a terminal state. Many Neon API mutations (branch and endpoint
// create/update/delete, among others) return an "operations" array
// representing asynchronous work the platform is still performing. Until
// those operations finish, the project (or resources within it) may be
// locked and subsequent API calls can fail with 423 Locked. Waiting for
// operations to complete before returning from a CRUD method removes the
// need for practitioners to work around such locking with explicit
// dependencies or sleeps, and surfaces operation failures as Terraform
// errors instead of silent partial success.
package neonwait

import (
	"context"
	"fmt"
	"time"

	"github.com/kenchan0130/terraform-provider-neon/internal/neon"
)

// DefaultInterval is the default polling interval used between operation
// status checks.
const DefaultInterval = 1 * time.Second

// OperationGetter is the subset of *neon.Client used to poll operation
// status. *neon.Client satisfies this interface.
type OperationGetter interface {
	GetProjectOperation(ctx context.Context, params neon.GetProjectOperationParams) (*neon.OperationResponse, error)
}

// WaitForOperations polls the Neon API until every operation in ops reaches
// a terminal state, or returns an error if any operation fails, is
// cancelled, or ctx is cancelled first.
//
// Operations that are already terminal (as reported in ops) are skipped
// without an API call. For all others, status is checked immediately, then
// re-checked every interval until terminal.
func WaitForOperations(ctx context.Context, client OperationGetter, projectID string, ops []neon.Operation, interval time.Duration) error {
	for _, op := range ops {
		if err := waitForOperation(ctx, client, projectID, op, interval); err != nil {
			return err
		}
	}
	return nil
}

func waitForOperation(ctx context.Context, client OperationGetter, projectID string, op neon.Operation, interval time.Duration) error {
	if terminal, err := checkTerminal(op); terminal {
		return err
	}

	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		resp, err := client.GetProjectOperation(ctx, neon.GetProjectOperationParams{
			ProjectID:   projectID,
			OperationID: op.ID,
		})
		if err != nil {
			return fmt.Errorf("get operation %s status: %w", op.ID, err)
		}

		if terminal, err := checkTerminal(resp.Operation); terminal {
			return err
		}

		select {
		case <-ctx.Done():
			return fmt.Errorf("wait for operation %s: %w", op.ID, ctx.Err())
		case <-ticker.C:
		}
	}
}

// checkTerminal reports whether op has reached a terminal state. When it
// has, terminal is true and err is nil for success or non-nil describing
// the failure. When op is still in progress, terminal is false.
func checkTerminal(op neon.Operation) (terminal bool, err error) {
	switch op.Status {
	case neon.OperationStatusFinished, neon.OperationStatusSkipped:
		return true, nil
	case neon.OperationStatusFailed, neon.OperationStatusError, neon.OperationStatusCancelled:
		msg := fmt.Sprintf("operation %s (action: %s) ended with status %q", op.ID, op.Action, op.Status)
		if reason, ok := op.Error.Get(); ok && reason != "" {
			msg = fmt.Sprintf("%s: %s", msg, reason)
		}
		return true, fmt.Errorf("%s", msg)
	case neon.OperationStatusScheduling, neon.OperationStatusRunning, neon.OperationStatusCancelling:
		return false, nil
	default:
		// Unknown status values are treated as still in progress so a
		// future API-added status doesn't cause a false-positive success.
		return false, nil
	}
}
