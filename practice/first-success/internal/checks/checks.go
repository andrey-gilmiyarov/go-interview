package checks

import (
	"context"
	"errors"
	"testing"
	"testing/synctest"
	"time"
)

type firstResult struct {
	value string
	err   error
}

func await[T any](t *testing.T, ch <-chan T) T {
	t.Helper()
	timer := time.NewTimer(5 * time.Second)
	defer timer.Stop()
	select {
	case value := <-ch:
		return value
	case <-timer.C:
		t.Fatal("timed out waiting for concurrent operation")
		var zero T
		return zero
	}
}

func assertNoResult(t *testing.T, ch <-chan firstResult) {
	t.Helper()
	synctest.Wait()
	select {
	case result := <-ch:
		t.Fatalf("First returned before a started callback finished: %#v", result)
	default:
	}
}

func runInSynctest(t *testing.T, name string, test func(*testing.T)) {
	t.Helper()
	t.Run(name, func(t *testing.T) {
		synctest.Test(t, test)
	})
}

func Run(t *testing.T, noJobs error, first func(context.Context, ...func(context.Context) (string, error)) (string, error)) {
	t.Helper()
	runInSynctest(t, "reports no jobs", func(t *testing.T) {
		if noJobs == nil {
			t.Fatal("ErrNoJobs is nil")
		}
		if value, err := first(context.Background()); value != "" || !errors.Is(err, noJobs) {
			t.Fatalf("First() = %q, %v; want ErrNoJobs", value, err)
		}
	})

	runInSynctest(t, "returns first success and joins canceled sibling", func(t *testing.T) {
		parentCtx, parentCancel := context.WithCancel(context.Background())
		defer parentCancel()
		started := make(chan int, 2)
		allowSuccess := make(chan struct{})
		siblingCanceled := make(chan struct{})
		releaseSibling := make(chan struct{})
		done := make(chan firstResult, 1)
		jobs := []func(context.Context) (string, error){
			func(context.Context) (string, error) {
				started <- 0
				<-allowSuccess
				return "winner", nil
			},
			func(ctx context.Context) (string, error) {
				started <- 1
				<-ctx.Done()
				close(siblingCanceled)
				<-releaseSibling
				return "", ctx.Err()
			},
		}
		go func() {
			value, err := first(parentCtx, jobs...)
			done <- firstResult{value: value, err: err}
		}()
		a, b := await(t, started), await(t, started)
		if a == b {
			t.Fatalf("only callback %d started", a)
		}
		close(allowSuccess)
		await(t, siblingCanceled)
		assertNoResult(t, done)
		close(releaseSibling)
		result := await(t, done)
		if result.value != "winner" || result.err != nil {
			t.Fatalf("First() = %q, %v; want winner, nil", result.value, result.err)
		}
		if parentCtx.Err() != nil {
			t.Fatalf("First canceled its caller-owned context: %v", parentCtx.Err())
		}
	})

	runInSynctest(t, "joins every error when all jobs fail", func(t *testing.T) {
		errA := errors.New("service A failed")
		errB := errors.New("service B failed")
		value, err := first(context.Background(),
			func(context.Context) (string, error) { return "", errA },
			func(context.Context) (string, error) { return "", errB },
		)
		if value != "" || !errors.Is(err, errA) || !errors.Is(err, errB) {
			t.Fatalf("First() = %q, %v; want joined errors A and B", value, err)
		}
	})

	runInSynctest(t, "parent cancellation is returned when no success was accepted", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		started := make(chan struct{})
		allowReturn := make(chan struct{})
		observedCancel := make(chan struct{})
		done := make(chan firstResult, 1)
		go func() {
			value, err := first(ctx, func(ctx context.Context) (string, error) {
				close(started)
				<-ctx.Done()
				close(observedCancel)
				<-allowReturn
				return "", ctx.Err()
			})
			done <- firstResult{value: value, err: err}
		}()
		await(t, started)
		cancel()
		await(t, observedCancel)
		assertNoResult(t, done)
		close(allowReturn)
		result := await(t, done)
		if result.value != "" || !errors.Is(result.err, context.Canceled) {
			t.Fatalf("First() after cancellation = %q, %v; want context.Canceled", result.value, result.err)
		}
	})
}
