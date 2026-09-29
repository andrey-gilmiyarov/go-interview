package example

import (
	"context"
	"reflect"
	"testing"
	"testing/synctest"
)

func TestPipelineExamplePreservesTwoStageScenario(t *testing.T) {
	got := PipelineExample(context.Background())
	want := []int{4, 16, 36, 64, 100}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("PipelineExample() = %v, want %v", got, want)
	}
}

func TestDoubleOutputCancellationClosesAndJoins(t *testing.T) {
	for _, testCase := range []struct {
		name  string
		input []int
	}{
		{name: "waiting for input"},
		{name: "blocked forwarding", input: []int{3}},
	} {
		t.Run(testCase.name, func(t *testing.T) {
			synctest.Test(t, func(t *testing.T) {
				ctx, cancel := context.WithCancel(context.Background())
				input := make(chan int, len(testCase.input))
				for _, value := range testCase.input {
					input <- value
				}
				out, done := doubleOutput(ctx, input)
				synctest.Wait()
				cancel()
				synctest.Wait()

				select {
				case value, ok := <-out:
					if ok {
						t.Fatalf("doubleOutput emitted %d after cancellation", value)
					}
				default:
					t.Fatal("doubleOutput did not close output after cancellation")
				}
				select {
				case <-done:
				default:
					t.Fatal("doubleOutput returned before its goroutine exited")
				}
			})
		})
	}
}
