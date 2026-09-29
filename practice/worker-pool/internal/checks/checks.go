package checks

import (
	"context"
	"errors"
	"reflect"
	"sync/atomic"
	"testing"
	"testing/synctest"
	"time"
)

type mapFunc func(context.Context, []int, int, func(context.Context, int) (int, error)) ([]int, error)

type mapResult struct {
	values []int
	err    error
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

func assertNoValue[T any](t *testing.T, ch <-chan T) {
	t.Helper()
	synctest.Wait()
	select {
	case value := <-ch:
		t.Fatalf("received unexpected value while workers were blocked: %#v", value)
	default:
	}
}

func runInSynctest(t *testing.T, name string, test func(*testing.T)) {
	t.Helper()
	t.Run(name, func(t *testing.T) {
		synctest.Test(t, test)
	})
}

func Run(t *testing.T, solve mapFunc) {
	t.Helper()
	runInSynctest(t, "bounds callbacks and preserves input order", func(t *testing.T) {
		values := []int{0, 1, 2, 3, 4, 5}
		const workers = 2
		started := make(chan int, len(values))
		release := make(chan struct{})
		done := make(chan mapResult, 1)
		go func() {
			result, err := solve(context.Background(), values, workers, func(_ context.Context, value int) (int, error) {
				started <- value
				<-release
				return value * 10, nil
			})
			done <- mapResult{values: result, err: err}
		}()

		for range workers {
			await(t, started)
		}
		assertNoValue(t, started)
		close(release)
		result := await(t, done)
		if result.err != nil {
			t.Fatalf("Map returned error: %v", result.err)
		}
		want := []int{0, 10, 20, 30, 40, 50}
		if !reflect.DeepEqual(result.values, want) {
			t.Fatalf("Map result = %v, want %v", result.values, want)
		}
	})

	runInSynctest(t, "validates workers and handles empty or canceled input", func(t *testing.T) {
		if _, err := solve(context.Background(), []int{1}, 0, func(_ context.Context, value int) (int, error) {
			return value, nil
		}); err == nil {
			t.Fatal("Map accepted workers <= 0")
		}
		empty, err := solve(context.Background(), nil, 2, func(_ context.Context, value int) (int, error) {
			return value, nil
		})
		if err != nil || len(empty) != 0 {
			t.Fatalf("Map(empty) = %v, %v; want empty result and nil error", empty, err)
		}

		ctx, cancel := context.WithCancel(context.Background())
		cancel()
		var calls atomic.Int32
		result, err := solve(ctx, []int{1}, 1, func(_ context.Context, value int) (int, error) {
			calls.Add(1)
			return value, nil
		})
		if !errors.Is(err, context.Canceled) || result != nil {
			t.Fatalf("Map(pre-canceled) = %v, %v; want nil, context.Canceled", result, err)
		}
		if calls.Load() != 0 {
			t.Fatalf("Map started %d callbacks after pre-cancellation", calls.Load())
		}
	})

	runInSynctest(t, "error cancels and joins started callbacks", func(t *testing.T) {
		wantErr := errors.New("callback failed")
		parentCtx, parentCancel := context.WithCancel(context.Background())
		defer parentCancel()
		started := make(chan int, 2)
		failNow := make(chan struct{})
		releaseSibling := make(chan struct{})
		siblingCanceled := make(chan struct{})
		done := make(chan mapResult, 1)
		go func() {
			result, err := solve(parentCtx, []int{1, 2, 3}, 2, func(ctx context.Context, value int) (int, error) {
				started <- value
				if value == 1 {
					<-failNow
					return 0, wantErr
				}
				if value == 2 {
					<-ctx.Done()
					close(siblingCanceled)
					<-releaseSibling
					return 0, ctx.Err()
				}
				return value, nil
			})
			done <- mapResult{values: result, err: err}
		}()
		first, second := await(t, started), await(t, started)
		if (first != 1 && first != 2) || (second != 1 && second != 2) || first == second {
			t.Fatalf("started callbacks = %d, %d; want callbacks 1 and 2", first, second)
		}
		close(failNow)
		await(t, siblingCanceled)
		assertNoValue(t, done)
		close(releaseSibling)
		result := await(t, done)
		if result.values != nil || !errors.Is(result.err, wantErr) {
			t.Fatalf("Map after callback error = %v, %v; want nil and original error", result.values, result.err)
		}
		if parentCtx.Err() != nil {
			t.Fatalf("Map canceled its caller-owned context: %v", parentCtx.Err())
		}
	})

	runInSynctest(t, "parent cancellation waits for active callback", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()
		started := make(chan struct{})
		allowReturn := make(chan struct{})
		observedCancel := make(chan struct{})
		done := make(chan mapResult, 1)
		go func() {
			result, err := solve(ctx, []int{1}, 1, func(ctx context.Context, value int) (int, error) {
				close(started)
				<-ctx.Done()
				close(observedCancel)
				<-allowReturn
				return 0, ctx.Err()
			})
			done <- mapResult{values: result, err: err}
		}()
		await(t, started)
		cancel()
		await(t, observedCancel)
		assertNoValue(t, done)
		close(allowReturn)
		result := await(t, done)
		if result.values != nil || !errors.Is(result.err, context.Canceled) {
			t.Fatalf("Map after parent cancellation = %v, %v; want nil, context.Canceled", result.values, result.err)
		}
	})
}
