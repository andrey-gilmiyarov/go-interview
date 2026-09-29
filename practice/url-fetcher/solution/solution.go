package solution

import (
	"context"
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/andreygilmiyarov/go-interview/practice/url-fetcher/contract"
)

type Result = contract.Result

func FetchAll(ctx context.Context, client *http.Client, urls []string, workers int, timeout time.Duration) ([]Result, error) {
	if client == nil {
		return nil, fmt.Errorf("url-fetcher: client is nil")
	}
	if workers <= 0 {
		return nil, fmt.Errorf("url-fetcher: workers must be positive")
	}
	if timeout <= 0 {
		return nil, fmt.Errorf("url-fetcher: timeout must be positive")
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if len(urls) == 0 {
		return []Result{}, nil
	}

	results := make([]Result, len(urls))
	jobs := make(chan int)
	var group sync.WaitGroup
	for range min(workers, len(urls)) {
		group.Go(func() {
			for {
				select {
				case <-ctx.Done():
					return
				case index, ok := <-jobs:
					if !ok {
						return
					}
					if ctx.Err() != nil {
						return
					}
					results[index] = fetchOne(ctx, client, urls[index], timeout)
				}
			}
		})
	}

dispatch:
	for index := range urls {
		select {
		case <-ctx.Done():
			break dispatch
		case jobs <- index:
		}
	}
	close(jobs)
	group.Wait()
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	return results, nil
}

func fetchOne(ctx context.Context, client *http.Client, url string, timeout time.Duration) Result {
	result := Result{URL: url}
	requestCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	request, err := http.NewRequestWithContext(requestCtx, http.MethodGet, url, nil)
	if err != nil {
		result.Err = err
		return result
	}
	response, err := client.Do(request)
	if err != nil {
		result.Err = err
		return result
	}
	result.Status = response.StatusCode
	if err := response.Body.Close(); err != nil {
		result.Err = err
	}
	return result
}
