package saga

import (
	"context"
	"errors"
	"testing"
	"time"
)

func TestInMemoryCoordinator_SuccessFlow(t *testing.T) {
	coord := NewInMemoryCoordinator()
	ctx := context.Background()

	step1 := SagaStep{
		Name: "s1",
		Handler: func(ctx context.Context, data any) (any, error) {
			return "d1", nil
		},
		Compensation: func(ctx context.Context, data any) error { return nil },
	}
	step2 := SagaStep{
		Name: "s2",
		Handler: func(ctx context.Context, data any) (any, error) {
			if data != "d1" {
				t.Fatalf("unexpected input to step2: %v", data)
			}
			return "d2", nil
		},
		Compensation: func(ctx context.Context, data any) error { return nil },
	}

	def := &SagaDefinition{Name: "ok", Steps: []SagaStep{step1, step2}}

	exec, err := coord.StartSaga(ctx, def, "i0")
	if err != nil {
		t.Fatalf("StartSaga error: %v", err)
	}
	if exec.Status != StatusCompleted {
		t.Fatalf("unexpected status: %v", exec.Status)
	}
	if len(exec.Executed) != 2 || exec.Executed[0] != "s1" || exec.Executed[1] != "s2" {
		t.Fatalf("unexpected executed steps: %+v", exec.Executed)
	}

	// get status
	got, err := coord.GetSagaStatus(exec.ID)
	if err != nil {
		t.Fatalf("GetSagaStatus error: %v", err)
	}
	if got.Status != StatusCompleted {
		t.Fatalf("unexpected final status: %v", got.Status)
	}
}

func TestInMemoryCoordinator_FailureAndCompensation(t *testing.T) {
	coord := NewInMemoryCoordinator()
	ctx := context.Background()

	var compensated []string

	step1 := SagaStep{
		Name:         "s1",
		Handler:      func(ctx context.Context, data any) (any, error) { return "d1", nil },
		Compensation: func(ctx context.Context, data any) error { compensated = append(compensated, "s1"); return nil },
	}
	step2 := SagaStep{
		Name:         "s2",
		Handler:      func(ctx context.Context, data any) (any, error) { return nil, errors.New("boom") },
		Compensation: func(ctx context.Context, data any) error { compensated = append(compensated, "s2"); return nil },
	}

	def := &SagaDefinition{Name: "fail", Steps: []SagaStep{step1, step2}}

	exec, err := coord.StartSaga(ctx, def, "i0")
	if err == nil {
		t.Fatalf("expected error, got nil")
	}
	if exec.Status != StatusCompensated {
		t.Fatalf("unexpected status: %v", exec.Status)
	}
	// only s1 executed, so only s1 should be compensated
	if len(compensated) != 1 || compensated[0] != "s1" {
		t.Fatalf("unexpected compensated steps: %+v", compensated)
	}
}

func TestInMemoryCoordinator_ContextCancel(t *testing.T) {
	coord := NewInMemoryCoordinator()
	ctx, cancel := context.WithCancel(context.Background())

	step1 := SagaStep{
		Name: "s1",
		Handler: func(ctx context.Context, data any) (any, error) {
			cancel()
			return "d1", nil
		},
		Compensation: func(ctx context.Context, data any) error { return nil },
	}
	step2 := SagaStep{
		Name: "s2",
		Handler: func(ctx context.Context, data any) (any, error) {
			// should not run due to cancel before loop iteration
			select {
			case <-time.After(10 * time.Millisecond):
				return "d2", nil
			case <-ctx.Done():
				return nil, ctx.Err()
			}
		},
		Compensation: func(ctx context.Context, data any) error { return nil },
	}

	def := &SagaDefinition{Name: "cancel", Steps: []SagaStep{step1, step2}}

	_, err := coord.StartSaga(ctx, def, "i0")
	if err == nil {
		t.Fatalf("expected cancel error, got nil")
	}
}
