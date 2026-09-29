package solution

import (
	"context"
	"errors"
	"sync"
)

var ErrNoJobs = errors.New("first-success: no jobs")

func First(ctx context.Context, jobs ...func(context.Context) (string, error)) (string, error) {
	if len(jobs) == 0 {
		return "", ErrNoJobs
	}
	if err := ctx.Err(); err != nil {
		return "", err
	}

	jobCtx, cancel := context.WithCancel(ctx)
	defer cancel()

	var group sync.WaitGroup
	var mu sync.Mutex
	won := false
	var winner string
	errs := make([]error, len(jobs))

	for index, job := range jobs {
		index, job := index, job
		group.Go(func() {
			if jobCtx.Err() != nil {
				return
			}
			value, err := job(jobCtx)
			if err == nil {
				mu.Lock()
				if !won && ctx.Err() == nil {
					won = true
					winner = value
					cancel()
				}
				mu.Unlock()
				return
			}
			errs[index] = err
		})
	}
	group.Wait()

	mu.Lock()
	hasWinner, value := won, winner
	mu.Unlock()
	if hasWinner {
		return value, nil
	}
	if err := ctx.Err(); err != nil {
		return "", err
	}
	return "", errors.Join(errs...)
}
