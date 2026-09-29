package example

import (
	"context"
	"net/http"
	"time"

	"github.com/andreygilmiyarov/go-interview/practice/url-fetcher/solution"
)

func FetchURLsExample(ctx context.Context, client *http.Client, urls []string) ([]solution.Result, error) {
	return solution.FetchAll(ctx, client, urls, 3, 2*time.Second)
}
