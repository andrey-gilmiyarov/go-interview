package concurrency

import (
	"context"

	example "github.com/andreygilmiyarov/go-interview/practice/errgroup-batch/example"
)

func BatchExample(ctx context.Context) error {
	return example.BatchExample(ctx)
}
