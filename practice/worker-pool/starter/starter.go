package starter

import "context"

func Map(ctx context.Context, values []int, workers int, fn func(context.Context, int) (int, error)) ([]int, error) {
	panic("TODO")
}
