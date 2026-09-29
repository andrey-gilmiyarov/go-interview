package concurrency

import (
	"context"

	example "github.com/andreygilmiyarov/go-interview/practice/worker-pool/example"
)

func WorkerPoolExample(ctx context.Context) ([]int, error) {
	return example.WorkerPoolExample(ctx)
}
