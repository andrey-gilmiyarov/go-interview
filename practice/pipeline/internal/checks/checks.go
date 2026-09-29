package checks

import (
	"context"
	"reflect"
	"testing"
	"time"
)

func collectValues(t *testing.T, cancel context.CancelFunc, out <-chan int, count int) []int {
	t.Helper()
	timer := time.NewTimer(5 * time.Second)
	defer timer.Stop()
	got := make([]int, 0, count)
	for len(got) < count {
		select {
		case value, ok := <-out:
			if !ok {
				cancel()
				t.Fatalf("output closed after %d of %d values", len(got), count)
			}
			got = append(got, value)
		case <-timer.C:
			cancel()
			t.Fatalf("timed out waiting for %d output values; received %v", count, got)
		}
	}
	return got
}

func awaitClosed(t *testing.T, cancel context.CancelFunc, out <-chan int) {
	t.Helper()
	timer := time.NewTimer(5 * time.Second)
	defer timer.Stop()
	select {
	case value, ok := <-out:
		if ok {
			cancel()
			t.Fatalf("received unexpected %d before channel closed", value)
		}
	case <-timer.C:
		cancel()
		t.Fatal("output channel did not close")
	}
}

func drainToClosedAfterCancel(t *testing.T, cancel context.CancelFunc, out <-chan int) {
	t.Helper()
	timer := time.NewTimer(5 * time.Second)
	defer timer.Stop()
	for {
		select {
		case _, ok := <-out:
			if !ok {
				return
			}
		case <-timer.C:
			cancel()
			t.Fatal("output channel did not close after cancellation")
		}
	}
}

func awaitSignal(t *testing.T, cancel context.CancelFunc, ch <-chan struct{}) {
	t.Helper()
	timer := time.NewTimer(5 * time.Second)
	defer timer.Stop()
	select {
	case <-ch:
	case <-timer.C:
		cancel()
		t.Fatal("timed out waiting for signal")
	}
}

func Run(t *testing.T, square func(context.Context, <-chan int) <-chan int) {
	t.Helper()
	t.Run("squares values and closes after input", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()
		input := make(chan int, 3)
		input <- 1
		input <- 2
		input <- 3
		close(input)
		out := square(ctx, input)
		got := collectValues(t, cancel, out, 3)
		awaitClosed(t, cancel, out)
		if want := []int{1, 4, 9}; !reflect.DeepEqual(got, want) {
			t.Fatalf("Square output = %v, want %v", got, want)
		}
	})

	t.Run("pre-canceled context closes output without consuming input", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		cancel()
		input := make(chan int, 1)
		input <- 11
		out := square(ctx, input)
		select {
		case value, ok := <-out:
			if ok {
				t.Fatalf("pre-canceled Square emitted %d", value)
			}
		default:
			t.Fatal("Square did not return an already-closed output for a pre-canceled context")
		}
		select {
		case value, ok := <-input:
			if !ok || value != 11 {
				t.Fatalf("Square changed caller-owned input: received %d, open=%v", value, ok)
			}
		default:
			t.Fatal("Square consumed caller-owned input before noticing pre-cancellation")
		}
	})

	t.Run("cancellation unblocks a send", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		input := make(chan int)
		out := square(ctx, input)
		sent := make(chan struct{})
		go func() {
			select {
			case input <- 7:
				close(sent)
			case <-ctx.Done():
			}
		}()
		awaitSignal(t, cancel, sent)
		cancel()
		drainToClosedAfterCancel(t, cancel, out)
	})

	t.Run("cancellation unblocks a receive and leaves input owned by caller", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		input := make(chan int, 1)
		out := square(ctx, input)
		cancel()
		awaitClosed(t, cancel, out)
		select {
		case input <- 8:
		default:
			t.Fatal("Square closed or consumed the caller-owned input channel")
		}
	})
}
