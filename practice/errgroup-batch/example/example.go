package example

import (
	"context"

	"github.com/andreygilmiyarov/go-interview/practice/errgroup-batch/solution"
)

func BatchExample(ctx context.Context) error {
	jobs := []func(context.Context) error{
		func(ctx context.Context) error { return ctx.Err() },
		func(context.Context) error { return nil },
	}
	return solution.Run(ctx, jobs, 2)
}
