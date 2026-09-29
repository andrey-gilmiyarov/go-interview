package starter

import (
	"context"
	"errors"
)

var ErrNoJobs = errors.New("first-success: no jobs")

func First(ctx context.Context, jobs ...func(context.Context) (string, error)) (string, error) {
	panic("TODO")
}
