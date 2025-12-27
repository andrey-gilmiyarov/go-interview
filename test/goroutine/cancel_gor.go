package goroutine

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"sync"
)

func CancelGoroutinesExample() {
	ch := make(chan string)
	var wg sync.WaitGroup

	urls := []string{
		"https://www.youtube.com/",
		"https://en.wikipedia.org/",
		"https://google.com/",
		"https://ya.ru/",
		"https://habr.com/",
		"https://leetcode.com",
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	wg.Add(len(urls))
	for _, url := range urls {
		go func(ctx context.Context, u string) {
			defer wg.Done()
			fmt.Printf("Start fetching url: %s\n", u)

			select {
			case <-ctx.Done():
				fmt.Printf("Context canceled while fetching %s url\n", u)
				return
			default:
				resp, err := GetResp(ctx, u)
				if err != nil {
					fmt.Printf("Get error %s when trying to fetch %s url\n", err, u)
					cancel() // Cancel all goroutines on error
					return
				}
				ch <- fmt.Sprintf("URL %s with status %s", u, resp.Status)
				fmt.Printf("Finish fetching url: %s\n", u)
			}
		}(ctx, url)
	}

	go func() {
		wg.Wait()
		close(ch)
	}()

	for resp := range ch {
		fmt.Println(resp)
	}
}

func GetResp(ctx context.Context, url string) (*http.Response, error) {
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, err
	}

	client := &http.Client{}
	_, err = client.Do(req)
	if err != nil {
		return nil, err
	}

	//    return resp, nil
	return nil, errors.New("some error")
}
