package starter

import (
	"context"
	"net/http"
	"time"

	"github.com/andreygilmiyarov/go-interview/practice/url-fetcher/contract"
)

type Result = contract.Result

func FetchAll(ctx context.Context, client *http.Client, urls []string, workers int, timeout time.Duration) ([]Result, error) {
	panic("TODO")
}
