package checks

import (
	"context"
	"errors"
	"sync/atomic"
	"testing"
	"testing/synctest"
	"time"
)

func await[T any](t *testing.T, ch <-chan T) T {
	t.Helper()
	timer := time.NewTimer(5 * time.Second)
	defer timer.Stop()
	select {
	case value := <-ch:
		return value
	case <-timer.C:
		t.Fatal("timed out waiting for errgroup operation")
		var zero T
		return zero
	}
}

func assertNoResult(t *testing.T, ch <-chan error) {
	t.Helper()
	synctest.Wait()
	select {
	case err := <-ch:
		t.Fatalf("Run returned before a started job finished: %v", err)
	default:
	}
}

func runInSynctest(t *testing.T, name string, test func(*testing.T)) {
	t.Helper()
	t.Run(name, func(t *testing.T) {
		synctest.Test(t, test)
	})
}

func Run(t *testing.T, run func(context.Context, []func(context.Context) error, int) error) {
	t.Helper()
	runInSynctest(t, "limits active jobs and completes all callbacks", func(t *testing.T) {
		const limit = 2
		started := make(chan struct{}, 4)
		release := make(chan struct{})
		done := make(chan error, 1)
		jobs := make([]func(context.Context) error, 4)
		for index := range jobs {
			jobs[index] = func(context.Context) error {
				started <- struct{}{}
				<-release
				return nil
			}
		}
		go func() { done <- run(context.Background(), jobs, limit) }()
		await(t, started)
		await(t, started)
		synctest.Wait()
		select {
		case <-started:
			close(release)
			t.Fatal("Run exceeded the configured active-job limit")
		default:
		}
		close(release)
		if err := await(t, done); err != nil {
			t.Fatalf("Run returned error: %v", err)
		}
		for range len(jobs) - limit {
			await(t, started)
		}
	})

	runInSynctest(t, "cancels on error, joins active jobs, and skips pending callbacks", func(t *testing.T) {
		wantErr := errors.New("job failed")
		parentCtx, parentCancel := context.WithCancel(context.Background())
		defer parentCancel()
		started := make(chan int, 2)
		failNow := make(chan struct{})
		siblingCanceled := make(chan struct{})
		releaseSibling := make(chan struct{})
		done := make(chan error, 1)
		var pendingCalls atomic.Int32
		jobs := []func(context.Context) error{
			func(context.Context) error {
				started <- 0
				<-failNow
				return wantErr
			},
			func(ctx context.Context) error {
				started <- 1
				<-ctx.Done()
				close(siblingCanceled)
				<-releaseSibling
				return ctx.Err()
			},
			func(context.Context) error {
				pendingCalls.Add(1)
				return nil
			},
			func(context.Context) error {
				pendingCalls.Add(1)
				return nil
			},
		}
		go func() { done <- run(parentCtx, jobs, 2) }()
		first, second := await(t, started), await(t, started)
		if first == second {
			t.Fatalf("only job %d started", first)
		}
		close(failNow)
		await(t, siblingCanceled)
		assertNoResult(t, done)
		close(releaseSibling)
		err := await(t, done)
		if !errors.Is(err, wantErr) {
			t.Fatalf("Run error = %v, want original job error", err)
		}
		if parentCtx.Err() != nil {
			t.Fatalf("Run canceled its caller-owned context: %v", parentCtx.Err())
		}
		if pendingCalls.Load() != 0 {
			t.Fatalf("Run called %d jobs after cancellation", pendingCalls.Load())
		}
	})

	runInSynctest(t, "parent cancellation is returned even with no jobs", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		cancel()
		if err := run(ctx, nil, 1); !errors.Is(err, context.Canceled) {
			t.Fatalf("Run(canceled, empty) = %v, want context.Canceled", err)
		}
		if err := run(context.Background(), nil, 1); err != nil {
			t.Fatalf("Run(empty) = %v, want nil", err)
		}
	})

	runInSynctest(t, "rejects nonpositive limits and joins after parent cancellation", func(t *testing.T) {
		if err := run(context.Background(), nil, 0); err == nil {
			t.Fatal("Run accepted limit <= 0")
		}
		ctx, cancel := context.WithCancel(context.Background())
		started := make(chan struct{})
		observedCancel := make(chan struct{})
		release := make(chan struct{})
		done := make(chan error, 1)
		job := func(ctx context.Context) error {
			close(started)
			<-ctx.Done()
			close(observedCancel)
			<-release
			return ctx.Err()
		}
		go func() { done <- run(ctx, []func(context.Context) error{job}, 1) }()
		await(t, started)
		cancel()
		await(t, observedCancel)
		assertNoResult(t, done)
		close(release)
		if err := await(t, done); !errors.Is(err, context.Canceled) {
			t.Fatalf("Run after parent cancellation = %v, want context.Canceled", err)
		}
	})
}
