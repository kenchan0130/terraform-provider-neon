package neonwait_test

import (
	"context"
	"errors"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/kenchan0130/terraform-provider-neon/internal/neon"
	"github.com/kenchan0130/terraform-provider-neon/internal/neonwait"
)

const testInterval = time.Millisecond

// sequenceGetter returns a canned sequence of *neon.OperationResponse
// (keyed by operation ID) for each call to GetProjectOperation, advancing
// through the sequence on each call and repeating the last entry once
// exhausted. It also counts total calls.
type sequenceGetter struct {
	responses []neon.Operation
	calls     atomic.Int32
	err       error
}

func (g *sequenceGetter) GetProjectOperation(_ context.Context, params neon.GetProjectOperationParams) (*neon.OperationResponse, error) {
	n := g.calls.Add(1)
	if g.err != nil {
		return nil, g.err
	}
	idx := int(n) - 1
	if idx >= len(g.responses) {
		idx = len(g.responses) - 1
	}
	op := g.responses[idx]
	op.ID = params.OperationID
	return &neon.OperationResponse{Operation: op}, nil
}

func newOp(status neon.OperationStatus) neon.Operation {
	return neon.Operation{
		ID:     uuid.New(),
		Action: neon.OperationAction("create_branch"),
		Status: status,
	}
}

func TestWaitForOperations_EmptyOpsReturnsNilFast(t *testing.T) {
	t.Parallel()

	getter := &sequenceGetter{}

	start := time.Now()
	err := neonwait.WaitForOperations(context.Background(), getter, "proj", nil, testInterval)
	elapsed := time.Since(start)

	if err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}
	if getter.calls.Load() != 0 {
		t.Errorf("expected zero API calls, got %d", getter.calls.Load())
	}
	if elapsed > 100*time.Millisecond {
		t.Errorf("expected fast return, took %v", elapsed)
	}
}

func TestWaitForOperations_AlreadyFinishedSkipsPolling(t *testing.T) {
	t.Parallel()

	getter := &sequenceGetter{}
	ops := []neon.Operation{newOp(neon.OperationStatusFinished)}

	err := neonwait.WaitForOperations(context.Background(), getter, "proj", ops, testInterval)
	if err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}
	if getter.calls.Load() != 0 {
		t.Errorf("expected zero API calls for an already-terminal operation, got %d", getter.calls.Load())
	}
}

func TestWaitForOperations_RunningThenFinished(t *testing.T) {
	t.Parallel()

	getter := &sequenceGetter{
		responses: []neon.Operation{
			newOp(neon.OperationStatusRunning),
			newOp(neon.OperationStatusRunning),
			newOp(neon.OperationStatusFinished),
		},
	}
	ops := []neon.Operation{newOp(neon.OperationStatusRunning)}

	err := neonwait.WaitForOperations(context.Background(), getter, "proj", ops, testInterval)
	if err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}
	if getter.calls.Load() != 3 {
		t.Errorf("expected 3 polls, got %d", getter.calls.Load())
	}
}

func TestWaitForOperations_FailedOperationReturnsError(t *testing.T) {
	t.Parallel()

	running := newOp(neon.OperationStatusRunning)

	failed := newOp(neon.OperationStatusFailed)
	failed.ID = running.ID
	failed.Error = neon.NewOptString("compute provisioning failed")
	getter := &sequenceGetter{responses: []neon.Operation{failed}}
	ops := []neon.Operation{running}

	err := neonwait.WaitForOperations(context.Background(), getter, "proj", ops, testInterval)
	if err == nil {
		t.Fatal("expected an error, got nil")
	}
	msg := err.Error()
	for _, want := range []string{running.ID.String(), string(failed.Action), "failed", "compute provisioning failed"} {
		if !strings.Contains(msg, want) {
			t.Errorf("expected error message %q to contain %q", msg, want)
		}
	}
}

func TestWaitForOperations_AlreadyFailedSkipsPolling(t *testing.T) {
	t.Parallel()

	getter := &sequenceGetter{}
	ops := []neon.Operation{newOp(neon.OperationStatusFailed)}

	err := neonwait.WaitForOperations(context.Background(), getter, "proj", ops, testInterval)
	if err == nil {
		t.Fatal("expected an error, got nil")
	}
	if getter.calls.Load() != 0 {
		t.Errorf("expected zero API calls for an already-terminal operation, got %d", getter.calls.Load())
	}
}

func TestWaitForOperations_ContextCancellationAborts(t *testing.T) {
	t.Parallel()

	getter := &sequenceGetter{
		responses: []neon.Operation{newOp(neon.OperationStatusRunning)},
	}
	ops := []neon.Operation{newOp(neon.OperationStatusRunning)}

	ctx, cancel := context.WithCancel(context.Background())
	go func() {
		time.Sleep(5 * time.Millisecond)
		cancel()
	}()

	err := neonwait.WaitForOperations(ctx, getter, "proj", ops, 200*time.Millisecond)
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("expected context.Canceled, got %v", err)
	}
}

// TestWaitForOperations_AllStatusValues exercises every neon.OperationStatus
// value to make sure each is classified correctly: scheduling/running/
// cancelling keep polling (and the wait succeeds once a later poll reports
// finished), finished/skipped succeed immediately, and failed/error/
// cancelled fail immediately.
func TestWaitForOperations_AllStatusValues(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		status    neon.OperationStatus
		wantErr   bool
		wantPolls int32 // polls before the getter reports "finished"
	}{
		"scheduling keeps polling then finishes": {status: neon.OperationStatusScheduling, wantErr: false, wantPolls: 1},
		"running keeps polling then finishes":    {status: neon.OperationStatusRunning, wantErr: false, wantPolls: 1},
		"cancelling keeps polling then finishes": {status: neon.OperationStatusCancelling, wantErr: false, wantPolls: 1},
		"finished succeeds immediately":          {status: neon.OperationStatusFinished, wantErr: false, wantPolls: 0},
		"skipped succeeds immediately":           {status: neon.OperationStatusSkipped, wantErr: false, wantPolls: 0},
		"failed fails immediately":               {status: neon.OperationStatusFailed, wantErr: true, wantPolls: 0},
		"error fails immediately":                {status: neon.OperationStatusError, wantErr: true, wantPolls: 0},
		"cancelled fails immediately":            {status: neon.OperationStatusCancelled, wantErr: true, wantPolls: 0},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			getter := &sequenceGetter{
				responses: []neon.Operation{newOp(neon.OperationStatusFinished)},
			}
			ops := []neon.Operation{newOp(tt.status)}

			err := neonwait.WaitForOperations(context.Background(), getter, "proj", ops, testInterval)

			if tt.wantErr && err == nil {
				t.Fatal("expected an error, got nil")
			}
			if !tt.wantErr && err != nil {
				t.Fatalf("expected nil error, got %v", err)
			}
			if getter.calls.Load() != tt.wantPolls {
				t.Errorf("expected %d polls, got %d", tt.wantPolls, getter.calls.Load())
			}
		})
	}
}

func TestWaitForOperations_APIErrorIsWrapped(t *testing.T) {
	t.Parallel()

	getter := &sequenceGetter{err: errors.New("boom")}
	ops := []neon.Operation{newOp(neon.OperationStatusRunning)}

	err := neonwait.WaitForOperations(context.Background(), getter, "proj", ops, testInterval)
	if err == nil {
		t.Fatal("expected an error, got nil")
	}
	if !strings.Contains(err.Error(), "boom") {
		t.Errorf("expected wrapped error to contain %q, got %q", "boom", err.Error())
	}
}
