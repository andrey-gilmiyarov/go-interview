package solution

import (
	"context"
	"fmt"

	"golang.org/x/sync/errgroup"
)

func Run(ctx context.Context, jobs []func(context.Context) error, limit int) error {
	if limit <= 0 {
		return fmt.Errorf("errgroup-batch: limit must be positive")
	}
	if err := ctx.Err(); err != nil {
		return err
	}

	group, groupCtx := errgroup.WithContext(ctx)
	group.SetLimit(limit)
	for _, job := range jobs {
		if groupCtx.Err() != nil {
			break
		}
		job := job
		group.Go(func() error {
			if groupCtx.Err() != nil {
				return nil
			}
			return job(groupCtx)
		})
	}
	if err := group.Wait(); err != nil {
		return err
	}
	return ctx.Err()
}
