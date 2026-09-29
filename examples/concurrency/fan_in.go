package concurrency

import (
	"context"

	example "github.com/andreygilmiyarov/go-interview/practice/fan-in/example"
)

func FanInExample(ctx context.Context) []int {
	return example.FanInExample(ctx)
}
