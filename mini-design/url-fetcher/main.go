package main

import (
	"context"
	"fmt"
	"net/http"
    "time"
    "sync"
)

const (
    workersCount = 3
    timeout = time.Second * 2
)

type Fetcher struct {
    client *http.Client
}

type FetchResult struct {
    URL    string
    Status int
    Err    error
}

func (f Fetcher) FetchAll(ctx context.Context, urls []string, workers int) <-chan FetchResult {
    result := make(chan FetchResult, 10)
    jobs := make(chan string, 10)
    
	var wg sync.WaitGroup
    wg.Add(workers)

	sendResult := func(fr FetchResult) bool {
		select {
		case <-ctx.Done():
			return false
		case result <- fr:
			return true
		}
	}

    go func() {
        defer close(jobs)

        for _, url := range urls {
            select {
                case <-ctx.Done():
                    return
                case jobs <- url:
            }
        }
    }()

    for range workers {
        go func() {
			reqCtx, cancel := context.WithTimeout(ctx, 2*time.Second)
			defer cancel()
            defer wg.Done()

            for url := range jobs {
                req, err := http.NewRequestWithContext(reqCtx, http.MethodGet, url, nil)
                if err != nil {
					if !sendResult(FetchResult{URL: url, Status: 0, Err: err}) {
						return
					}
					continue
				}

				resp, err := f.client.Do(req)
				if err != nil {
					if !sendResult(FetchResult{URL: url, Status: 0, Err: err}) {
						return
					}
					continue
				}
				resp.Body.Close()

                ft := FetchResult{
                    URL: url,
                	Status: resp.StatusCode,
                }

                if !sendResult(ft) {
					return
				}
            }
        }()
    }

    go func() {
        wg.Wait()
        close(result)
    }()

    return result
}

func main() {
    ctx, cancel := context.WithTimeout(context.Background(), timeout)
    defer cancel()

    clientHTTP := &http.Client{
	    Timeout: timeout,
    }

    fetcher := Fetcher{
        client: clientHTTP,
    }

    // assume we have a lot of urls here
    urls := []string{
        "https://www.youtube.com/",
        "https://www.google.com/",
        "https://www.instagram.com/",
        "https://www.apple.com/",
    }

    resp := fetcher.FetchAll(ctx, urls, workersCount)

    for r := range resp {
        fmt.Printf("URL: %s, Status: %d, Err: %w\n", r.URL, r.Status, r.Err)
    } 
}
