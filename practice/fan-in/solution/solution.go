package solution

import (
	"context"
	"sync"
)

func Merge(ctx context.Context, inputs ...<-chan int) <-chan int {
	out := make(chan int)
	if ctx.Err() != nil {
		close(out)
		return out
	}
	var group sync.WaitGroup
	count := 0
	for _, input := range inputs {
		if input == nil {
			continue
		}
		count++
		source := input
		group.Go(func() {
			for {
				select {
				case <-ctx.Done():
					return
				case value, ok := <-source:
					if !ok {
						return
					}
					select {
					case out <- value:
					case <-ctx.Done():
						return
					}
				}
			}
		})
	}
	if count == 0 {
		close(out)
		return out
	}
	go func() {
		group.Wait()
		close(out)
	}()
	return out
}
