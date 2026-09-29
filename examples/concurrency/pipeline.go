package concurrency

import (
	"context"

	example "github.com/andreygilmiyarov/go-interview/practice/pipeline/example"
)

func PipelineExample(ctx context.Context) []int {
	return example.PipelineExample(ctx)
}
