package concurrency

import (
	"context"

	example "github.com/andreygilmiyarov/go-interview/practice/first-success/example"
)

func TaxiExample(ctx context.Context) (string, error) {
	return example.TaxiExample(ctx)
}
