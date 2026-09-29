package checks

import (
	"context"
	"reflect"
	"testing"
	"time"
)

const failureTimeout = 5 * time.Second

func collectValues(t *testing.T, cancel context.CancelFunc, out <-chan int, count int) []int {
	t.Helper()
	timer := time.NewTimer(failureTimeout)
	defer timer.Stop()

	got := make([]int, 0, count)
	for len(got) < count {
		select {
		case value, ok := <-out:
			if !ok {
				cancel()
				t.Fatalf("merged output closed after %d of %d values", len(got), count)
			}
			got = append(got, value)
		case <-timer.C:
			cancel()
			t.Fatalf("timed out waiting for %d merged values; received %v", count, got)
		}
	}
	return got
}

func awaitClosed(t *testing.T, cancel context.CancelFunc, out <-chan int) {
	t.Helper()
	timer := time.NewTimer(failureTimeout)
	defer timer.Stop()
	select {
	case value, ok := <-out:
		if ok {
			cancel()
			t.Fatalf("received unexpected %d before output closed", value)
		}
	case <-timer.C:
		cancel()
		t.Fatal("merged output did not close")
	}
}

func drainToClosedAfterCancel(t *testing.T, cancel context.CancelFunc, out <-chan int) {
	t.Helper()
	timer := time.NewTimer(failureTimeout)
	defer timer.Stop()
	for {
		select {
		case _, ok := <-out:
			if !ok {
				return
			}
		case <-timer.C:
			cancel()
			t.Fatal("merged output did not close after cancellation")
		}
	}
}

func awaitSignal(t *testing.T, cancel context.CancelFunc, ch <-chan struct{}) {
	t.Helper()
	timer := time.NewTimer(failureTimeout)
	defer timer.Stop()
	select {
	case <-ch:
	case <-timer.C:
		cancel()
		t.Fatal("timed out waiting for signal")
	}
}

func Run(t *testing.T, merge func(context.Context, ...<-chan int) <-chan int) {
	t.Helper()
	t.Run("preserves each input order", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()
		left := make(chan int, 3)
		right := make(chan int, 3)
		for _, value := range []int{1, 3, 5} {
			left <- value
		}
		for _, value := range []int{2, 4, 6} {
			right <- value
		}
		close(left)
		close(right)

		out := merge(ctx, left, right)
		got := collectValues(t, cancel, out, 6)
		awaitClosed(t, cancel, out)
		seen := make(map[int]bool, len(got))
		lastLeft, lastRight := 0, 0
		for _, value := range got {
			if seen[value] {
				t.Fatalf("Merge duplicated value %d: %v", value, got)
			}
			seen[value] = true
			if value%2 == 1 {
				if value <= lastLeft {
					t.Fatalf("left input order changed: %v", got)
				}
				lastLeft = value
			} else {
				if value <= lastRight {
					t.Fatalf("right input order changed: %v", got)
				}
				lastRight = value
			}
		}
		for value := 1; value <= 6; value++ {
			if !seen[value] {
				t.Fatalf("Merge omitted %d: %v", value, got)
			}
		}
	})

	t.Run("ignores nil inputs and closes with no inputs", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()
		empty := make(chan int)
		close(empty)
		out := merge(ctx, nil, empty, nil)
		got := collectValues(t, cancel, out, 0)
		awaitClosed(t, cancel, out)
		if !reflect.DeepEqual(got, []int{}) {
			t.Fatalf("Merge(nil, empty, nil) = %v, want empty", got)
		}

		noInputsCtx, cancelNoInputs := context.WithCancel(context.Background())
		defer cancelNoInputs()
		awaitClosed(t, cancelNoInputs, merge(noInputsCtx))
	})

	t.Run("pre-canceled context closes output without consuming input", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		cancel()
		input := make(chan int, 1)
		input <- 11
		out := merge(ctx, input)
		select {
		case value, ok := <-out:
			if ok {
				t.Fatalf("pre-canceled Merge emitted %d", value)
			}
		default:
			t.Fatal("Merge did not return an already-closed output for a pre-canceled context")
		}
		select {
		case value, ok := <-input:
			if !ok || value != 11 {
				t.Fatalf("Merge changed caller-owned input: received %d, open=%v", value, ok)
			}
		default:
			t.Fatal("Merge consumed caller-owned input before noticing pre-cancellation")
		}
	})

	t.Run("cancellation unblocks forwarding and receiving", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()
		input := make(chan int)
		out := merge(ctx, input)
		sent := make(chan struct{})
		go func() {
			select {
			case input <- 9:
				close(sent)
			case <-ctx.Done():
			}
		}()
		awaitSignal(t, cancel, sent)
		cancel()
		drainToClosedAfterCancel(t, cancel, out)

		ctx2, cancel2 := context.WithCancel(context.Background())
		defer cancel2()
		waiting := make(chan int)
		out2 := merge(ctx2, waiting)
		cancel2()
		awaitClosed(t, cancel2, out2)
		select {
		case waiting <- 10:
			t.Fatal("Merge consumed caller-owned input after cancellation")
		default:
		}
	})
}
