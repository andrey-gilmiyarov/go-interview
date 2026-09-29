package concurrency

import (
	"context"
	"net/http"

	example "github.com/andreygilmiyarov/go-interview/practice/url-fetcher/example"
	"github.com/andreygilmiyarov/go-interview/practice/url-fetcher/solution"
)

func FetchURLsExample(ctx context.Context, client *http.Client, urls []string) ([]solution.Result, error) {
	return example.FetchURLsExample(ctx, client, urls)
}
