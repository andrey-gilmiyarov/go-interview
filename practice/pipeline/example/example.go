package example

import (
	"context"

	"github.com/andreygilmiyarov/go-interview/practice/pipeline/solution"
)

func PipelineExample(ctx context.Context) []int {
	input := make(chan int, 5)
	for value := 1; value <= 5; value++ {
		input <- value
	}
	close(input)

	doubled, doubleDone := doubleOutput(ctx, input)
	var squares []int
	for value := range solution.Square(ctx, doubled) {
		squares = append(squares, value)
	}
	<-doubleDone
	return squares
}

func doubleOutput(ctx context.Context, in <-chan int) (<-chan int, <-chan struct{}) {
	out := make(chan int)
	done := make(chan struct{})
	if ctx.Err() != nil {
		close(out)
		close(done)
		return out, done
	}

	go func() {
		defer close(done)
		defer close(out)
		for {
			select {
			case <-ctx.Done():
				return
			case value, ok := <-in:
				if !ok {
					return
				}
				select {
				case out <- value * 2:
				case <-ctx.Done():
					return
				}
			}
		}
	}()
	return out, done
}
