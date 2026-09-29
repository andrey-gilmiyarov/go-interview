package example

import (
	"context"

	"github.com/andreygilmiyarov/go-interview/practice/worker-pool/solution"
)

func WorkerPoolExample(ctx context.Context) ([]int, error) {
	return solution.Map(ctx, []int{2, 3, 4}, 2, func(_ context.Context, value int) (int, error) {
		return value * value, nil
	})
}
