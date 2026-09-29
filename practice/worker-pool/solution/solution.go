package solution

import (
	"context"
	"fmt"
	"sync"
)

func Map(ctx context.Context, values []int, workers int, fn func(context.Context, int) (int, error)) ([]int, error) {
	if workers <= 0 {
		return nil, fmt.Errorf("worker-pool: workers must be positive")
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if len(values) == 0 {
		return []int{}, nil
	}

	workCtx, cancel := context.WithCancel(ctx)
	defer cancel()

	type job struct{ index int }
	jobs := make(chan job)
	results := make([]int, len(values))
	var firstErr error
	var errMu sync.Mutex
	var group sync.WaitGroup

	activeWorkers := min(workers, len(values))
	for range activeWorkers {
		group.Go(func() {
			for {
				select {
				case <-workCtx.Done():
					return
				case item, ok := <-jobs:
					if !ok {
						return
					}
					if workCtx.Err() != nil {
						return
					}
					value, err := fn(workCtx, values[item.index])
					if err != nil {
						errMu.Lock()
						if firstErr == nil {
							firstErr = err
						}
						errMu.Unlock()
						cancel()
						return
					}
					results[item.index] = value
				}
			}
		})
	}

dispatch:
	for index := range values {
		select {
		case <-workCtx.Done():
			break dispatch
		case jobs <- job{index: index}:
		}
	}
	close(jobs)
	group.Wait()

	errMu.Lock()
	err := firstErr
	errMu.Unlock()
	if err != nil {
		return nil, err
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	return results, nil
}
